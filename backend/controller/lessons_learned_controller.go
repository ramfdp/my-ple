package controller

import (
	"database/sql"
	"errors"
	"log"
	"my-ple-backend/db"
	"my-ple-backend/middleware"
	"my-ple-backend/model"
	"net/http"
	"strconv"
	"strings"
)

type LessonsLearnedController struct{ DB *db.Database }

func (c *LessonsLearnedController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		WriteJSON(w, 401, map[string]string{"error": "Autentikasi diperlukan."})
		return
	}
	if !model.CanUseLessonsLearned(claims.RoleName) {
		WriteJSON(w, 403, map[string]string{"error": "Akses evaluasi ditolak."})
		return
	}
	fail := func(err error) {
		status, message := 500, "Evaluasi tidak dapat diproses."
		switch {
		case errors.Is(err, model.ErrInvalidLesson):
			status, message = 400, err.Error()
		case errors.Is(err, model.ErrLessonConflict):
			status, message = 409, err.Error()
		default:
			log.Printf("lessons learned: %v", err)
		}
		WriteJSON(w, status, map[string]string{"error": message})
	}
	if r.URL.Path == "/lessons-learned" {
		switch r.Method {
		case "GET":
			jobs, err := c.DB.ListJobs(r.Context(), nil, nil, nil, nil)
			if err != nil {
				fail(err)
				return
			}
			eligible := []model.Job{}
			for _, job := range jobs {
				if job.Role == "Project Engineer" {
					eligible = append(eligible, job)
				}
			}
			lessons, err := c.DB.LessonsLearned(r.Context())
			if err != nil {
				fail(err)
				return
			}
			sections, err := db.RekapLayout()
			if err != nil {
				fail(err)
				return
			}
			WriteJSON(w, 200, map[string]any{"jobs": eligible, "lessons": lessons, "sections": sections})
		case "POST":
			var in model.LessonLearned
			if !DecodeStrictJSON(w, r, &in) {
				return
			}
			if in.ID != 0 || in.Version != 0 || in.SourceRow != 0 || in.SourceGroup != "" || in.SourceProject != "" || in.SourceNumber != "" {
				WriteJSON(w, 400, map[string]string{"error": model.ErrInvalidLesson.Error()})
				return
			}
			if !c.projectEngineerJob(w, r, in.JobID) {
				return
			}
			saved, err := c.DB.SaveLessonLearned(r.Context(), 0, in)
			if err != nil {
				fail(err)
				return
			}
			WriteJSON(w, 201, map[string]any{"lesson": saved})
		default:
			w.WriteHeader(405)
		}
		return
	}
	part := strings.TrimPrefix(r.URL.Path, "/lessons-learned/")
	id, err := strconv.Atoi(part)
	if err != nil || id <= 0 || strings.Contains(part, "/") {
		WriteJSON(w, 404, map[string]string{"error": "Evaluasi tidak ditemukan."})
		return
	}
	switch r.Method {
	case "PUT":
		var in model.LessonLearned
		if !DecodeStrictJSON(w, r, &in) {
			return
		}
		if in.ID != id || in.Version < 1 {
			WriteJSON(w, 400, map[string]string{"error": model.ErrInvalidLesson.Error()})
			return
		}
		if in.SourceRow == 0 && !c.projectEngineerJob(w, r, in.JobID) {
			return
		}
		saved, err := c.DB.SaveLessonLearned(r.Context(), id, in)
		if err != nil {
			fail(err)
			return
		}
		WriteJSON(w, 200, map[string]any{"lesson": saved})
	case "DELETE":
		version, err := strconv.Atoi(r.URL.Query().Get("version"))
		if err != nil || version < 1 {
			WriteJSON(w, 400, map[string]string{"error": "Versi evaluasi tidak valid."})
			return
		}
		if err := c.DB.DeleteLessonLearned(r.Context(), id, version); err != nil {
			fail(err)
			return
		}
		w.WriteHeader(204)
	default:
		w.WriteHeader(405)
	}
}

func (c *LessonsLearnedController) projectEngineerJob(w http.ResponseWriter, r *http.Request, id int) bool {
	job, err := c.DB.Job(r.Context(), id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Printf("lessons learned job: %v", err)
		WriteJSON(w, 500, map[string]string{"error": "Pekerjaan tidak dapat diperiksa."})
		return false
	}
	if err != nil || job.Role != "Project Engineer" {
		WriteJSON(w, 400, map[string]string{"error": "Pekerjaan Project Engineer tidak valid."})
		return false
	}
	return true
}
