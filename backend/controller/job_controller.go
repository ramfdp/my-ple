package controller

import (
	"database/sql"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"log"
	"my-ple-backend/db"
	"my-ple-backend/middleware"
	"my-ple-backend/model"
	"net/http"
	"strconv"
	"strings"
)

type JobController struct{ DB *db.Database }

func (c *JobController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		WriteJSON(w, 401, map[string]string{"error": "Autentikasi diperlukan."})
		return
	}
	fail := func(err error) {
		status, message := 500, "Pekerjaan tidak dapat diproses."
		var pg *pgconn.PgError
		switch {
		case errors.Is(err, sql.ErrNoRows):
			status, message = 404, "Pekerjaan tidak ditemukan."
		case errors.Is(err, db.ErrInvalidReport):
			status, message = 400, "Nama, urutan, atau konteks pekerjaan tidak valid."
		case errors.As(err, &pg):
			switch pg.Code {
			case "23505":
				status, message = 409, "Nama pekerjaan sudah ada pada konteks ini."
			case "23503":
				status, message = 409, "Relasi tidak valid atau pekerjaan sudah digunakan. Nonaktifkan pekerjaan sebagai gantinya."
			case "23514":
				status, message = 400, "Konteks tidak valid. Pekerjaan yang sudah digunakan tidak dapat dipindahkan."
			case "22003", "22P02":
				status, message = 400, "ID pekerjaan atau relasi tidak valid."
			}
		}
		if status == 500 {
			log.Printf("jobs: %v", err)
		}
		WriteJSON(w, status, map[string]string{"error": message})
	}
	if r.URL.Path == "/jobs/masters" && r.Method == http.MethodGet {
		result, err := c.DB.JobMasters(r.Context())
		if err != nil {
			fail(err)
			return
		}
		WriteJSON(w, 200, result)
		return
	}
	if r.URL.Path == "/jobs" {
		switch r.Method {
		case http.MethodGet:
			filters := make([]*int, 3)
			for i, key := range []string{"division_id", "role_id", "job_category_id"} {
				if raw := r.URL.Query().Get(key); raw != "" {
					v, err := strconv.Atoi(raw)
					if err != nil || v < 0 {
						fail(db.ErrInvalidReport)
						return
					}
					filters[i] = &v
				}
			}
			var active *bool
			if raw := r.URL.Query().Get("active"); raw != "" {
				v, err := strconv.ParseBool(raw)
				if err != nil {
					fail(db.ErrInvalidReport)
					return
				}
				active = &v
			}
			jobs, err := c.DB.ListJobs(r.Context(), filters[0], filters[1], filters[2], active)
			if err != nil {
				fail(err)
				return
			}
			WriteJSON(w, 200, map[string]any{"jobs": jobs})
			return
		case http.MethodPost:
			input := model.Job{IsActive: true}
			if !DecodeJSON(w, r, &input) {
				return
			}
			j, err := c.DB.SaveJob(r.Context(), 0, input, claims.ID)
			if err != nil {
				fail(err)
				return
			}
			WriteJSON(w, 201, map[string]any{"job": j})
			return
		}
	} else {
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/jobs/"), "/")
		id, err := strconv.Atoi(parts[0])
		if err != nil || id <= 0 || len(parts) > 2 {
			fail(sql.ErrNoRows)
			return
		}
		var job model.Job
		if len(parts) == 2 {
			if parts[1] != "status" || r.Method != http.MethodPatch {
				w.WriteHeader(405)
				return
			}
			var input struct {
				IsActive *bool `json:"isActive"`
			}
			if !DecodeJSON(w, r, &input) {
				return
			}
			if input.IsActive == nil {
				fail(db.ErrInvalidReport)
				return
			}
			job, err = c.DB.SetJobStatus(r.Context(), id, *input.IsActive)
		} else {
			switch r.Method {
			case http.MethodGet:
				job, err = c.DB.Job(r.Context(), id)
			case http.MethodPut:
				var input model.Job
				if !DecodeJSON(w, r, &input) {
					return
				}
				job, err = c.DB.SaveJob(r.Context(), id, input, claims.ID)
			case http.MethodDelete:
				err = c.DB.DeleteJob(r.Context(), id)
				if err != nil {
					fail(err)
					return
				}
				w.WriteHeader(204)
				return
			default:
				w.WriteHeader(405)
				return
			}
		}
		if err != nil {
			fail(err)
			return
		}
		WriteJSON(w, 200, map[string]any{"job": job})
		return
	}
	w.WriteHeader(405)
}
