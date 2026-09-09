package email

import (
	"context"
	"errors"
	"fmt"
	"my-ple-backend/model"
	"os"
)

func WorkAssignmentMessage(from, to, id string, a model.WorkAssignmentEmail) ([]byte, error) {
	subject := "New Work Assignment - " + a.ProjectName + " - " + a.Title
	body := fmt.Sprintf("Anda telah ditugaskan ke pekerjaan baru.\n\nJudul:\n%s\n\nProject:\n%s\n\nDitugaskan oleh:\n%s\n\nTanggal Mulai:\n%s\n\nDeadline:\n%s\n\nPriority:\n%s\n\nDescription:\n%s\n", a.Title, a.ProjectName, a.AssignedByName, a.StartDate, a.DueDate, a.Priority, a.Description)
	return alternativeMessage(from, to, id, "work-assignment.local", subject, body)
}

func SendWorkAssignment(ctx context.Context, to, id string, a model.WorkAssignmentEmail) error {
	if !ZoomEnabled() {
		return errors.New("email disabled")
	}
	if err := ValidateZoomConfig(); err != nil {
		return err
	}
	from := os.Getenv("SMTP_FROM")
	message, err := WorkAssignmentMessage(from, to, id, a)
	if err != nil {
		return err
	}
	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "587"
	}
	return sendMessage(ctx, os.Getenv("SMTP_HOST"), port, from, to, message)
}
