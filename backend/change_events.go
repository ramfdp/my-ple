package main

import (
	"my-ple-backend/middleware"
	"my-ple-backend/notification"
	"net/http"
	"strings"
)

type mutationResponse struct {
	http.ResponseWriter
	status int
}

func (w *mutationResponse) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
	w.ResponseWriter.WriteHeader(status)
}
func (w *mutationResponse) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(200)
	}
	return w.ResponseWriter.Write(p)
}

func changeEvents(next http.Handler, s *notification.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := &mutationResponse{ResponseWriter: w}
		next.ServeHTTP(response, r)
		if s == nil || response.status < 200 || response.status >= 300 || r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
			return
		}
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		event := map[string]string{"reports": "reports.changed", "zoom-bookings": "zoom.changed", "zoom-resources": "zoom.changed", "jobs": "jobs.changed", "users": "users.changed", "profile": "users.changed", "attendance": "attendance.changed", "document-checklists": "document-checklists.changed", "notifications": "notifications.changed"}[parts[0]]
		if event != "" {
			id := ""
			if len(parts) > 1 {
				id = parts[1]
			}
			if parts[0] == "profile" || parts[0] == "attendance" || parts[0] == "notifications" {
				current, ok := middleware.GetClaims(r.Context())
				if !ok {
					return
				}
				id = current.ID
			}
			s.Changed(r.Context(), event, 0, id, 0)
		}
	})
}
