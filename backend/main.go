package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"my-ple-backend/controller"
	"my-ple-backend/db"
	"my-ple-backend/middleware"
	"my-ple-backend/notification"
	"my-ple-backend/realtime"
)

func newHandler(database *db.Database, corsOrigin string) http.Handler {
	sessionTTL, err := time.ParseDuration(envOr("SESSION_TTL", "8h"))
	if err != nil || sessionTTL <= 0 {
		sessionTTL = 8 * time.Hour
	}
	secureCookie := strings.EqualFold(os.Getenv("COOKIE_SECURE"), "true") || strings.EqualFold(os.Getenv("APP_ENV"), "production")
	authCtrl := controller.NewAuthController(database, sessionTTL, secureCookie)
	hub := realtime.NewHub(corsOrigin)
	authCtrl.OnLogout = hub.CloseUser
	notifications := &notification.Service{DB: database, Hub: hub}
	protected := middleware.RequireAuth(database, protectedHandler(database, authCtrl, notifications))
	uploads := http.StripPrefix("/uploads/", http.FileServer(http.Dir(envOr("UPLOAD_DIR", "upload"))))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Origin", corsOrigin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Header.Get("Origin") != "" && r.Header.Get("Origin") != corsOrigin {
			controller.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "Origin tidak diizinkan."})
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/health" {
			status := "not configured"
			if database != nil {
				status = "configured"
			}
			controller.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "database": status})
			return
		}
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/uploads/") {
			uploads.ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/ws/notifications" {
			hub.ServeHTTP(database, w, r)
			return
		}
		if database == nil {
			controller.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Database belum dikonfigurasi."})
			return
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/auth/login":
			authCtrl.Login(w, r)
			return
		}
		protected.ServeHTTP(w, r)
	})
}

// ponytail: unified protected routing for reports and employee profile
func protectedHandler(database *db.Database, authCtrl *controller.AuthController, services ...*notification.Service) http.Handler {
	weeklyCtrl := controller.NewWeeklyController(database)
	jobCtrl := &controller.JobController{DB: database}
	var notificationService *notification.Service
	if len(services) > 0 {
		notificationService = services[0]
	}
	zoomCtrl := controller.NewZoomBookingController(database, notificationService)
	zoomResourceCtrl := controller.NewZoomResourceController(database)
	notificationCtrl := &controller.NotificationController{DB: database}
	leaveCtrl := &controller.LeaveController{DB: database, Notifications: notificationService}
	workspaceCtrl := &controller.WorkspaceController{DB: database, Notifications: notificationService}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.GetClaims(r.Context())
		if !ok {
			controller.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Autentikasi diperlukan."})
			return
		}
		if r.URL.Path == "/leave-requests" || strings.HasPrefix(r.URL.Path, "/leave-requests/") {
			leaveCtrl.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == "/jobs" || strings.HasPrefix(r.URL.Path, "/jobs/") {
			jobCtrl.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == "/workspaces" || strings.HasPrefix(r.URL.Path, "/workspaces/") || strings.HasPrefix(r.URL.Path, "/work-items/") || strings.HasPrefix(r.URL.Path, "/workspace-pages/") {
			workspaceCtrl.ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodGet && (r.URL.Path == "/me" || r.URL.Path == "/auth/me" || r.URL.Path == "/api/auth/me") {
			authCtrl.Profile(w, r, current.ID)
			return
		}
		if r.Method == http.MethodPost && (r.URL.Path == "/auth/logout" || r.URL.Path == "/api/auth/logout") {
			authCtrl.Logout(w, r)
			return
		}
		if (r.Method == http.MethodGet) && r.URL.Path == "/profile" {
			authCtrl.Profile(w, r, current.ID)
			return
		}
		if (r.Method == http.MethodPut || r.Method == http.MethodPost || r.Method == http.MethodPatch) && r.URL.Path == "/profile" {
			authCtrl.UpdateProfile(w, r, current.ID)
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/profile/avatar" {
			authCtrl.UploadAvatar(w, r, current.ID)
			return
		}
		if strings.EqualFold(current.RoleName, "Admin") || strings.EqualFold(current.RoleName, "Management") || strings.EqualFold(current.RoleName, "HR") {
			if r.Method == http.MethodGet && r.URL.Path == "/roles" {
				authCtrl.Roles(w, r)
				return
			}
			if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/users/") && strings.HasSuffix(r.URL.Path, "/avatar") {
				authCtrl.UploadAvatar(w, r, strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/users/"), "/avatar"))
				return
			}
			if r.URL.Path == "/users" {
				if r.Method == http.MethodGet {
					authCtrl.ListUsers(w, r)
					return
				}
				if r.Method == http.MethodPost {
					authCtrl.Register(w, r)
					return
				}
			}
			if strings.HasPrefix(r.URL.Path, "/users/") {
				id := strings.TrimPrefix(r.URL.Path, "/users/")
				if id == "" {
					controller.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Endpoint tidak ditemukan."})
					return
				}
				if r.Method == http.MethodPatch || r.Method == http.MethodPut {
					authCtrl.UpdateUser(w, r, id)
					return
				}
				if r.Method == http.MethodDelete {
					if id == current.ID {
						controller.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Anda tidak dapat menghapus akun sendiri."})
						return
					}
					authCtrl.DeleteUser(w, r, id)
					return
				}
			}
		}
		if r.URL.Path == "/roles" || r.URL.Path == "/users" || strings.HasPrefix(r.URL.Path, "/users/") {
			controller.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "Akses ditolak."})
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/dashboard" {
			name := current.Name
			if name == "" {
				name = current.Username
			}
			controller.WriteJSON(w, http.StatusOK, map[string]any{
				"greeting": "Halo, " + name,
				"menus": []map[string]string{
					{"id": "attendance", "name": "Absensi"},
					{"id": "weekly-progress", "name": "Weekly Progress"},
					{"id": "overtime", "name": "List Lembur"},
					{"id": "zoom-booking", "name": "Booking Zoom"},
					{"id": "workspace", "name": "Team Workspace"},
				},
			})
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/notifications" {
			notificationCtrl.List(w, r)
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/notifications/unread-count" {
			notificationCtrl.UnreadCount(w, r)
			return
		}
		if r.Method == http.MethodPatch && r.URL.Path == "/notifications/read-all" {
			notificationCtrl.MarkAllRead(w, r)
			return
		}
		if r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/notifications/") {
			notificationCtrl.Update(w, r)
			return
		}
		if r.URL.Path == "/zoom-bookings" {
			if r.Method == http.MethodGet {
				zoomCtrl.List(w, r)
				return
			}
			if r.Method == http.MethodPost {
				zoomCtrl.Create(w, r)
				return
			}
		}
		if r.Method == http.MethodGet && r.URL.Path == "/zoom-resources" {
			zoomResourceCtrl.List(w, r)
			return
		}
		if (r.Method == http.MethodPut || r.Method == http.MethodPatch) && strings.HasPrefix(r.URL.Path, "/zoom-resources/") {
			id, err := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/zoom-resources/"))
			if err == nil {
				zoomResourceCtrl.Update(w, r, id)
				return
			}
		}
		if strings.HasPrefix(r.URL.Path, "/zoom-bookings/") {
			id := strings.TrimPrefix(r.URL.Path, "/zoom-bookings/")
			if id != "" && !strings.Contains(id, "/") {
				if r.Method == http.MethodPut || r.Method == http.MethodPatch {
					zoomCtrl.Update(w, r, id)
					return
				}
				if r.Method == http.MethodDelete {
					zoomCtrl.Delete(w, r, id)
					return
				}
			}
		}
		if r.Method == http.MethodGet && (r.URL.Path == "/reports/history" || strings.HasPrefix(r.URL.Path, "/reports/history/")) {
			weeklyCtrl.History(w, r)
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/reports" {
			weeklyCtrl.Create(w, r)
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/reports" {
			weeklyCtrl.List(w, r)
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/reports.pdf" {
			weeklyCtrl.ExportPDF(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/reports/") {
			id := strings.TrimPrefix(r.URL.Path, "/reports/")
			if id == "" || strings.Contains(id, "/") {
				controller.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Endpoint tidak ditemukan."})
				return
			}
			if r.Method == http.MethodPut || r.Method == http.MethodPatch {
				weeklyCtrl.Update(w, r, id)
				return
			}
			if r.Method == http.MethodDelete {
				weeklyCtrl.Delete(w, r, id)
				return
			}
		}
		controller.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Endpoint tidak ditemukan."})
	})
}

