package email

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"mime/quotedprintable"
	"os"
	"time"

	"my-ple-backend/model"
)

const FeedbackRecipient = "aiple5766@gmail.com"

var ErrFeedbackUnavailable = errors.New("feedback email unavailable")

func FeedbackMessage(from, id string, user model.Claims, message string) ([]byte, error) {
	if !ValidAddress(user.Email) {
		return nil, errors.New("invalid reply address")
	}
	headers, err := messageHeaders(from, FeedbackRecipient, "Kritik & Saran MyHIS", id, "feedback")
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	out.WriteString(headers)
	fmt.Fprintf(&out, "Reply-To: %s\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n", user.Email)
	writer := quotedprintable.NewWriter(&out)
	_, err = fmt.Fprintf(writer, "Nama: %s\nEmail: %s\nWaktu: %s\n\nKritik & Saran:\n%s\n", user.Name, user.Email, time.Now().UTC().Format(time.RFC3339), message)
	if err != nil {
		return nil, err
	}
	if err = writer.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func SendFeedback(ctx context.Context, user model.Claims, message string) error {
	if !Enabled() || ValidateZoomConfig() != nil {
		return ErrFeedbackUnavailable
	}
	from := os.Getenv("SMTP_FROM")
	body, err := FeedbackMessage(from, rand.Text(), user, message)
	if err != nil {
		return err
	}
	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "587"
	}
	return sendMessage(ctx, os.Getenv("SMTP_HOST"), port, from, FeedbackRecipient, body)
}
