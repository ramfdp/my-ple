package email

import (
	"context"
	"errors"
	"fmt"
	"my-ple-backend/model"
	"os"
	"strings"
	"time"
)

func leaveDate(value string) string {
	d, err := time.Parse("2006-01-02", value)
	if err != nil {
		return ""
	}
	months := []string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
	return fmt.Sprintf("%d %s %d", d.Day(), months[int(d.Month())-1], d.Year())
}
func LeaveMessage(from, to, id string, a model.LeaveNotification) ([]byte, error) {
	label := "Cuti"
	if a.RequestType == "IZIN" {
		label = "Izin"
	}
	status := map[string]string{"PENDING": "Menunggu Persetujuan", "APPROVED": "Disetujui", "REJECTED": "Ditolak"}[a.Status]
	body := fmt.Sprintf("%s\n\n%s\n\nNama: %s\nDivisi: %s\nJenis: %s\nTanggal: %s - %s\n\nTujuan / Alasan:\n%s\n\nCatatan:\n%s\n\nStatus: %s\n", a.Title, a.Message, a.EmployeeName, a.Division, label, leaveDate(a.StartDate), leaveDate(a.EndDate), a.Reason, a.Notes, status)
	if a.ReviewedAt != nil {
		body += fmt.Sprintf("\nReviewer: %s\nTanggal keputusan (WIB): %s\nCatatan reviewer:\n%s\n", a.ReviewerName, a.ReviewedAt.In(time.FixedZone("WIB", 7*60*60)).Format("02/01/2006 15:04"), a.ReviewerNote)
	}
	if origin := strings.TrimRight(os.Getenv("FRONTEND_ORIGIN"), "/"); origin != "" {
		body += "\nLihat Pengajuan: " + origin + a.Path + "\n"
	}
	return alternativeMessage(from, to, id, "leave-request.local", a.Title+" - "+a.EmployeeName+" - "+leaveDate(a.StartDate), body)
}
func SendLeave(ctx context.Context, to, id string, a model.LeaveNotification) error {
	if !Enabled() {
		return errors.New("email disabled")
	}
	if err := ValidateZoomConfig(); err != nil {
		return err
	}
	from := os.Getenv("SMTP_FROM")
	message, err := LeaveMessage(from, to, id, a)
	if err != nil {
		return err
	}
	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "587"
	}
	return sendMessage(ctx, os.Getenv("SMTP_HOST"), port, from, to, message)
}
