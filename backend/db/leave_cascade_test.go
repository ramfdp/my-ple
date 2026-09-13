package db

import (
	"strings"
	"testing"
)

func TestLeaveHistoryCascadesWhenUserIsDeleted(t *testing.T) {
	for _, clause := range []string{
		"leave_requests_user_id_fkey FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE",
		"leave_requests_reviewed_by_fkey FOREIGN KEY(reviewed_by) REFERENCES users(id) ON DELETE SET NULL",
		"zoom_booking_email_deliveries_leave_request_id_fkey FOREIGN KEY(leave_request_id) REFERENCES leave_requests(id) ON DELETE CASCADE",
	} {
		if !strings.Contains(leaveSQL, clause) {
			t.Fatalf("missing migration clause %q", clause)
		}
	}
}
