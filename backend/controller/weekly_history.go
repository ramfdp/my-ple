package controller

import (
	"database/sql"
	"errors"
	"fmt"
	"my-ple-backend/middleware"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (c *WeeklyController) History(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		WriteJSON(w, 401, map[string]string{"error": "Autentikasi diperlukan."})
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	path := strings.TrimPrefix(r.URL.Path, "/reports/history")
	if path == "" {
		page := 1
		if value := r.URL.Query().Get("page"); value != "" {
			var err error
			page, err = strconv.Atoi(value)
			if err != nil || page < 1 || page > 100000 {
				WriteJSON(w, 400, map[string]string{"error": "Halaman tidak valid."})
				return
			}
		}
		division, date := strings.TrimSpace(r.URL.Query().Get("division")), r.URL.Query().Get("date")
		if own := weeklyDivision(claims.RoleName); own != "" {
			if division != "" && division != own {
				WriteJSON(w, http.StatusForbidden, map[string]string{"error": "Laporan divisi lain tidak dapat diakses."})
				return
			}
			division = own
		} else if claims.RoleName != "Admin" && claims.RoleName != "Management" {
			WriteJSON(w, http.StatusForbidden, map[string]string{"error": "Weekly report tidak tersedia untuk role ini."})
			return
		}
		if len(division) > 100 {
			WriteJSON(w, 400, map[string]string{"error": "Divisi tidak valid."})
			return
		}
		if date != "" {
			if _, err := time.Parse("2006-01-02", date); err != nil {
				WriteJSON(w, 400, map[string]string{"error": "Tanggal tidak valid."})
				return
			}
		}
		archives, err := c.DB.ArchiveHistory(r.Context(), division, date, (page-1)*20)
		if err != nil {
			WriteJSON(w, 500, map[string]string{"error": "Riwayat tidak dapat dimuat."})
			return
		}
		hasMore := len(archives) > 20
		if hasMore {
			archives = archives[:20]
		}
		WriteJSON(w, 200, map[string]any{"archives": archives, "page": page, "hasMore": hasMore})
		return
	}
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) > 2 || (len(parts) == 2 && parts[1] != "pdf") || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	a, err := c.DB.Archive(r.Context(), parts[0])
	if errors.Is(err, sql.ErrNoRows) {
		WriteJSON(w, 404, map[string]string{"error": "Arsip tidak ditemukan."})
		return
	}
	if err != nil {
		WriteJSON(w, 500, map[string]string{"error": "Arsip tidak dapat dimuat."})
		return
	}
	if !canViewWeekly(claims, a.Division) {
		WriteJSON(w, http.StatusForbidden, map[string]string{"error": "Laporan divisi lain tidak dapat diakses."})
		return
	}
	if len(parts) == 1 {
		WriteJSON(w, 200, map[string]any{"archive": a, "groups": exportGroups(a.Reports())})
		return
	}
	if a.Status != "finalized" {
		WriteJSON(w, 409, map[string]string{"error": "PDF arsip sedang diproses."})
		return
	}
	data, err := NewWeeklyLifecycle(c.DB).EnsurePDF(r.Context(), &a)
	if err != nil {
		WriteJSON(w, 503, map[string]string{"error": "PDF sementara tidak tersedia; snapshot arsip tetap aman."})
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, a.Filename))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Write(data)
}
