package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"

	"my-ple-backend/controller"
	"my-ple-backend/db"
	"my-ple-backend/middleware"
)

func newHandler(database *db.Database, secret, corsOrigin string) http.Handler {
	authCtrl := controller.NewAuthController(database, secret)
	protected := middleware.RequireAuth(secret, protectedHandler(database, authCtrl))
	uploads := http.StripPrefix("/uploads/", http.FileServer(http.Dir(envOr("UPLOAD_DIR", "upload"))))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", corsOrigin)
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
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
		if database == nil {
			controller.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Database belum dikonfigurasi."})
			return
		}

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/roles":
			authCtrl.Roles(w, r)
			return
		case r.Method == http.MethodPost && r.URL.Path == "/auth/register":
			authCtrl.Register(w, r)
			return
		case r.Method == http.MethodPost && r.URL.Path == "/auth/login":
			authCtrl.Login(w, r)
			return
		}
		protected.ServeHTTP(w, r)
	})
}

// ponytail: unified protected routing for reports and employee profile
func protectedHandler(database *db.Database, authCtrl *controller.AuthController) http.Handler {
	weeklyCtrl := controller.NewWeeklyController(database)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.GetClaims(r.Context())
		if !ok {
			controller.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Autentikasi diperlukan."})
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/me" {
			authCtrl.Profile(w, r, current.ID)
			return
		}
		if (r.Method == http.MethodGet) && r.URL.Path == "/profile" {
			authCtrl.Profile(w, r, current.ID)
			return
		}
		if (r.Method == http.MethodPut || r.Method == http.MethodPost || r.Method == http.MethodPatch) && r.URL.Path == "/profile" {
			controller.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Profil hanya dapat diperbarui melalui Manage Employees."})
			return
		}
		if strings.EqualFold(current.RoleName, "Admin") || strings.EqualFold(current.RoleName, "Management") || strings.EqualFold(current.RoleName, "HR") {
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
				},
			})
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
		controller.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Endpoint tidak ditemukan."})
	})
}

func main() {
	loadDotEnv(".env")
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
			fmt.Printf("Warning: auto-migration error: %v\n", err)
		} else {
			fmt.Println("Database schema auto-migrated successfully.")
		}
		defer conn.Close()
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	secret := os.Getenv("AUTH_SECRET")
	if secret == "" {
		secret = "development-only-secret-change-me"
	}
	fmt.Printf("my-ple backend listening on http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, newHandler(database, secret, envOr("FRONTEND_ORIGIN", "http://localhost:5173"))); err != nil {
		panic(err)
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
