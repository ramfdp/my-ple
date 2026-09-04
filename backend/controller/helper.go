package controller

import (
	"encoding/json"
	"net/http"
)

func DecodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 16_384)
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON tidak valid."})
		return false
	}
	return true
}

func DecodeStrictJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 16_384)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON berisi field yang tidak diizinkan."})
		return false
	}
	return true
}

func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
