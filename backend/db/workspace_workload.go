package db

import (
	"context"
	"my-ple-backend/model"
	"my-ple-backend/workspace"
)

func (db *Database) WorkspaceWorkload(ctx context.Context, userID string, division int) ([]workspace.Workload, error) {
	members, err := db.WorkspaceMembers(ctx, userID, division)
	if err != nil {
		return nil, err
	}
	rows, err := db.Conn.QueryContext(ctx, `SELECT a.user_id,t.id,t.title,t.status,t.priority,COALESCE(t.start_date::text,''),COALESCE(t.due_date::text,'')
 FROM work_items t JOIN work_item_assignees a ON a.work_item_id=t.id JOIN workspace_members m ON m.user_id=a.user_id AND m.division_id=t.division_id
 WHERE t.division_id=$1 AND t.archived_at IS NULL AND t.status<>'DONE' ORDER BY t.due_date NULLS LAST,t.id`, division)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byUser := map[string][]model.WorkItem{}
	for rows.Next() {
		var id string
		var t model.WorkItem
		if err = rows.Scan(&id, &t.ID, &t.Title, &t.Status, &t.Priority, &t.StartDate, &t.DueDate); err != nil {
			return nil, err
		}
		byUser[id] = append(byUser[id], t)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	items := []workspace.Workload{}
	for _, m := range members {
		w := workspace.Calculate(byUser[m.ID], db.Now())
		w.Member = m
		items = append(items, w)
	}
	return items, nil
}
func (db *Database) WorkspaceOverview(ctx context.Context, userID string, division int) (map[string]int, error) {
	if _, err := db.WorkspaceAccess(ctx, userID, division); err != nil {
		return nil, err
	}
	var active, progress, revision, done, overdue int
	err := db.Conn.QueryRowContext(ctx, `SELECT count(*) FILTER(WHERE status<>'DONE'),count(*) FILTER(WHERE status='IN_PROGRESS'),count(*) FILTER(WHERE status='NEED_REVISION'),count(*) FILTER(WHERE status='DONE'),count(*) FILTER(WHERE status<>'DONE' AND due_date<(NOW() AT TIME ZONE 'Asia/Jakarta')::date) FROM work_items WHERE division_id=$1 AND archived_at IS NULL`, division).Scan(&active, &progress, &revision, &done, &overdue)
	return map[string]int{"active": active, "inProgress": progress, "needRevision": revision, "done": done, "overdue": overdue}, err
}
