package notification

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"log"
	"time"
)

type Change struct {
	Event       string `json:"event"`
	WorkspaceID int    `json:"workspaceId,omitempty"`
	EntityID    string `json:"entityId,omitempty"`
	Revision    int64  `json:"revision"`
}

// Call only after the mutation commits. The existing PostgreSQL connection carries
// invalidations between backend replicas; HTTP remains the source of truth.
func (s *Service) Changed(ctx context.Context, event string, workspace int, id string, revision int64) {
	if s == nil || s.DB == nil {
		return
	}
	if revision == 0 {
		revision = time.Now().UnixMicro()
	}
	change := Change{event, workspace, id, revision}
	payload, _ := json.Marshal(change)
	if _, err := s.DB.Conn.ExecContext(ctx, `SELECT pg_notify('my_ple_changes',$1)`, string(payload)); err != nil {
		log.Print("realtime category=publish_failed")
	}
	// Deliver locally too, including deployments without the relay. Duplicate
	// invalidations are coalesced by clients and never apply payloads over drafts.
	s.deliverChange(ctx, change)
}

func (s *Service) deliverChange(ctx context.Context, c Change) {
	if s.Hub == nil {
		return
	}
	query := `SELECT u.id FROM users u JOIN roles r ON r.id=u.role_id WHERE u.is_active`
	args := []any{}
	switch c.Event {
	case "workspace.changed":
		query += ` AND EXISTS(SELECT 1 FROM workspace_access a WHERE a.user_id=u.id AND a.division_id=$1)`
		args = append(args, c.WorkspaceID)
	case "users.changed":
		query += ` AND (r.name IN ('Admin','Management','HR') OR u.id=NULLIF($1,'')::uuid)`
		args = append(args, c.EntityID)
	case "attendance.changed", "leave.changed":
		query += ` AND (r.name IN ('Admin','Management','HR') OR u.id=$1)`
		args = append(args, c.EntityID)
	case "overtime.changed":
		query += ` AND (r.name IN ('Finance','Admin') OR u.id=$1)`
		args = append(args, c.EntityID)
	case "notifications.changed":
		query += ` AND u.id=$1`
		args = append(args, c.EntityID)
	case "document-checklists.changed", "billing-documents.changed":
		query += ` AND lower(r.name) IN ('project engineer','finance','admin','management')`
	case "lessons-learned.changed":
		query += ` AND lower(r.name) IN ('project engineer','admin','management')`
	case "reports.changed", "zoom.changed", "jobs.changed": // Same authenticated scope as the existing read endpoints.
	default:
		return
	}
	rows, err := s.DB.Conn.QueryContext(ctx, query, args...)
	if err != nil {
		return
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if rows.Scan(&id) != nil {
			rows.Close()
			return
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err == nil {
		s.Hub.SendEvent(ids, c)
	}
}

func (s *Service) Relay(ctx context.Context, databaseURL string) {
	retry := time.Second
	for ctx.Err() == nil {
		conn, err := pgx.Connect(ctx, databaseURL)
		if err == nil {
			_, err = conn.Exec(ctx, `LISTEN my_ple_changes`)
			if err == nil {
				retry = time.Second
			}
			for err == nil {
				var event *pgconn.Notification
				event, err = conn.WaitForNotification(ctx)
				if err == nil {
					var change Change
					if json.Unmarshal([]byte(event.Payload), &change) == nil {
						s.deliverChange(ctx, change)
					}
				}
			}
			closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			_ = conn.Close(closeCtx)
			cancel()
		}
		if ctx.Err() != nil {
			return
		}
		log.Print("realtime category=relay_reconnecting")
		select {
		case <-ctx.Done():
			return
		case <-time.After(retry):
		}
		retry = min(retry*2, 30*time.Second)
	}
}
