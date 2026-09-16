package controller

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"my-ple-backend/db"
	"my-ple-backend/middleware"
	"my-ple-backend/model"
	"my-ple-backend/notification"
)

type ZoomBookingController struct {
	DB            *db.Database
	Notifications *notification.Service
}

func cleanZoomBooking(input *model.ZoomBookingInput) bool {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	return model.ValidZoomBooking(*input)
}

func (c *ZoomBookingController) Create(w http.ResponseWriter, r *http.Request) {
	current, _ := middleware.GetClaims(r.Context())
	var input model.ZoomBookingInput
	if !DecodeJSON(w, r, &input) {
		return
	}
	if !cleanZoomBooking(&input) {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Judul, waktu, atau URL meeting tidak valid."})
		return
	}
	booking, err := c.Notifications.CreateZoomBooking(r.Context(), current.ID, input)
	if errors.Is(err, db.ErrZoomBookingConflict) {
		WriteJSON(w, http.StatusConflict, map[string]string{"error": "Waktu tersebut sudah digunakan oleh booking Zoom lain."})
		return
	}
	if err != nil {
		log.Printf("create zoom booking: %v", err)
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Booking Zoom tidak dapat disimpan."})
		return
	}
	WriteJSON(w, http.StatusCreated, map[string]any{"booking": booking})
}

func (c *ZoomBookingController) List(w http.ResponseWriter, r *http.Request) {
	start, startErr := time.Parse(time.RFC3339, r.URL.Query().Get("start"))
	end, endErr := time.Parse(time.RFC3339, r.URL.Query().Get("end"))
	if startErr != nil || endErr != nil || !start.Before(end) || end.Sub(start) > 31*24*time.Hour {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Rentang waktu booking tidak valid."})
		return
	}
	bookings, err := c.DB.ListZoomBookings(r.Context(), start, end)
	if err != nil {
		log.Printf("list zoom bookings: %v", err)
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Booking Zoom tidak dapat dimuat."})
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"bookings": bookings})
}

func (c *ZoomBookingController) Update(w http.ResponseWriter, r *http.Request, id string) {
	current, _ := middleware.GetClaims(r.Context())
	var input model.ZoomBookingInput
	if !DecodeJSON(w, r, &input) {
		return
	}
	if !cleanZoomBooking(&input) {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Judul, waktu, atau URL meeting tidak valid."})
		return
	}
	booking, err := c.DB.UpdateZoomBooking(r.Context(), id, current.ID, input)
	if errors.Is(err, db.ErrZoomBookingConflict) {
		WriteJSON(w, http.StatusConflict, map[string]string{"error": "Waktu tersebut sudah digunakan oleh booking Zoom lain."})
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Booking tidak ditemukan atau tidak dapat diubah."})
		return
	}
	if err != nil {
		log.Printf("update zoom booking: %v", err)
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Booking Zoom tidak dapat diperbarui."})
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"booking": booking})
}

func (c *ZoomBookingController) Delete(w http.ResponseWriter, r *http.Request, id string) {
	current, _ := middleware.GetClaims(r.Context())
	err := c.DB.DeleteZoomBooking(r.Context(), id, current.ID)
	if errors.Is(err, sql.ErrNoRows) {
		WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Booking tidak ditemukan atau tidak dapat dibatalkan."})
		return
	}
	if err != nil {
		log.Printf("delete zoom booking: %v", err)
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Booking Zoom tidak dapat dibatalkan."})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
