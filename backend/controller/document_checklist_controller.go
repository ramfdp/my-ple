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

type DocumentChecklistController struct{ DB *db.Database }

func (c *DocumentChecklistController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		WriteJSON(w, 401, map[string]string{"error": "Autentikasi diperlukan."})
		return
	}
	if !model.CanUseDocumentChecklist(claims.RoleName) {
		WriteJSON(w, 403, map[string]string{"error": "Akses checklist ditolak."})
		return
	}
	fail := func(err error) {
		status, message := 500, "Checklist tidak dapat diproses."
		switch {
		case errors.Is(err, sql.ErrNoRows):
			status, message = 404, "Pekerjaan, dokumen, atau penandatangan tidak ditemukan."
		case errors.Is(err, model.ErrInvalidChecklist):
			status, message = 400, err.Error()
		case errors.Is(err, model.ErrChecklistConflict):
			status, message = 409, err.Error()
		}
		if status == 500 {
			log.Printf("document checklist: %v", err)
		}
		WriteJSON(w, status, map[string]string{"error": message})
	}
	if r.URL.Path == "/document-checklists" {
		if r.Method != "GET" {
			w.WriteHeader(405)
			return
		}
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
		WriteJSON(w, 200, map[string]any{"jobs": eligible})
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/document-checklists/"), "/")
	jobID, err := strconv.Atoi(parts[0])
	if err != nil || jobID <= 0 {
		fail(sql.ErrNoRows)
		return
	}
	job, err := c.DB.Job(r.Context(), jobID)
	if err != nil {
		fail(err)
		return
	}
	if job.Role != "Project Engineer" {
		fail(sql.ErrNoRows)
		return
	}
	tableID := 0
	if len(parts) != 1 {
		if len(parts) < 3 || len(parts) > 5 || parts[1] != "tables" {
			fail(sql.ErrNoRows)
			return
		}
		tableID, err = strconv.Atoi(parts[2])
		if err != nil || tableID <= 0 {
			fail(sql.ErrNoRows)
			return
		}
	}
	if len(parts) == 1 && r.Method == "GET" {
		checklists, err := c.DB.DocumentChecklists(r.Context(), jobID)
		if err != nil {
			fail(err)
			return
		}
		WriteJSON(w, 200, map[string]any{"checklists": checklists})
		return
	}
	if len(parts) == 1 || len(parts) == 3 {
		switch {
		case len(parts) == 3 && r.Method == "GET":
		case len(parts) == 1 && r.Method == "POST", len(parts) == 3 && r.Method == "PUT":
			var in model.ChecklistConfigInput
			if !DecodeStrictJSON(w, r, &in) {
				return
			}
			if (r.Method == "POST" && in.Version != 0) || (r.Method == "PUT" && in.Version < 1) {
				fail(model.ErrInvalidChecklist)
				return
			}
			tableID, err = c.DB.SaveDocumentChecklist(r.Context(), jobID, tableID, in)
			if err != nil {
				fail(err)
				return
			}
		default:
			w.WriteHeader(405)
			return
		}
	} else if parts[3] == "items" {
		itemID := 0
		if len(parts) == 5 {
			itemID, err = strconv.Atoi(parts[4])
			if err != nil || itemID <= 0 {
				fail(sql.ErrNoRows)
				return
			}
		}
		if !((len(parts) == 4 && r.Method == "POST") || (len(parts) == 5 && (r.Method == "PATCH" || r.Method == "DELETE"))) {
			w.WriteHeader(405)
			return
		}
		var in model.ChecklistItemPatch
		if r.Method != "DELETE" && !DecodeStrictJSON(w, r, &in) {
			return
		}
		if err = c.DB.MutateChecklistItem(r.Context(), jobID, tableID, itemID, r.Method, in); err != nil {
			fail(err)
			return
		}
	} else {
		fail(sql.ErrNoRows)
		return
	}
	checklist, err := c.DB.DocumentChecklist(r.Context(), jobID, tableID)
	if err != nil {
		fail(err)
		return
	}
	WriteJSON(w, 200, map[string]any{"checklist": checklist})
}
