package controller

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"my-ple-backend/email"
	"my-ple-backend/middleware"
	"my-ple-backend/model"
)

func FeedbackHandler(send func(context.Context, model.Claims, string) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := middleware.GetClaims(r.Context())
		if !ok {
			WriteJSON(w, 401, map[string]string{"error": "Autentikasi diperlukan."})
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			w.WriteHeader(405)
			return
		}
		var input struct {
			Message string `json:"message"`
		}
		if !DecodeStrictJSON(w, r, &input) {
			return
		}
		input.Message = strings.TrimSpace(input.Message)
		if input.Message == "" || utf8.RuneCountInString(input.Message) > 4000 {
			WriteJSON(w, 400, map[string]string{"error": "Isi kritik atau saran antara 1 hingga 4.000 karakter."})
			return
		}
		if !email.ValidAddress(user.Email) {
			WriteJSON(w, 400, map[string]string{"error": "Email akun tidak valid. Hubungi administrator."})
			return
		}
		if err := send(r.Context(), user, input.Message); err != nil {
			status, message := 502, "Kritik dan saran gagal dikirim. Silakan coba lagi."
			if errors.Is(err, email.ErrFeedbackUnavailable) {
				status, message = 503, "Layanan email belum tersedia. Silakan coba nanti."
			}
			if errors.Is(err, email.ErrUncertain) {
				status, message = 502, "Status pengiriman belum pasti. Periksa dengan administrator sebelum mengirim ulang."
			}
			WriteJSON(w, status, map[string]string{"error": message})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
