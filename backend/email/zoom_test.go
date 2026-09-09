package email

import (
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/mail"
	"strings"
	"testing"
	"time"

	"my-ple-backend/model"
)

func TestZoomMessage(t *testing.T) {
	b := model.ZoomBooking{CreatedBy: "Budi <script>", Title: "Weekly meeting", Description: "<img src=x onerror=alert(1)> & progress", StartAt: time.Date(2026, 9, 9, 3, 0, 0, 0, time.UTC), EndAt: time.Date(2026, 9, 9, 4, 0, 0, 0, time.UTC), ZoomResource: model.ZoomResource{Name: "Zoom 2", Code: "MRP", MeetingURL: "https://zoom.us/j/222?pwd=abc", Passcode: "123", HostKey: "NEVER_INCLUDE"}}
	for _, creator := range []bool{true, false} {
		data, err := ZoomMessage("aiple5766@gmail.com", "user@example.com", "event-1", b, creator)
		if err != nil {
			t.Fatal(err)
		}
		m, err := mail.ReadMessage(strings.NewReader(string(data)))
		if err != nil {
			t.Fatal(err)
		}
		subject, _ := new(mime.WordDecoder).DecodeHeader(m.Header.Get("Subject"))
		if strings.Contains(subject, "Confirmed") != creator {
			t.Fatal(subject)
		}
		_, params, _ := mime.ParseMediaType(m.Header.Get("Content-Type"))
		r := multipart.NewReader(m.Body, params["boundary"])
		for i := 0; i < 2; i++ {
			p, err := r.NextPart()
			if err != nil {
				t.Fatal(err)
			}
			body, _ := io.ReadAll(p)
			for _, want := range []string{"Zoom 2 - MRP", "09 September 2026", "10:00", "11:00", "https://zoom.us/j/222?pwd=abc", "123"} {
				if !strings.Contains(string(body), want) {
					t.Fatalf("missing %s", want)
				}
			}
			if strings.Contains(string(body), "NEVER_INCLUDE") || (i == 1 && strings.Contains(string(body), "<img")) {
				t.Fatal("unsafe HTML or host key")
			}
		}
	}
	t.Setenv("ZOOM_EMAIL_ENABLED", "false")
	if err := ValidateZoomConfig(); err != nil {
		t.Fatal(err)
	}
	if err := SendZoom(context.Background(), "invalid", "id", b, false); err == nil {
		t.Fatal("disabled send should not claim delivery")
	}
	t.Setenv("ZOOM_EMAIL_ENABLED", "true")
	t.Setenv("SMTP_PASSWORD", "")
	if ValidateZoomConfig() == nil {
		t.Fatal("missing config accepted")
	}
	for k, v := range map[string]string{"SMTP_HOST": "smtp.gmail.com", "SMTP_FROM": "aiple5766@gmail.com", "SMTP_USERNAME": "aiple5766@gmail.com", "SMTP_PASSWORD": "test-placeholder", "SMTP_PORT": "587"} {
		t.Setenv(k, v)
	}
	if err := ValidateZoomConfig(); err != nil {
		t.Fatal(err)
	}
}

func TestSMTPFailureIsSanitized(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		c, err := listener.Accept()
		if err == nil {
			c.Close()
		}
	}()
	host, port, _ := net.SplitHostPort(listener.Addr().String())
	err = sendMessage(context.Background(), host, port, "aiple5766@gmail.com", "recipient@example.com", []byte("test"))
	if err == nil || err.Error() != "SMTP greeting failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}
