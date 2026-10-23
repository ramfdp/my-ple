package email

import (
	"context"
	"errors"
	"fmt"
	"my-ple-backend/model"
	"os"
	"strings"
)

func OvertimeMessage(from, to, id string, item model.OvertimeNotification) ([]byte, error) {
	body := fmt.Sprintf("%s\n\n%s\n\nNama: %s\nPeriode: %s\nStatus: %s\n", item.Title, item.Message, item.EmployeeName, item.Period, item.Status)
	if origin := strings.TrimRight(os.Getenv("FRONTEND_ORIGIN"), "/"); origin != "" {
		body += "\nLihat Lembur: " + origin + item.Path + "\n"
	}
	return alternativeMessage(from, to, id, "overtime.local", item.Title+" - "+item.EmployeeName+" - "+item.Period, body)
}

func SendOvertime(ctx context.Context, to, id string, item model.OvertimeNotification) error {
	if !Enabled() {
		return errors.New("email disabled")
	}
	if err := ValidateZoomConfig(); err != nil {
		return err
	}
	from := os.Getenv("SMTP_FROM")
	message, err := OvertimeMessage(from, to, id, item)
	if err != nil {
		return err
	}
	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "587"
	}
	return sendMessage(ctx, os.Getenv("SMTP_HOST"), port, from, to, message)
}
