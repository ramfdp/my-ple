package db

import (
	"context"
	"database/sql"
	"errors"
	"my-ple-backend/model"
)

var ErrWorkspaceForbidden = errors.New("Anda tidak memiliki akses ke workspace atau tindakan ini.")
var ErrWorkspaceInvalid = errors.New("Data workspace tidak valid.")
var ErrWorkspaceConflict = errors.New("Data sudah berubah. Silakan muat ulang.")
var ErrUserHasWorkspaceHistory = errors.New("user has workspace history")

// Always resolve current account privileges from PostgreSQL, never request fields.
func (db *Database) WorkspaceAccess(ctx context.Context, userID string, divisionID int) (bool, error) {
	var exists, allowed, manager bool
	err := db.Conn.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM divisions WHERE id=$2),
	EXISTS(SELECT 1 FROM workspace_access WHERE user_id=$1 AND division_id=$2),
	EXISTS(SELECT 1 FROM workspace_access WHERE user_id=$1 AND division_id=$2 AND can_manage)`, userID, divisionID).Scan(&exists, &allowed, &manager)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, sql.ErrNoRows
	}
	if !allowed && !manager {
		return false, ErrWorkspaceForbidden
	}
	return manager, nil
}

func (db *Database) Workspaces(ctx context.Context, userID string) ([]model.Workspace, error) {
	rows, err := db.Conn.QueryContext(ctx, `SELECT d.id,d.name,
 (SELECT count(*) FROM workspace_members m WHERE m.division_id=d.id),
 (SELECT count(*) FROM work_items t WHERE t.division_id=d.id AND t.archived_at IS NULL AND t.status<>'DONE'),
	a.can_manage
	FROM divisions d JOIN workspace_access a ON a.division_id=d.id
	WHERE a.user_id=$1
 ORDER BY d.sort_order,d.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.Workspace{}
	for rows.Next() {
		var w model.Workspace
		if err = rows.Scan(&w.ID, &w.Name, &w.MemberCount, &w.ActiveTaskCount, &w.CanManage); err != nil {
			return nil, err
		}
		items = append(items, w)
	}
	return items, rows.Err()
}

func (db *Database) WorkspaceMembers(ctx context.Context, userID string, divisionID int) ([]model.WorkspaceMember, error) {
	if _, err := db.WorkspaceAccess(ctx, userID, divisionID); err != nil {
		return nil, err
	}
	rows, err := db.Conn.QueryContext(ctx, `SELECT user_id,name,avatar_url FROM workspace_members WHERE division_id=$1 ORDER BY name,user_id`, divisionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.WorkspaceMember{}
	for rows.Next() {
		var m model.WorkspaceMember
		if err = rows.Scan(&m.ID, &m.Name, &m.AvatarURL); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}
