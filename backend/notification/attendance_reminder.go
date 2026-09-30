package notification

import (
	"context"
	"log"
	"my-ple-backend/db"
	"my-ple-backend/email"
	"time"
)

// AttendanceReminderService sends email reminders to users who haven't clocked in by 08:15 WIB
type AttendanceReminderService struct {
	DB        *db.Database
	SendEmail func(context.Context, string, string, string, string) error
}

// NewAttendanceReminderService creates a new attendance reminder service
func NewAttendanceReminderService(database *db.Database) *AttendanceReminderService {
	return &AttendanceReminderService{
		DB:        database,
		SendEmail: email.SendAttendanceReminder,
	}
}

// Start begins the background reminder service. It checks at 08:16 WIB every weekday if users haven't clocked in
func (s *AttendanceReminderService) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.checkAndSendReminders(ctx)
			}
		}
	}()
}

func (s *AttendanceReminderService) checkAndSendReminders(ctx context.Context) {
	if !email.Enabled() {
		return
	}

	now := time.Now()
	if s.DB.Now != nil {
		now = s.DB.Now()
	}
	location, _ := time.LoadLocation("Asia/Jakarta")
	now = now.In(location)

	// Check if it's a weekday (Mon-Fri)
	weekday := int(now.Weekday())
	if weekday == 0 || weekday == 6 {
		return // Skip Sunday & Saturday
	}

	// Check if it's between 08:16 and 08:17 WIB (1 minute window to avoid duplicate sends)
	hour, minute := now.Hour(), now.Minute()
	if hour != 8 || minute != 16 {
		return
	}

	// Check if reminder was already sent today
	sent, err := s.DB.HasReminderBeenSentToday(ctx)
	if err != nil {
		log.Printf("attendance reminder: check sent status failed: %v", err)
		return
	}
	if sent {
		return // Already sent today
	}

	// Get users who haven't clocked in
	users, err := s.DB.GetUsersNotClockInToday(ctx)
	if err != nil {
		log.Printf("attendance reminder: query users failed: %v", err)
		return
	}

	if len(users) == 0 {
		log.Println("attendance reminder: no users need reminder")
		// Still record that we checked, even if no one needs it
		_ = s.DB.RecordReminderSent(ctx)
		return
	}

	// Send reminders to all users
	todayDate := now.Format("2006-01-02")
	sentCount := 0
	failedCount := 0

	for _, user := range users {
		// Validate email before sending
		if !email.ValidAddress(user.Email) {
			log.Printf("attendance reminder: invalid email for user %s (%s): %s", user.ID, user.Name, user.Email)
			failedCount++
			continue
		}

		id := db.NewID()
		err := s.SendEmail(ctx, user.Email, id, user.Name, todayDate)
		if err != nil {
			status, category := email.Failure(err)
			log.Printf("attendance reminder: send failed for %s (%s) [%s]: %s", email.MaskAddress(user.Email), user.Name, status, category)
			failedCount++
		} else {
			sentCount++
			log.Printf("attendance reminder: sent to %s (%s)", email.MaskAddress(user.Email), user.Name)
		}
	}

	// Record that reminder was sent today
	if err := s.DB.RecordReminderSent(ctx); err != nil {
		log.Printf("attendance reminder: record sent failed: %v", err)
		return
	}

	log.Printf("attendance reminder: sent=%d failed=%d total=%d", sentCount, failedCount, len(users))
}

// Stop gracefully stops the reminder service
func (s *AttendanceReminderService) Stop() {
	// Ticker is already managed by Start() via context
}
