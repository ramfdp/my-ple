package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/bcrypt"
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{3,50}$`)
var workDivisions = map[string]map[string]bool{
	"Project Engineer":     {"Quality Assurance 1": true, "Quality Assurance 2": true, "Pengawasan": true, "Non-Pengawasan": true, "CIHPK": true},
	"AI Dev":               {"Pengembangan": true, "Pengujian Model": true, "Deployment": true, "Dokumentasi": true},
	"Keuangan":             {"Pembukuan": true, "Pembayaran": true, "Anggaran": true, "Pelaporan": true},
	"Business Development": {"Prospek": true, "Proposal": true, "Kemitraan": true, "Tindak Lanjut": true},
	"HR":                   {"Kehadiran": true, "Rekrutmen": true, "Administrasi": true, "Pengembangan SDM": true},
	"PMO":                  {"Milestone": true, "Rapat": true, "Risiko": true, "Pelaporan": true},
}

type database struct{ conn *sql.DB }
type user struct{ ID, Username, DisplayName, PasswordHash string }
type claims struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	ExpiresAt   int64  `json:"expiresAt"`
}
type credentials struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
}
type reportInput struct {
	PrimaryDivision string    `json:"primaryDivision"`
	WorkDivision    string    `json:"workDivision"`
	Description     string    `json:"description"`
	CompletedAt     time.Time `json:"completedAt"`
}
type report struct {
	ID              string    `json:"id"`
	PrimaryDivision string    `json:"primaryDivision"`
	WorkDivision    string    `json:"workDivision"`
	Number          int       `json:"number"`
	Description     string    `json:"description"`
	CompletedAt     time.Time `json:"completedAt"`
}

func (db *database) findUser(ctx context.Context, username string) (user, error) {
	var u user
	err := db.conn.QueryRowContext(ctx, `SELECT id, username, display_name, password_hash FROM users WHERE username = $1`, username).Scan(&u.ID, &u.Username, &u.DisplayName, &u.PasswordHash)
	return u, err
}

func (db *database) createUser(ctx context.Context, u user) (user, error) {
	err := db.conn.QueryRowContext(ctx, `INSERT INTO users (id, username, display_name, password_hash) VALUES ($1, $2, $3, $4) RETURNING id, username, display_name`, u.ID, u.Username, u.DisplayName, u.PasswordHash).Scan(&u.ID, &u.Username, &u.DisplayName)
	return u, err
}

func (db *database) createReport(ctx context.Context, userID string, input reportInput) (report, error) {
	var result report
	err := db.conn.QueryRowContext(ctx, `
		WITH next_number AS (
			INSERT INTO report_sequences (primary_division, work_division, last_number)
			VALUES ($1, $2, 1)
			ON CONFLICT (primary_division, work_division)
			DO UPDATE SET last_number = report_sequences.last_number + 1
			RETURNING last_number
		)
		INSERT INTO reports (id, user_id, primary_division, work_division, number, description, completed_at)
		SELECT $3, $4, $1, $2, last_number, $5, $6 FROM next_number
		RETURNING id, primary_division, work_division, number, description, completed_at`,
		input.PrimaryDivision, input.WorkDivision, newID(), userID, input.Description, input.CompletedAt,
	).Scan(&result.ID, &result.PrimaryDivision, &result.WorkDivision, &result.Number, &result.Description, &result.CompletedAt)
	return result, err
}

func (db *database) listReports(ctx context.Context, primaryDivision string) ([]report, error) {
	rows, err := db.conn.QueryContext(ctx, `SELECT id, primary_division, work_division, number, description, completed_at FROM reports WHERE primary_division = $1 ORDER BY work_division, number`, primaryDivision)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reports []report
	for rows.Next() {
		var item report
		if err := rows.Scan(&item.ID, &item.PrimaryDivision, &item.WorkDivision, &item.Number, &item.Description, &item.CompletedAt); err != nil {
			return nil, err
		}
		reports = append(reports, item)
	}
	return reports, rows.Err()
}

func newHandler(db *database, secret, corsOrigin string) http.Handler {
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
			if db != nil {
				status = "configured"
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "database": status})
			return
		}
		if db == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Database belum dikonfigurasi."})
			return
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/auth/register":
			var input credentials
			if !decodeJSON(w, r, &input) {
				return
			}
			input.DisplayName = strings.TrimSpace(input.DisplayName)
			if !usernamePattern.MatchString(input.Username) || len(input.DisplayName) < 2 || len(input.DisplayName) > 100 || len(input.Password) < 8 {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Username, nama, atau password tidak valid."})
				return
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": "Terjadi kesalahan pada server."})
				return
			}
			u, err := db.createUser(r.Context(), user{ID: newID(), Username: input.Username, DisplayName: input.DisplayName, PasswordHash: string(hash)})
			if uniqueViolation(err) {
				writeJSON(w, http.StatusConflict, map[string]string{"error": "Username sudah digunakan."})
				return
			}
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": "Terjadi kesalahan pada server."})
				return
			}
			writeJSON(w, http.StatusCreated, map[string]any{"user": publicUser(u)})
			return
		case r.Method == http.MethodPost && r.URL.Path == "/auth/login":
			var input credentials
			if !decodeJSON(w, r, &input) {
				return
			}
			u, err := db.findUser(r.Context(), input.Username)
			if err != nil || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(input.Password)) != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Username atau password salah."})
				return
			}
			profile := claims{ID: u.ID, Username: u.Username, DisplayName: u.DisplayName, ExpiresAt: time.Now().Add(8 * time.Hour).UnixMilli()}
			writeJSON(w, http.StatusOK, map[string]any{"token": signToken(profile, secret), "user": publicUser(u)})
			return
		}

		current, ok := authenticated(r, secret)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Autentikasi diperlukan."})
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/me" {
			writeJSON(w, http.StatusOK, map[string]any{"user": current})
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/dashboard" {
			writeJSON(w, http.StatusOK, map[string]any{"greeting": "Halo, " + current.DisplayName, "menus": []map[string]string{{"id": "attendance", "name": "Absensi"}, {"id": "weekly-progress", "name": "Weekly Progress"}, {"id": "overtime", "name": "List Lembur"}, {"id": "zoom-booking", "name": "Booking Zoom"}}})
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/reports" {
			var input reportInput
			if !decodeJSON(w, r, &input) {
				return
			}
			input.PrimaryDivision = strings.TrimSpace(input.PrimaryDivision)
			input.WorkDivision = strings.TrimSpace(input.WorkDivision)
			input.Description = strings.TrimSpace(input.Description)
			if !workDivisions[input.PrimaryDivision][input.WorkDivision] || input.Description == "" || len(input.Description) > 500 || input.CompletedAt.IsZero() {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Data laporan tidak valid."})
				return
			}
			report, err := db.createReport(r.Context(), current.ID, input)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Laporan tidak dapat disimpan."})
				return
			}
			writeJSON(w, http.StatusCreated, map[string]any{"report": report})
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/reports" {
			division := strings.TrimSpace(r.URL.Query().Get("primaryDivision"))
			if division == "" || len(division) > 100 {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Divisi utama wajib diisi."})
				return
			}
			reports, err := db.listReports(r.Context(), division)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Laporan tidak dapat dimuat."})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"reports": reports})
			return
		}
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Endpoint tidak ditemukan."})
	})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 16_384)
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON tidak valid."})
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
func publicUser(u user) map[string]string {
	return map[string]string{"id": u.ID, "username": u.Username, "displayName": u.DisplayName}
}
func uniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func signToken(payload claims, secret string) string {
	body, _ := json.Marshal(payload)
	encoded := base64.RawURLEncoding.EncodeToString(body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(encoded))
	return encoded + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func authenticated(r *http.Request, secret string) (claims, bool) {
	parts := strings.Split(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), ".")
	if len(parts) != 2 {
		return claims{}, false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(parts[0]))
	expected := mac.Sum(nil)
	actual, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(expected, actual) {
		return claims{}, false
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[0])
	var result claims
	if err != nil || json.Unmarshal(body, &result) != nil || result.ExpiresAt <= time.Now().UnixMilli() {
		return claims{}, false
	}
	return result, true
}
func newID() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return fmt.Sprintf("%x-%x-%x-%x-%x", bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16])
}

func main() {
	var db *database
	if url := os.Getenv("DATABASE_URL"); url != "" {
		conn, err := sql.Open("pgx", url)
		if err != nil {
			panic(err)
		}
		if err := conn.Ping(); err != nil {
			panic(err)
		}
		db = &database{conn: conn}
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
	if err := http.ListenAndServe(":"+port, newHandler(db, secret, envOr("FRONTEND_ORIGIN", "http://localhost:5173"))); err != nil {
		panic(err)
	}
}
func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
