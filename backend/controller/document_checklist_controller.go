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
		WriteJSON(w, 200, map[string]any{"jobs": jobs})
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/document-checklists/"), "/")
	jobID, err := strconv.Atoi(parts[0])
	if err != nil || jobID <= 0 || len(parts) > 3 {
		fail(sql.ErrNoRows)
		return
	}
	if _, err = c.DB.Job(r.Context(), jobID); err != nil {
		fail(err)
		return
	}
	if len(parts) == 1 {
		switch r.Method {
		case "GET":
		case "POST", "PUT":
			var in model.ChecklistConfigInput
			if !DecodeStrictJSON(w, r, &in) {
				return
			}
			if (r.Method == "POST" && in.Version != 0) || (r.Method == "PUT" && in.Version < 1) {
				fail(model.ErrInvalidChecklist)
				return
			}
			if err = c.DB.SaveDocumentChecklist(r.Context(), jobID, in); err != nil {
				fail(err)
				return
			}
		default:
			w.WriteHeader(405)
			return
		}
	} else if parts[1] == "items" {
		itemID := 0
		if len(parts) == 3 {
			itemID, err = strconv.Atoi(parts[2])
			if err != nil || itemID <= 0 {
				fail(sql.ErrNoRows)
				return
			}
		}
		if !((len(parts) == 2 && r.Method == "POST") || (len(parts) == 3 && (r.Method == "PATCH" || r.Method == "DELETE"))) {
			w.WriteHeader(405)
			return
		}
		var in model.ChecklistItemPatch
		if r.Method != "DELETE" && !DecodeStrictJSON(w, r, &in) {
			return
		}
		if err = c.DB.MutateChecklistItem(r.Context(), jobID, itemID, r.Method, in); err != nil {
			fail(err)
			return
		}
	} else {
		fail(sql.ErrNoRows)
		return
	}
	checklist, err := c.DB.DocumentChecklist(r.Context(), jobID)
	if err != nil {
		fail(err)
		return
	}
	WriteJSON(w, 200, map[string]any{"checklist": checklist})
}
