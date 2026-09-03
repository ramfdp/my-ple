package controller

import (
	"bytes"
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"my-ple-backend/db"
	"my-ple-backend/middleware"
	"my-ple-backend/model"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestZoomResourceRBACPostgres(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is required")
	}
	conn, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	database := db.NewDatabase(conn)
	if err := database.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	resources, err := database.ListZoomResources(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	original := model.ZoomResourceUpdateInput{Description: resources[0].Description, MeetingURL: resources[0].MeetingURL, MeetingID: resources[0].MeetingID, HostKey: resources[0].HostKey}
	t.Cleanup(func() { _, _ = database.UpdateZoomResource(context.Background(), 1, original) })
	ctrl := NewZoomResourceController(database)
	secret := "test-secret"

	for _, role := range []string{"Admin", "Management"} {
		handler := middleware.RequireAuth(secret, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { ctrl.Update(w, r, 1) }))
		token := middleware.SignToken(model.Claims{ID: "user", RoleName: role, ExpiresAt: time.Now().Add(time.Hour).UnixMilli()}, secret)
		request := httptest.NewRequest(http.MethodPut, "/zoom-resources/1", bytes.NewBufferString(`{"description":"updated by test","meetingUrl":"","meetingId":"","hostKey":"654321"}`))
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s update status = %d: %s", role, response.Code, response.Body.String())
		}
		resources, err = database.ListZoomResources(context.Background())
		if err != nil || resources[0].HostKey != "654321" {
			t.Fatalf("%s host key was not persisted: %#v, %v", role, resources[0], err)
		}
	}

	handler := middleware.RequireAuth(secret, http.HandlerFunc(ctrl.List))
	token := middleware.SignToken(model.Claims{ID: "user", RoleName: "HR", ExpiresAt: time.Now().Add(time.Hour).UnixMilli()}, secret)
	request := httptest.NewRequest(http.MethodGet, "/zoom-resources", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("HR read status = %d", response.Code)
	}
}
