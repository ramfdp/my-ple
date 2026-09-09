package email

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"os"
	"strings"
	"time"

	"my-ple-backend/model"
)

func ZoomEnabled() bool { return Enabled() }

func ValidateZoomConfig() error {
	if !ZoomEnabled() {
		return nil
	}
	if os.Getenv("SMTP_HOST") != "smtp.gmail.com" || os.Getenv("SMTP_FROM") != "aiple5766@gmail.com" || os.Getenv("SMTP_USERNAME") != "aiple5766@gmail.com" || strings.TrimSpace(os.Getenv("SMTP_PASSWORD")) == "" {
		return errors.New("Email requires SMTP_HOST=smtp.gmail.com, SMTP_FROM and SMTP_USERNAME=aiple5766@gmail.com, and SMTP_PASSWORD containing a Gmail App Password")
	}
	if port := os.Getenv("SMTP_PORT"); port != "" && port != "587" && port != "465" {
		return errors.New("Email SMTP_PORT must be 587 or 465")
	}
	return nil
}

// ZoomMessage contains only information already visible in Zoom Booking.
func ZoomMessage(from, to, id string, b model.ZoomBooking, creator bool) ([]byte, error) {
	if !ValidAddress(from) || !ValidAddress(to) || strings.ContainsAny(id, "\r\n<>") {
		return nil, errors.New("invalid email headers")
	}
	zone := time.FixedZone("WIB", 7*60*60)
	start, end := b.StartAt.In(zone), b.EndAt.In(zone)
	room := strings.Join(strings.Fields(b.ZoomResource.Name+" - "+b.ZoomResource.Code), " ")
	subject, intro := "New Zoom Booking", "A new Zoom meeting has been booked."
	if creator {
		subject, intro = "Zoom Booking Confirmed", "Your Zoom booking has been successfully created."
	}
	subject += " - " + room + " - " + start.Format("02 Jan 2006")
	body := fmt.Sprintf("Zoom Booking Notification\n\n%s\n\nBooked by:\n%s\n\nZoom Room:\n%s\n\nMeeting:\n%s\n\nDate:\n%s\n\nTime (WIB):\n%s - %s\n\nDescription:\n%s\n\nZoom Link:\n%s\n", intro, b.CreatedBy, room, b.Title, start.Format("02 January 2006"), start.Format("15:04"), end.Format("02 January 2006 15:04"), b.Description, b.ZoomResource.MeetingURL)
	if b.ZoomResource.Passcode != "" {
		body += "\nPasscode:\n" + b.ZoomResource.Passcode + "\n"
	}
	return alternativeMessage(from, to, id, "zoom-booking.local", subject, body)
}

// Shared escaped HTML/plain-text MIME strategy for the existing bot outbox.
func alternativeMessage(from, to, id, domain, subject, body string) ([]byte, error) {
	if !ValidAddress(from) || !ValidAddress(to) || strings.ContainsAny(id, "\r\n<>") {
		return nil, errors.New("invalid email headers")
	}
	subject = strings.Join(strings.Fields(subject), " ")
	var out bytes.Buffer
	parts := multipart.NewWriter(&out)
	headers, err := messageHeaders(from, to, subject, id, strings.TrimSuffix(domain, ".local"))
	if err != nil {
		return nil, err
	}
	out.WriteString(headers)
	fmt.Fprintf(&out, "Content-Type: multipart/alternative; boundary=%q\r\n\r\n", parts.Boundary())
	for _, part := range []struct{ kind, body string }{
		{"text/plain", body},
		{"text/html", `<html><body><div style="font-family:Arial,sans-serif;max-width:600px;line-height:1.5;overflow-wrap:anywhere;word-break:break-word">` + strings.ReplaceAll(html.EscapeString(body), "\n", "<br>\n") + `</div></body></html>`},
	} {
		w, err := parts.CreatePart(textproto.MIMEHeader{"Content-Type": {part.kind + "; charset=utf-8"}, "Content-Transfer-Encoding": {"quoted-printable"}})
		if err != nil {
			return nil, err
		}
		q := quotedprintable.NewWriter(w)
		if _, err = q.Write([]byte(part.body)); err != nil {
			return nil, err
		}
		if err = q.Close(); err != nil {
			return nil, err
		}
	}
	if err := parts.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func SendZoom(ctx context.Context, to, id string, b model.ZoomBooking, creator bool) error {
	if !ZoomEnabled() {
		return errors.New("Zoom email disabled")
	}
	if err := ValidateZoomConfig(); err != nil {
		return err
	}
	from := os.Getenv("SMTP_FROM")
	message, err := ZoomMessage(from, to, id, b, creator)
	if err != nil {
		return err
	}
	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "587"
	}
	return sendMessage(ctx, os.Getenv("SMTP_HOST"), port, from, to, message)
}
