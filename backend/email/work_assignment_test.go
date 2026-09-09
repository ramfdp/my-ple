package email

import (
	"io"
	"mime"
	"mime/multipart"
	"my-ple-backend/model"
	"net/mail"
	"strings"
	"testing"
)

func TestWorkAssignmentMessage(t *testing.T) {
	a := model.WorkAssignmentEmail{WorkspaceNotification: model.WorkspaceNotification{Title: "Review <script>\r\nBcc: attacker@example.com", ProjectName: "MPS <img>", StartDate: "2026-09-09", DueDate: "2026-09-12", Priority: "HIGH"}, AssignedByName: "Manager <svg>", Description: "<img onerror=alert(1)> & details"}
	data, err := WorkAssignmentMessage("aiple5766@gmail.com", "budi@example.com", "event-1", a)
	if err != nil {
		t.Fatal(err)
	}
	m, err := mail.ReadMessage(strings.NewReader(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	subject, err := new(mime.WordDecoder).DecodeHeader(m.Header.Get("Subject"))
	if err != nil || !strings.HasPrefix(subject, "New Work Assignment - MPS") {
		t.Fatal(subject, err)
	}
	if m.Header.Get("Bcc") != "" || m.Header.Get("Message-ID") != "<event-1@work-assignment.local>" {
		t.Fatal("unsafe headers", m.Header)
	}
	_, params, _ := mime.ParseMediaType(m.Header.Get("Content-Type"))
	parts := multipart.NewReader(m.Body, params["boundary"])
	for i := 0; i < 2; i++ {
		p, err := parts.NextPart()
		if err != nil {
			t.Fatal(err)
		}
		raw, err := io.ReadAll(p)
		if err != nil {
			t.Fatal(err)
		}
		body := string(raw)
		for _, want := range []string{"2026-09-09", "2026-09-12", "HIGH", "details"} {
			if !strings.Contains(body, want) {
				t.Fatal("missing content", want)
			}
		}
		if i == 0 && !strings.Contains(body, a.Description) {
			t.Fatal("missing plain text")
		}
		if i == 1 && (strings.Contains(body, "<img") || strings.Contains(body, "<script") || strings.Contains(body, "<svg") || !strings.Contains(body, "&lt;img")) {
			t.Fatal("unescaped user content")
		}
	}
	if _, err = WorkAssignmentMessage("aiple5766@gmail.com", "invalid", "id", a); err == nil {
		t.Fatal("invalid recipient accepted")
	}
	if _, err = WorkAssignmentMessage("aiple5766@gmail.com", "budi@example.com", "id\r\nBcc: bad", a); err == nil {
		t.Fatal("invalid ID accepted")
	}
}
