package email

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/quotedprintable"
	"my-ple-backend/model"
	"net/mail"
	"strings"
	"testing"
)

func TestFeedbackMessage(t *testing.T) {
	user := model.Claims{Name: "Pengguna", Email: "user@gmail.com"}
	raw, err := FeedbackMessage(FeedbackRecipient, "test-id", user, "Saran 👍\nBaris kedua")
	if err != nil {
		t.Fatal(err)
	}
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if msg.Header.Get("To") != FeedbackRecipient || msg.Header.Get("Reply-To") != user.Email {
		t.Fatal("wrong recipient or reply address")
	}
	from, err := mail.ParseAddress(msg.Header.Get("From"))
	if err != nil || from.Address != FeedbackRecipient {
		t.Fatal("wrong sender")
	}
	body, err := io.ReadAll(quotedprintable.NewReader(msg.Body))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{user.Name, user.Email, "Saran 👍\r\nBaris kedua"} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("missing %q", want)
		}
	}
	user.Email = "user@gmail.com\r\nBcc: other@gmail.com"
	if _, err = FeedbackMessage(FeedbackRecipient, "test-id", user, "test"); err == nil {
		t.Fatal("header injection accepted")
	}
	t.Setenv("EMAIL_ENABLED", "false")
	if !errors.Is(SendFeedback(context.Background(), user, "test"), ErrFeedbackUnavailable) {
		t.Fatal("disabled email accepted")
	}
}
