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

type BillingDocumentsController struct{ DB *db.Database }

func (c *BillingDocumentsController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		WriteJSON(w, 401, map[string]string{"error": "Autentikasi diperlukan."})
		return
	}
	if !model.CanUseBilling(claims.RoleName) {
		WriteJSON(w, 403, map[string]string{"error": "Akses penagihan ditolak."})
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	fail := func(err error) {
		status, message := 500, "Dokumen penagihan tidak dapat diproses."
		switch {
		case errors.Is(err, model.ErrBillingInvalid):
			status, message = 400, err.Error()
		case errors.Is(err, model.ErrBillingConflict):
			status, message = 409, err.Error()
		case errors.Is(err, sql.ErrNoRows):
			status, message = 404, "Data penagihan tidak ditemukan."
		}
		if status == 500 {
			log.Printf("billing documents: %v", err)
		}
		WriteJSON(w, status, map[string]string{"error": message})
	}
	id := 0
	var err error
	if r.URL.Path == "/billing-documents" {
		if r.Method != "GET" {
			w.WriteHeader(405)
			return
		}
	} else {
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/billing-documents/"), "/")
		if len(parts) < 1 || parts[0] != "stages" || len(parts) > 3 {
			http.NotFound(w, r)
			return
		}
		if len(parts) > 1 {
			id, err = strconv.Atoi(parts[1])
			if err != nil || id <= 0 {
				http.NotFound(w, r)
				return
			}
		}
		switch {
		case len(parts) == 1 && r.Method == "POST", len(parts) == 3 && parts[2] == "configuration" && r.Method == "PUT":
			var in model.BillingConfiguration
			if !DecodeStrictJSON(w, r, &in) {
				return
			}
			id, err = c.DB.ConfigureBilling(r.Context(), id, in)
		case len(parts) == 2 && r.Method == "PATCH":
			var in model.BillingPatch
			if !DecodeStrictJSON(w, r, &in) {
				return
			}
			err = c.DB.PatchBilling(r.Context(), id, in)
		default:
			w.WriteHeader(405)
			return
		}
		if err != nil {
			fail(err)
			return
		}
	}
	stages, err := c.DB.BillingStages(r.Context())
	if err != nil {
		fail(err)
		return
	}
	if id > 0 {
		for _, s := range stages {
			if s.ID == id {
				WriteJSON(w, 200, map[string]any{"stage": s})
				return
			}
		}
		fail(sql.ErrNoRows)
		return
	}
	jobs, err := c.DB.ListJobs(r.Context(), nil, nil, nil, nil)
	if err != nil {
		fail(err)
		return
	}
	included := map[int]bool{}
	for _, s := range stages {
		included[s.JobID] = true
	}
	eligible := []model.Job{}
	for _, j := range jobs {
		if j.IsActive || included[j.ID] {
			eligible = append(eligible, j)
		}
	}
	WriteJSON(w, 200, map[string]any{"jobs": eligible, "stages": stages})
}
