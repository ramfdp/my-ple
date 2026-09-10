package email

import (
	"context"
	"errors"
	"fmt"
	"my-ple-backend/model"
	"net/url"
	"os"
	"strings"
)

func WorkAssignmentMessage(from, to, id string, a model.WorkAssignmentEmail) ([]byte, error) {
	subject := "New Work Assignment - " + a.ProjectName + " - " + a.Title
	body := fmt.Sprintf("Anda telah ditugaskan ke pekerjaan baru.\n\nJudul:\n%s\n\nProject:\n%s\n\nDitugaskan oleh:\n%s\n\nTanggal Mulai:\n%s\n\nDeadline:\n%s\n\nPriority:\n%s\n\nDescription:\n%s\n", a.Title, a.ProjectName, a.AssignedByName, a.StartDate, a.DueDate, a.Priority, a.Description)
	if a.Message != "" && a.Message != "Anda ditugaskan ke pekerjaan baru" {
		subject = a.Message + " - " + a.Title
		body = strings.Replace(body, "Anda telah ditugaskan ke pekerjaan baru.", a.Message+" oleh "+a.AssignedByName+".", 1)
		body = strings.Replace(body, "Ditugaskan oleh:", "Oleh:", 1)
	}
	if origin := strings.TrimRight(os.Getenv("FRONTEND_ORIGIN"), "/"); origin != "" {
		path := "/workspace"
		if a.TaskID != "" {
			if u, e := url.Parse(origin); e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
				return nil, errors.New("invalid frontend origin")
			}
			path = fmt.Sprintf("/workspace/%d?task=%s", a.WorkspaceID, url.QueryEscape(a.TaskID))
			body += "\nWorkspace:\n" + a.WorkspaceName + "\n"
		}
		body += "\nBuka Team Workspace: " + origin + path + "\n"
	} else if a.TaskID != "" {
		return nil, errors.New("frontend origin required for assignment link")
	}
	return alternativeMessage(from, to, id, "work-assignment.local", subject, body)
}

func SendWorkAssignment(ctx context.Context, to, id string, a model.WorkAssignmentEmail) error {
	if !Enabled() {
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
