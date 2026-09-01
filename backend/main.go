package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"

	"my-ple-backend/controller"
	"my-ple-backend/db"
	"my-ple-backend/middleware"
)

func newHandler(database *db.Database, secret, corsOrigin string) http.Handler {
	authCtrl := controller.NewAuthController(database, secret)
	protected := middleware.RequireAuth(secret, protectedHandler(database))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", corsOrigin)
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
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
		if database == nil {
			controller.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Database belum dikonfigurasi."})
			return
		}

		switch {
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

func protectedHandler(database *db.Database) http.Handler {
	weeklyCtrl := controller.NewWeeklyController(database)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.GetClaims(r.Context())
		if !ok {
			controller.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Autentikasi diperlukan."})
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/me" {
			controller.WriteJSON(w, http.StatusOK, map[string]any{"user": current})
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/dashboard" {
			controller.WriteJSON(w, http.StatusOK, map[string]any{
				"greeting": "Halo, " + current.DisplayName,
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
