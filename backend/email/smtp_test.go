package email

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"testing"
)

func TestAttachmentMessageAndHeaderValidation(t *testing.T) {
	pdf := []byte("%PDF-1.4\n" + strings.Repeat("report data", 70) + "\n%%EOF")
	data, err := Message("reports@example.test", "manager@example.test", "Weekly Report - Project Engineer", "Finalized report attached", "weekly-report-2026-08-31.pdf", "delivery-id", pdf)
	if err != nil {
		t.Fatal(err)
	}
	message, err := mail.ReadMessage(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	media, params, err := mime.ParseMediaType(message.Header.Get("Content-Type"))
	if err != nil || media != "multipart/mixed" {
		t.Fatal("missing multipart")
	}
	reader := multipart.NewReader(message.Body, params["boundary"])
	body, err := reader.NextPart()
	if err != nil {
		t.Fatal(err)
	}
	text, _ := io.ReadAll(body)
	if !bytes.Contains(text, []byte("Finalized")) {
		t.Fatal("missing body")
	}
	attachment, err := reader.NextPart()
	if err != nil {
		t.Fatal(err)
	}
	if attachment.FileName() != "weekly-report-2026-08-31.pdf" || attachment.Header.Get("Content-Type") != "application/pdf" {
		t.Fatal("attachment metadata")
	}
	decoded, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, attachment))
	if err != nil || !bytes.Equal(decoded, pdf) {
		t.Fatal("attachment bytes changed")
	}
	if _, err = Message("reports@example.test", "bad\r\nBcc: thief@example.test", "subject", "body", "test.pdf", "id", pdf); err == nil {
		t.Fatal("header injection accepted")
	}
	for _, invalid := range []string{"", "bad", "Name <a@example.test>", "a@example.test\r\n"} {
		if ValidAddress(invalid) {
			t.Fatalf("accepted %q", invalid)
		}
	}
}
