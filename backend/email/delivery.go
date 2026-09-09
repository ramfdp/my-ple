package email

import (
	"errors"
	"fmt"
	"net/textproto"
	"os"
	"strings"
)

// EMAIL_ENABLED is authoritative; retain the old switch for existing deployments.
func Enabled() bool {
	if value, ok := os.LookupEnv("EMAIL_ENABLED"); ok {
		return strings.EqualFold(strings.TrimSpace(value), "true")
	}
	return strings.EqualFold(os.Getenv("ZOOM_EMAIL_ENABLED"), "true")
}

func NormalizeAddress(value string) string { return strings.ToLower(strings.TrimSpace(value)) }
func MaskAddress(value string) string {
	local, domain, ok := strings.Cut(value, "@")
	if !ok || local == "" {
		return "invalid"
	}
	return local[:1] + "***@" + domain
}

// Keep provider text out of logs: it may contain addresses or authentication data.
type DeliveryError struct {
	Category  string
	Code      int
	Permanent bool
}

func (e *DeliveryError) Error() string { return fmt.Sprintf("%s smtp_code=%d", e.Category, e.Code) }
func smtpFailure(phase string, err error) error {
	e := &DeliveryError{Category: "smtp_" + phase + "_failed"}
	var reply *textproto.Error
	if errors.As(err, &reply) {
		e.Code = reply.Code
		e.Permanent = reply.Code >= 500 && reply.Code < 600
		if phase == "recipient" && e.Permanent {
			e.Category = "recipient_rejected"
		}
		if reply.Code == 550 && strings.Contains(reply.Msg, "5.1.1") {
			e.Category = "recipient_not_found"
		}
	}
	return e
}
func Failure(err error) (status, category string) {
	if errors.Is(err, ErrUncertain) {
		return "uncertain", "smtp_acceptance_uncertain"
	}
	var e *DeliveryError
	if errors.As(err, &e) {
		if e.Permanent {
			return "failed_permanent", e.Error()
		}
		return "failed", e.Error()
	}
	return "failed", "smtp_delivery_failed"
}
