package db

import (
	"context"
	"encoding/json"
	"my-ple-backend/model"
	"slices"
	"strings"
)

func ValidateWorkspacePage(in *model.WorkspacePageInput) error {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" || len(in.Title) > 200 || !slices.Contains([]string{model.PagePersonal, model.PageDivision}, in.Visibility) || len(in.Content) > 200 {
		return ErrWorkspaceInvalid
	}
	if in.ParentID != nil && !workspaceUUID(*in.ParentID) {
		return ErrWorkspaceInvalid
	}
	total := 0
	for _, line := range in.Content {
		total += len(line.Text) + len(line.URL)
		if !slices.Contains([]string{"text", "heading", "bullet", "check", "link", "bold", "italic"}, line.Kind) || !validWorkURL(line.URL) {
			return ErrWorkspaceInvalid
		}
	}
	if total > 10000 {
		return ErrWorkspaceInvalid
	}
	if in.Content == nil {
		in.Content = []model.PageLine{}
	}
	return nil
}
func scanWorkspacePage(s interface{ Scan(...any) error }) (model.WorkspacePage, error) {
	var p model.WorkspacePage
	var raw []byte
	err := s.Scan(&p.ID, &p.WorkspaceID, &p.ParentID, &p.Title, &raw, &p.Visibility, &p.CreatedBy, &p.UpdatedAt, &p.Version)
	if err == nil {
		err = json.Unmarshal(raw, &p.Content)
	}
	return p, err
}

const pageColumns = `id,division_id,parent_id,title,content,visibility,created_by,updated_at,version`

