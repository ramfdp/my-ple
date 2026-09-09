package email

import (
	"io"
	"mime"
	"mime/multipart"
	"my-ple-backend/model"
	"net/mail"
	"strings"
	"testing"
	"time"
)

func TestLeaveMessage(t *testing.T) {
	t.Setenv("FRONTEND_ORIGIN", "https://ple.example")
	reviewed := time.Date(2026, 9, 9, 7, 45, 0, 0, time.UTC)
	for _, status := range []string{"PENDING", "APPROVED", "REJECTED"} {
		item := model.LeaveNotification{LeaveRequest: model.LeaveRequest{EmployeeName: "Budi\r\nBcc: injected@example.com", RequestType: "CUTI", StartDate: "2026-09-15", EndDate: "2026-09-17", Reason: "<script>private</script>", Notes: "Catatan", Status: status, ReviewerName: "Reviewer", ReviewerNote: "Agenda", ReviewedAt: &reviewed}, Title: "Pengajuan Cuti", Message: "Pengajuan", Path: "/leave?request=id"}
		raw, err := LeaveMessage("aiple5766@gmail.com", "employee@example.com", "event-id", item)
		if err != nil {
			t.Fatal(err)
		}
		message, err := mail.ReadMessage(strings.NewReader(string(raw)))
		if err != nil {
			t.Fatal(err)
		}
		if message.Header.Get("To") != "employee@example.com" || message.Header.Get("Cc") != "" || message.Header.Get("Bcc") != "" {
			t.Fatal("recipient leak")
		}
		if message.Header.Get("Message-ID") != "<event-id@leave-request.local>" {
			t.Fatal("unstable message ID")
		}
		_, params, _ := mime.ParseMediaType(message.Header.Get("Content-Type"))
		parts := multipart.NewReader(message.Body, params["boundary"])
		for i := 0; i < 2; i++ {
			part, err := parts.NextPart()
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(part)
			if err != nil {
				t.Fatal(err)
			}
			body := string(data)
			for _, want := range []string{"15 September 2026", "17 September 2026", "14:45", "https://ple.example/leave?request=id", "Agenda", map[string]string{"PENDING": "Menunggu Persetujuan", "APPROVED": "Disetujui", "REJECTED": "Ditolak"}[status]} {
				if !strings.Contains(body, want) {
					t.Fatal("missing", want)
				}
			}
			if i == 1 && strings.Contains(body, "<script>") {
				t.Fatal("unescaped HTML")
			}
		}
	}
	if _, err := LeaveMessage("aiple5766@gmail.com", "invalid", "event", model.LeaveNotification{}); err == nil {
		t.Fatal("invalid address accepted")
	}
}