func main() {
	loadDotEnv(".env")
	if len(os.Args) == 2 && os.Args[1] == "email-audit" {
		if err := runEmailAudit(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	var database *db.Database
	if url := os.Getenv("DATABASE_URL"); url != "" {
		conn, err := sql.Open("pgx", url)
		if err != nil {
			panic(err)
		}
		if err := conn.Ping(); err != nil {
			panic(err)
		}
		database = db.NewDatabase(conn)
		if err := database.Migrate(context.Background()); err != nil {
			panic(fmt.Errorf("database migration failed: %w", err))
		} else {
			fmt.Println("Database schema auto-migrated successfully.")
		}
		stopCleanup := startCleanup(database)
		defer conn.Close()
		defer stopCleanup()
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	fmt.Printf("my-ple backend listening on http://localhost:%s\n", port)
	server := &http.Server{Addr: ":" + port, Handler: newHandler(database, envOr("FRONTEND_ORIGIN", "http://localhost:5173"))}
	shutdown, stopSignal := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignal()
	go func() {
		<-shutdown.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}

func startCleanup(database *db.Database) func() {
	ctx, cancel := context.WithCancel(context.Background())
	workerDone := make(chan struct{})
	go func() { defer close(workerDone); (&notification.ZoomEmailWorker{DB: database}).Start(ctx) }()
	go func() {
		timer := time.NewTimer(time.Until(time.Now().Truncate(time.Minute).Add(time.Minute)))
		defer timer.Stop()
		weekly := controller.NewWeeklyLifecycle(database)
		runWeekly := func() {
			if err := weekly.Run(ctx, time.Now()); err != nil {
				fmt.Printf("Weekly lifecycle error: %v\n", err)
			}
		}
		runWeekly() // Catch up all completed periods after downtime.
		lastCleanup := time.Time{}
		for {
			select {
			case <-timer.C:
				runWeekly()
				timer.Reset(time.Until(time.Now().Truncate(time.Minute).Add(time.Minute)))
				if time.Since(lastCleanup) < time.Hour {
					continue
				}
				lastCleanup = time.Now()
				if _, err := database.DeleteExpiredSessions(ctx); err != nil {
					fmt.Printf("Warning: session cleanup error: %v\n", err)
				}
				if deleted, err := database.DeleteExpiredNotifications(ctx); err != nil {
					fmt.Printf("Warning: notification cleanup error: %v\n", err)
				} else if deleted > 0 {
					fmt.Printf("Deleted %d expired notifications.\n", deleted)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return func() { cancel(); <-workerDone }
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