func (db *Database) WorkspacePage(ctx context.Context, userID, id string) (model.WorkspacePage, error) {
	var p model.WorkspacePage
	if !workspaceUUID(id) {
		return p, ErrWorkspaceInvalid
	}
	// First lookup is metadata only; unauthorized personal content never leaves PostgreSQL.
	var division int
	var creator, visibility string
	err := db.Conn.QueryRowContext(ctx, `SELECT division_id,created_by,visibility FROM workspace_pages WHERE id=$1`, id).Scan(&division, &creator, &visibility)
	if err != nil {
		return p, err
	}
	manager, err := db.WorkspaceAccess(ctx, userID, division)
	if err != nil {
		return p, err
	}
	if visibility == model.PagePersonal && creator != userID {
		return p, ErrWorkspaceForbidden
	}
	p, err = scanWorkspacePage(db.Conn.QueryRowContext(ctx, `SELECT `+pageColumns+` FROM workspace_pages WHERE id=$1 AND (visibility='DIVISION' OR created_by=$2)`, id, userID))
	p.CanEdit = creator == userID || manager
	return p, err
}
func (db *Database) WorkspacePages(ctx context.Context, userID string, division int) ([]model.WorkspacePage, error) {
	manager, err := db.WorkspaceAccess(ctx, userID, division)
	if err != nil {
		return nil, err
	}
	// Listing excludes content; fetch a page's body only after authorization.
	rows, err := db.Conn.QueryContext(ctx, `SELECT id,division_id,parent_id,title,'[]'::jsonb,visibility,created_by,updated_at,version FROM workspace_pages WHERE division_id=$1 AND (visibility='DIVISION' OR created_by=$2) ORDER BY title,id`, division, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.WorkspacePage{}
	for rows.Next() {
		p, e := scanWorkspacePage(rows)
		if e != nil {
			return nil, e
		}
		p.CanEdit = manager || p.CreatedBy == userID
		items = append(items, p)
	}
	return items, rows.Err()
}
func (db *Database) SaveWorkspacePage(ctx context.Context, userID string, division int, id string, in model.WorkspacePageInput) (model.WorkspacePage, error) {
	var empty model.WorkspacePage
	if err := ValidateWorkspacePage(&in); err != nil {
		return empty, err
	}
	if _, err := db.WorkspaceAccess(ctx, userID, division); err != nil {
		return empty, err
	}
	var old model.WorkspacePage
	var err error
	if id != "" {
		old, err = db.WorkspacePage(ctx, userID, id)
		if err != nil {
			return empty, err
		}
		if old.WorkspaceID != division || !old.CanEdit {
			return empty, ErrWorkspaceForbidden
		}
	}
	creator := userID
	if old.ID != "" {
		creator = old.CreatedBy
	}
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return empty, err
	}
	defer tx.Rollback()
	// Serialize hierarchy writes per workspace so simultaneous moves cannot create cycles.
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(71260410,$1)`, division); err != nil {
		return empty, err
	}
	if id != "" {
		var version int
		if err = tx.QueryRowContext(ctx, `SELECT version FROM workspace_pages WHERE id=$1 FOR UPDATE`, id).Scan(&version); err != nil {
			return empty, err
		}
		if version != in.Version || version != old.Version {
			return empty, ErrWorkspaceConflict
		}
	}
	if in.ParentID != nil {
		var parentDivision int
		var parentVisibility, parentCreator string
		err = tx.QueryRowContext(ctx, `SELECT division_id,visibility,created_by FROM workspace_pages WHERE id=$1`, *in.ParentID).Scan(&parentDivision, &parentVisibility, &parentCreator)
		if err != nil {
			return empty, ErrWorkspaceInvalid
		}
		if parentDivision != division || parentVisibility != in.Visibility || (parentVisibility == model.PagePersonal && (parentCreator != creator || parentCreator != userID)) {
			return empty, ErrWorkspaceForbidden
		}
		if id != "" {
			var cycle bool
			err = tx.QueryRowContext(ctx, `WITH RECURSIVE ancestors AS(SELECT id,parent_id FROM workspace_pages WHERE id=$1 UNION SELECT p.id,p.parent_id FROM workspace_pages p JOIN ancestors a ON p.id=a.parent_id) SELECT EXISTS(SELECT 1 FROM ancestors WHERE id=$2)`, *in.ParentID, id).Scan(&cycle)
			if err != nil {
				return empty, err
			}
			if cycle {
				return empty, ErrWorkspaceInvalid
			}
		}
	}
	if old.ID != "" && old.Visibility != in.Visibility {
		var children bool
		if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_pages WHERE parent_id=$1)`, id).Scan(&children); err != nil {
			return empty, err
		}
		if children {
			return empty, ErrWorkspaceInvalid
		}
		if in.Visibility == model.PagePersonal && old.CreatedBy != userID {
			return empty, ErrWorkspaceForbidden
		}
	}
	raw, _ := json.Marshal(in.Content)
	if id == "" {
		id = NewID()
		_, err = tx.ExecContext(ctx, `INSERT INTO workspace_pages(id,division_id,parent_id,title,content,visibility,created_by) VALUES($1,$2,$3,$4,$5,$6,$7)`, id, division, in.ParentID, in.Title, string(raw), in.Visibility, userID)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE workspace_pages SET parent_id=$2,title=$3,content=$4,visibility=$5,updated_at=NOW(),version=version+1 WHERE id=$1`, id, in.ParentID, in.Title, string(raw), in.Visibility)
	}
	if err != nil {
		return empty, err
	}
	if err = tx.Commit(); err != nil {
		return empty, err
	}
	return db.WorkspacePage(ctx, userID, id)
}
func (db *Database) DeleteWorkspacePage(ctx context.Context, userID, id string, version int) error {
	p, err := db.WorkspacePage(ctx, userID, id)
	if err != nil {
		return err
	}
	if !p.CanEdit {
		return ErrWorkspaceForbidden
	}
	if version != p.Version { return ErrWorkspaceConflict }
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(71260410,$1)`, p.WorkspaceID); err != nil {
		return err
	}
	var children bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_pages WHERE parent_id=$1)`, id).Scan(&children); err != nil {
		return err
	}
	if children {
		return ErrWorkspaceInvalid
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM workspace_pages WHERE id=$1 AND version=$2`, id, version)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrWorkspaceConflict
	}
	return tx.Commit()
}
