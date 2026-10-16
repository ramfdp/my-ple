package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"my-ple-backend/model"
)

var ErrAISnapshotTooLarge = errors.New("AI snapshot exceeds 200 actionable tasks or 64 KiB")

// AISnapshot uses one read-only repeatable-read view; 201 rows detect overflow.
func (db *Database) AISnapshot(ctx context.Context, workspaceID int) (model.AISnapshot, error) {
	tx, err := db.Conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return model.AISnapshot{}, err
	}
	defer tx.Rollback()
	s, err := aiSnapshotTx(ctx, tx, workspaceID)
	if err != nil {
		return s, err
	}
	if err = tx.Commit(); err != nil {
		return s, err
	}
	return s, nil
}

// aiSnapshotTx also serves imports and reviews after their transaction locks source tables.
func aiSnapshotTx(ctx context.Context, tx *sql.Tx, workspaceID int) (model.AISnapshot, error) {
	s := model.AISnapshot{SchemaVersion: "1.0", WorkspaceID: workspaceID, Tasks: []model.AITask{}, AllowedAssigneeIDs: []string{}}
	var err error
	var exists bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM divisions WHERE id=$1)`, workspaceID).Scan(&exists); err != nil {
		return s, err
	}
	if !exists {
		return s, sql.ErrNoRows
	}
	if err = tx.QueryRowContext(ctx, `SELECT CURRENT_TIMESTAMP`).Scan(&s.SnapshotAt); err != nil {
		return s, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT jsonb_build_object(
	'id',t.id,'workspaceId',t.division_id,'jobId',t.job_id,'jobName',left(COALESCE(j.name,''),160),
	'title',left(t.title,160),'description',left(t.description,1000),
	'startDate',COALESCE(t.start_date::text,''),'dueDate',COALESCE(t.due_date::text,''),
	'status',t.status,'priority',t.priority,'assigneeIds','[]'::jsonb,
	'revisionCount',COALESCE(t.revision_count,0),'requiresQa',t.requires_qa,'qaStage',COALESCE(t.qa_stage,''),
	'archived',false,'version',t.version)
	FROM work_items t LEFT JOIN jobs j ON j.id=t.job_id AND j.division_id=t.division_id
	WHERE t.division_id=$1 AND t.archived_at IS NULL AND t.status<>'DONE'
	ORDER BY t.id LIMIT 201`, workspaceID)
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var raw []byte
		var task model.AITask
		if err = rows.Scan(&raw); err != nil {
			break
		}
		if err = json.Unmarshal(raw, &task); err != nil {
			break
		}
		s.Tasks = append(s.Tasks, task)
		if len(s.Tasks) > 200 {
			err = ErrAISnapshotTooLarge
			break
		}
	}
	if err == nil {
		err = rows.Err()
	}
	if closeErr := rows.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return s, err
	}
	data, err := json.Marshal(s)
	if err != nil {
		return s, err
	}
	if len(data) > 64*1024 {
		return s, ErrAISnapshotTooLarge
	}
	s.SnapshotAt = s.SnapshotAt.UTC().Truncate(time.Microsecond)
	return s, nil
}
