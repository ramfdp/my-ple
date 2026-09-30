package email

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
)

func AttendanceReminderMessage(from, to, id, name, todayDate string) ([]byte, error) {
	subject := fmt.Sprintf("Pengingat Absensi - %s", todayDate)
	origin := strings.TrimRight(os.Getenv("FRONTEND_ORIGIN"), "/")
	attendanceURL := origin + "/attendance"
	if origin == "" {
		attendanceURL = "/attendance"
	}

	body := fmt.Sprintf("Halo %s,\n\nIni adalah pengingat otomatis bahwa Anda belum melakukan Clock In untuk hari ini (%s).\n\nBatas waktu absensi pagi adalah jam 08:15 WIB. Silakan segera melakukan absensi melalui aplikasi MyHIS.\n\nKlik tautan berikut untuk melakukan absensi:\n%s\n\nTerima kasih.", name, todayDate, attendanceURL)
	return alternativeMessage(from, to, id, "attendance-reminder.local", subject, body)
}

func SendAttendanceReminder(ctx context.Context, to, id, name, todayDate string) error {
	if !Enabled() {
		return errors.New("email disabled")
	}
	from := os.Getenv("SMTP_FROM")
	message, err := AttendanceReminderMessage(from, to, id, name, todayDate)
	if err != nil {
		return err
	}
	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "587"
	}
	return sendMessage(ctx, os.Getenv("SMTP_HOST"), port, from, to, message)
}
