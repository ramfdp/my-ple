package controller

import (
	"database/sql"
	"errors"
	"my-ple-backend/db"
	"my-ple-backend/middleware"
	"my-ple-backend/model"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type AttendanceController struct {
	DB     *db.Database
	Office db.AttendanceOffice
}

func NewAttendanceController(database *db.Database) *AttendanceController {
	lat, latErr := strconv.ParseFloat(os.Getenv("ATTENDANCE_OFFICE_LATITUDE"), 64)
	lng, lngErr := strconv.ParseFloat(os.Getenv("ATTENDANCE_OFFICE_LONGITUDE"), 64)
	return &AttendanceController{DB: database, Office: db.AttendanceOffice{Latitude: lat, Longitude: lng, Configured: latErr == nil && lngErr == nil}}
}

func attendanceError(w http.ResponseWriter, err error) {
	status, message := http.StatusInternalServerError, "Gagal memproses absensi. Coba muat ulang."
	switch {
	case errors.Is(err, db.ErrAttendanceNotes), errors.Is(err, db.ErrAttendanceInvalid), errors.Is(err, db.ErrAttendanceAccuracy), errors.Is(err, db.ErrAttendanceDate):
		status, message = 400, err.Error()
	case errors.Is(err, db.ErrAttendanceOutside):
		status, message = 403, err.Error()
	case errors.Is(err, db.ErrAttendanceOffice):
		status, message = 503, err.Error()
	case errors.Is(err, db.ErrAttendanceOpen):
		status, message = 409, err.Error()
	case errors.Is(err, sql.ErrNoRows):
		status, message = 404, "Absensi tidak ditemukan."
	}
	WriteJSON(w, status, map[string]string{"error": message})
}

func (c *AttendanceController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetClaims(r.Context())
	if !ok {
		WriteJSON(w, 401, map[string]string{"error": "Autentikasi diperlukan."})
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	path := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(path) == 2 && path[1] == "overview" && r.Method == http.MethodGet {
		if !strings.EqualFold(user.RoleName, "Admin") && !strings.EqualFold(user.RoleName, "Management") && !strings.EqualFold(user.RoleName, "HR") {
			WriteJSON(w, http.StatusForbidden, map[string]string{"error": "Akses rekap absensi hanya untuk HR, Management, dan Admin."})
			return
		}
		result, err := c.DB.AttendanceOverview(r.Context(), r.URL.Query().Get("date"))
		if err != nil {
			attendanceError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, result)
		return
	}
	if len(path) == 1 && r.Method == http.MethodGet {
		result, err := c.DB.AttendanceHistory(r.Context(), user.ID, r.URL.Query().Get("month"))
		if err != nil {
			attendanceError(w, err)
			return
		}
		result.OfficeConfigured = c.Office.Ready()
		WriteJSON(w, 200, result)
		return
	}
	var item model.Attendance
	var err error
	if len(path) == 2 && path[1] == "clock-in" && r.Method == http.MethodPost {
		var in model.AttendanceInput
		if !DecodeStrictJSON(w, r, &in) {
			return
		}
		item, err = c.DB.ClockIn(r.Context(), user.ID, in, c.Office)
	} else if len(path) == 3 && path[2] == "clock-out" && r.Method == http.MethodPost {
		item, err = c.DB.ClockOut(r.Context(), user.ID, path[1])
	} else {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		attendanceError(w, err)
		return
	}
	WriteJSON(w, 200, map[string]any{"attendance": item})
}
