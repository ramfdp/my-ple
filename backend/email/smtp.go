// Package email sends MIME attachments using the standard library and TLS.
package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"os"
	"strings"
	"time"
)

// ErrUncertain means DATA may have been accepted. Never automatically resend.
var ErrUncertain = errors.New("SMTP acceptance uncertain; operator review required")

func ValidAddress(value string) bool {
	a, err := mail.ParseAddress(value)
	return err == nil && a.Address == value && !strings.ContainsAny(value, "\r\n") && strings.Contains(value, "@")
}

func Message(from, to, subject, body, filename, id string, pdf []byte) ([]byte, error) {
	if !ValidAddress(from) || !ValidAddress(to) || strings.ContainsAny(subject+filename+id, "\r\n") {
		return nil, errors.New("invalid email headers")
	}
	var out bytes.Buffer
	parts := multipart.NewWriter(&out)
	fmt.Fprintf(&out, "From: %s\r\nTo: %s\r\nSubject: %s\r\nMessage-ID: <%s@weekly-report.local>\r\nDate: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=%q\r\n\r\n", from, to, mime.QEncoding.Encode("utf-8", subject), id, time.Now().UTC().Format(time.RFC1123Z), parts.Boundary())
	text, err := parts.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/plain; charset=utf-8"}})
	if err != nil {
		return nil, err
	}
	if _, err = text.Write([]byte(body)); err != nil {
		return nil, err
	}
	attachment, err := parts.CreatePart(textproto.MIMEHeader{"Content-Type": {"application/pdf"}, "Content-Disposition": {mime.FormatMediaType("attachment", map[string]string{"filename": filename})}, "Content-Transfer-Encoding": {"base64"}})
	if err != nil {
		return nil, err
	}
	encoded := base64.StdEncoding.EncodeToString(pdf)
	for len(encoded) > 0 {
		n := 76
		if len(encoded) < n {
			n = len(encoded)
		}
		if _, err = fmt.Fprintf(attachment, "%s\r\n", encoded[:n]); err != nil {
			return nil, err
		}
		encoded = encoded[n:]
	}
	if err = parts.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func Send(ctx context.Context, to, subject, body, filename, id string, pdf []byte) error {
	host, port, from := os.Getenv("SMTP_HOST"), os.Getenv("SMTP_PORT"), os.Getenv("SMTP_FROM")
	if port == "" {
		port = "587"
	}
	if host == "" || from == "" {
		return errors.New("SMTP_HOST and SMTP_FROM must be configured")
	}
	message, err := Message(from, to, subject, body, filename, id, pdf)
	if err != nil {
		return err
	}
	return sendMessage(ctx, host, port, from, to, message)
}

// Shared TLS transport; callers retain their own message format and feature flag.
func sendMessage(ctx context.Context, host, port, from, to string, message []byte) error {
	dialer := net.Dialer{Timeout: 15 * time.Second}
	connection, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, port))
	if err != nil {
		return errors.New("SMTP connection failed")
	}
	defer connection.Close()
	deadline := time.Now().Add(30 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	connection.SetDeadline(deadline)
	tlsConfig := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	if port == "465" {
		secured := tls.Client(connection, tlsConfig)
		if err = secured.HandshakeContext(ctx); err != nil {
			return errors.New("SMTP TLS failed")
		}
		connection = secured
	}
	client, err := smtp.NewClient(connection, host)
	if err != nil {
		return errors.New("SMTP greeting failed")
	}
	defer client.Close()
	if port != "465" {
		if err = client.StartTLS(tlsConfig); err != nil {
			return errors.New("SMTP STARTTLS required")
		}
	}
	if username := os.Getenv("SMTP_USERNAME"); username != "" {
		if err = client.Auth(smtp.PlainAuth("", username, os.Getenv("SMTP_PASSWORD"), host)); err != nil {
			return smtpFailure("authentication", err)
		}
	}
	if err = client.Mail(from); err != nil {
		return smtpFailure("sender", err)
	}
	if err = client.Rcpt(to); err != nil {
		return smtpFailure("recipient", err)
	}
	writer, err := client.Data()
	if err != nil {
		return smtpFailure("data", err)
	}
	if _, err = writer.Write(message); err != nil {
		return ErrUncertain
	}
	if err = writer.Close(); err != nil {
		var rejection *textproto.Error
		if errors.As(err, &rejection) {
			return smtpFailure("message", err)
		}
		return ErrUncertain
	}
	// Receipt of DATA's success response is delivery acceptance; QUIT is not.
	_ = client.Quit()
	return nil
}
