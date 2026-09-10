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

func TestAssignmentMessageUsesConfiguredTaskLink(t *testing.T) {
	t.Setenv("FRONTEND_ORIGIN", "https://ple.example.test")
	a := model.WorkAssignmentEmail{WorkspaceNotification: model.WorkspaceNotification{WorkspaceID: 7, Title: "Bridge", DueDate: "2026-09-14"}, TaskID: "task-1", WorkspaceName: "Engineering", AssignedByName: "Supervisor"}
	message, err := WorkAssignmentMessage("sender@example.test", "assignee@example.test", "event-1", a)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := mail.ReadMessage(strings.NewReader(string(message)))
	if err != nil {
		t.Fatal(err)
	}
	_, params, err := mime.ParseMediaType(parsed.Header.Get("Content-Type"))
	if err != nil {
		t.Fatal(err)
	}
	part, err := multipart.NewReader(parsed.Body, params["boundary"]).NextPart()
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(part)
	if err != nil {
		t.Fatal(err)
	}
	decoded := string(body)
	for _, want := range []string{"Bridge", "Supervisor", "Engineering", "2026-09-14", "https://ple.example.test/workspace/7?task=task-1"} {
		if !strings.Contains(decoded, want) {
			t.Errorf("missing assignment content %q", want)
		}
	}
	t.Setenv("FRONTEND_ORIGIN", "")
	if _, err = WorkAssignmentMessage("sender@example.test", "assignee@example.test", "event-1", a); err == nil {
		t.Fatal("missing origin must not silently use localhost")
	}
	t.Setenv("FRONTEND_ORIGIN", "javascript:alert(1)")
	if _, err = WorkAssignmentMessage("sender@example.test", "assignee@example.test", "event-1", a); err == nil {
		t.Fatal("unsafe origin accepted")
	}
}
