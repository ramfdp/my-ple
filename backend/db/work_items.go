package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"my-ple-backend/model"
	"net/url"
	"slices"
	"strings"
	"time"
)

func validWorkURL(value string) bool {
	if value == "" {
		return true
	}
	u, e := url.Parse(value)
	return e == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" && u.User == nil
}
func ValidateWorkItem(in *model.WorkItemInput) error {
	if in.RequestID != "" && !workspaceUUID(in.RequestID) {
		return ErrWorkspaceInvalid
	}
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" || len(in.Title) > 200 || len(in.Description) > 10000 || len(in.RelatedURL) > 2000 || !validWorkURL(in.RelatedURL) {
		return ErrWorkspaceInvalid
	}
	if !slices.Contains([]string{model.WorkTodo, model.WorkInProgress, model.WorkNeedRevision, model.WorkDone}, in.Status) || !slices.Contains([]string{model.PriorityLow, model.PriorityMedium, model.PriorityHigh, model.PriorityUrgent}, in.Priority) {
		return ErrWorkspaceInvalid
	}
	for _, d := range []string{in.StartDate, in.DueDate} {
		if d != "" {
			if _, e := time.Parse("2006-01-02", d); e != nil {
				return ErrWorkspaceInvalid
			}
		}
	}
	if in.StartDate != "" && in.DueDate != "" && in.DueDate < in.StartDate {
		return ErrWorkspaceInvalid
	}
	seen := map[string]bool{}
	for i, id := range in.AssigneeIDs {
		id = strings.ToLower(id)
		in.AssigneeIDs[i] = id
		if !workspaceUUID(id) || seen[id] {
			return ErrWorkspaceInvalid
		}
		seen[id] = true
	}
	return nil
}
func workspaceUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
			return false
		}
	}
	return true
}

const workJSON = `jsonb_build_object('id',t.id,'workspaceId',t.division_id,'jobId',t.job_id,'jobName',COALESCE(j.name,''),
 'requiresQa',t.requires_qa,'qaStage',t.qa_stage,
 'qaValidatedAt',t.qa_validated_at,'qaValidatedBy',t.qa_validated_by,'qaValidatedByName',COALESCE((SELECT name FROM users WHERE id=t.qa_validated_by),''),
 'title',t.title,'description',t.description,'relatedUrl',t.related_url,'startDate',COALESCE(t.start_date::text,''),'dueDate',COALESCE(t.due_date::text,''),
 'status',t.status,'priority',t.priority,'createdBy',COALESCE(t.created_by::text,''),'createdAt',t.created_at,'updatedAt',t.updated_at,'version',t.version,
 'assignees',COALESCE((SELECT jsonb_agg(jsonb_build_object('id',u.id,'name',u.name,'avatarUrl',u.avatar_url) ORDER BY u.name,u.id)
 FROM work_item_assignees a JOIN users u ON u.id=a.user_id WHERE a.work_item_id=t.id),'[]'),
 'commentCount',(SELECT count(*) FROM work_item_comments c WHERE c.work_item_id=t.id))`

func workPermissions(t *model.WorkItem, userID string, manager bool) {
	t.CanDelete = manager || t.CreatedBy == userID
	t.CanEdit = t.CanDelete
	for _, u := range t.Assignees {
		t.CanEdit = t.CanEdit || u.ID == userID
	}
}
func (db *Database) WorkItem(ctx context.Context, userID, id string) (model.WorkItem, error) {
	var t model.WorkItem
	if !workspaceUUID(id) {
		return t, ErrWorkspaceInvalid
	}
	var division int
	if err := db.Conn.QueryRowContext(ctx, `SELECT division_id FROM work_items WHERE id=$1`, id).Scan(&division); err != nil {
		return t, err
	}
	manager, err := db.WorkspaceAccess(ctx, userID, division)
	if err != nil {
		return t, err
	}
	var raw []byte
	err = db.Conn.QueryRowContext(ctx, `SELECT `+workJSON+` FROM work_items t LEFT JOIN jobs j ON j.id=t.job_id WHERE t.id=$1 AND t.archived_at IS NULL`, id).Scan(&raw)
	if err != nil {
		return t, err
	}
	if err = json.Unmarshal(raw, &t); err != nil {
		return t, err
	}
	workPermissions(&t, userID, manager)
	t.CanCheckQA, err = db.WorkspaceQA(ctx, userID, t.WorkspaceID)
	if err != nil {
		return t, err
	}
	return t, nil
}

type WorkFilter struct {
	Search, Assignee, Status, Priority, Start, End, Sort string
	JobID, Limit, Offset                                 int
}

func (db *Database) WorkItems(ctx context.Context, userID string, division int, f WorkFilter) ([]model.WorkItem, int, error) {
	manager, err := db.WorkspaceAccess(ctx, userID, division)
	if err != nil {
		return nil, 0, err
	}
	if f.Limit < 1 || f.Limit > 200 {
		f.Limit = 100
	}
	if f.Offset < 0 || f.Offset > 1000000 {
		return nil, 0, ErrWorkspaceInvalid
	}
	if f.Assignee != "" && !workspaceUUID(f.Assignee) {
		return nil, 0, ErrWorkspaceInvalid
	}
	for _, d := range []string{f.Start, f.End} {
		if d != "" {
			if _, e := time.Parse("2006-01-02", d); e != nil {
				return nil, 0, ErrWorkspaceInvalid
			}
		}
	}
	if f.Start != "" && f.End != "" && f.End < f.Start {
		return nil, 0, ErrWorkspaceInvalid
	}
	order := map[string]string{"title": "t.title,t.id", "due": "t.due_date NULLS LAST,t.id", "priority": "CASE t.priority WHEN 'URGENT' THEN 0 WHEN 'HIGH' THEN 1 WHEN 'MEDIUM' THEN 2 ELSE 3 END,t.id", "updated": "t.updated_at DESC,t.id"}[f.Sort]
	if order == "" {
		order = "t.created_at DESC,t.id"
	}
	args := []any{division, f.Search, f.Assignee, f.Status, f.Priority, f.JobID, f.Start, f.End}
	where := `t.division_id=$1 AND t.archived_at IS NULL AND ($2='' OR t.title ILIKE '%'||$2||'%' OR t.description ILIKE '%'||$2||'%')
 AND ($3='' OR EXISTS(SELECT 1 FROM work_item_assignees a WHERE a.work_item_id=t.id AND a.user_id=NULLIF($3,'')::uuid))
	AND ($4='' OR t.status=$4 OR ($4='ACTIVE' AND t.status<>'DONE')) AND ($5='' OR t.priority=$5) AND ($6=0 OR t.job_id=$6)
 AND ($7='' OR COALESCE(t.due_date,t.start_date)>=NULLIF($7,'')::date) AND ($8='' OR COALESCE(t.start_date,t.due_date)<=NULLIF($8,'')::date)`
	var total int
	if err = db.Conn.QueryRowContext(ctx, `SELECT count(*) FROM work_items t WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := db.Conn.QueryContext(ctx, `SELECT `+workJSON+` FROM work_items t LEFT JOIN jobs j ON j.id=t.job_id WHERE `+where+` ORDER BY `+order+` LIMIT $9 OFFSET $10`, append(args, f.Limit, f.Offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []model.WorkItem{}
	for rows.Next() {
		var raw []byte
		var t model.WorkItem
		if err = rows.Scan(&raw); err != nil {
			return nil, 0, err
		}
		if err = json.Unmarshal(raw, &t); err != nil {
			return nil, 0, err
		}
		workPermissions(&t, userID, manager)
		items = append(items, t)
	}
	return items, total, rows.Err()
}

func recordWorkActivity(ctx context.Context, tx *sql.Tx, id, actor, action, field string, old, new any) error {
	a, e := json.Marshal(old)
	if e != nil {
		return e
	}
	b, e := json.Marshal(new)
	if e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO work_item_activity(id,work_item_id,actor_user_id,action,field_name,old_value,new_value) VALUES($1,$2,$3,$4,$5,$6,$7)`, NewID(), id, actor, action, field, string(a), string(b))
	return e
}
func (db *Database) SaveWorkItem(ctx context.Context, userID string, division int, id string, in model.WorkItemInput) (model.WorkItem, error) {
	var empty model.WorkItem
	assignmentsProvided := in.AssigneeIDs != nil
	if err := ValidateWorkItem(&in); err != nil {
		return empty, err
	}
	manager, err := db.WorkspaceAccess(ctx, userID, division)
	if err != nil {
		return empty, err
	}
	var old model.WorkItem
	if id != "" {
		old, err = db.WorkItem(ctx, userID, id)
		if err != nil {
			return empty, err
		}
		if old.WorkspaceID != division || (!old.CanEdit && !in.PriorityOnly && !in.AssigneesOnly && !in.QAOnly) {
			return empty, ErrWorkspaceForbidden
		}
		if in.PriorityOnly || in.QAOnly || in.AssigneesOnly {
			// Internal flags are never accepted from JSON; guard other fields here too.
			a, b := in, model.WorkItemInput{Title: old.Title, Description: old.Description, RelatedURL: old.RelatedURL, JobID: old.JobID, StartDate: old.StartDate, DueDate: old.DueDate, Status: old.Status, Priority: old.Priority, RequiresQA: old.RequiresQA, QAStage: old.QAStage}
			a.Version = 0
			a.PriorityOnly = false
			a.AssigneesOnly = false
			a.QAOnly = false
			a.AssigneeIDs = nil
			a.RequestID = ""
			if in.PriorityOnly {
				a.Priority = b.Priority
			} else if in.QAOnly {
				a.Status = b.Status
				a.QAStage = b.QAStage
			}
			x, _ := json.Marshal(a)
			y, _ := json.Marshal(b)
			if string(x) != string(y) {
				return empty, ErrWorkspaceForbidden
			}
		}
	}
	if err = validateQATransition(old, in, old.CanCheckQA); err != nil {
		return empty, err
	}
	if in.AssigneeIDs == nil {
		if id == "" {
			if !manager {
				in.AssigneeIDs = []string{userID}
			}
		} else {
			for _, a := range old.Assignees {
				in.AssigneeIDs = append(in.AssigneeIDs, a.ID)
			}
		}
	}
	previous := []string{}
	for _, a := range old.Assignees {
		previous = append(previous, a.ID)
	}
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return empty, err
	}
	defer tx.Rollback()
	if id == "" && in.RequestID != "" {
		// Serialize retries of this creation only. A reused key returns its original task.
		if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, userID+in.RequestID); err != nil {
			return empty, err
		}
		var existing string
		err = tx.QueryRowContext(ctx, `SELECT id FROM work_items WHERE created_by=$1 AND creation_request_id=$2 AND division_id=$3`, userID, in.RequestID, division).Scan(&existing)
		if err == nil {
			_ = tx.Rollback()
			return db.WorkItem(ctx, userID, existing)
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return empty, err
		}
	}
	if id != "" {
		var version int
		err = tx.QueryRowContext(ctx, `SELECT version FROM work_items WHERE id=$1 AND archived_at IS NULL FOR UPDATE`, id).Scan(&version)
		if err != nil {
			return empty, err
		}
		if version != in.Version || version != old.Version {
			return empty, ErrWorkspaceConflict
		}
	}
	if in.JobID != nil {
		var ok bool
		err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM jobs WHERE id=$1 AND division_id=$2 AND (is_active OR id=$3))`, *in.JobID, division, old.JobID).Scan(&ok)
		if err != nil {
			return empty, err
		}
		if !ok {
			return empty, ErrWorkspaceInvalid
		}
	}
	for _, a := range in.AssigneeIDs {
		if slices.Contains(previous, a) && !assignmentsProvided {
			continue
		} // Existing assignments survive transfers; no new out-of-division assignment is allowed.
		var ok bool
		err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members WHERE user_id=$1 AND division_id=$2)`, a, division).Scan(&ok)
		if err != nil {
			return empty, err
		}
		if !ok {
			return empty, ErrWorkspaceForbidden
		}
	}
	if id == "" {
		id = NewID()
		_, err = tx.ExecContext(ctx, `INSERT INTO work_items(id,division_id,job_id,title,description,related_url,start_date,due_date,status,priority,created_by,creation_request_id,requires_qa,qa_stage) VALUES($1,$2,$3,$4,$5,$6,NULLIF($7,'')::date,NULLIF($8,'')::date,$9,$10,$11,NULLIF($12,'')::uuid,$13,$14)`, id, division, in.JobID, in.Title, in.Description, in.RelatedURL, in.StartDate, in.DueDate, in.Status, in.Priority, userID, in.RequestID, in.RequiresQA, in.QAStage)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE work_items SET job_id=$2,title=$3,description=$4,related_url=$5,start_date=NULLIF($6,'')::date,due_date=NULLIF($7,'')::date,status=$8,priority=$9,requires_qa=$10,qa_stage=$11,updated_at=NOW(),version=version+1 WHERE id=$1 AND version=$12`, id, in.JobID, in.Title, in.Description, in.RelatedURL, in.StartDate, in.DueDate, in.Status, in.Priority, in.RequiresQA, in.QAStage, in.Version)
	}
	if err != nil {
		return empty, err
	}
	if in.QAStage == "COMPLETE" && old.QAValidatedAt == nil && old.Status != model.WorkDone {
		if _, err = tx.ExecContext(ctx, `UPDATE work_items SET qa_validated_at=NOW(),qa_validated_by=$2 WHERE id=$1`, id, userID); err != nil {
			return empty, err
		}
		if err = recordWorkActivity(ctx, tx, id, userID, "UPDATE", "qaStage", old.QAStage, "COMPLETE"); err != nil {
			return empty, err
		}
	}
	for _, a := range previous {
		if !slices.Contains(in.AssigneeIDs, a) {
			if _, err = tx.ExecContext(ctx, `DELETE FROM work_item_assignees WHERE work_item_id=$1 AND user_id=$2`, id, a); err != nil {
				return empty, err
			}
			if err = recordWorkActivity(ctx, tx, id, userID, "UNASSIGN", "assignees", a, nil); err != nil {
				return empty, err
			}
		}
	}
	added := []string{}
	for _, a := range in.AssigneeIDs {
		if !slices.Contains(previous, a) {
			if _, err = tx.ExecContext(ctx, `INSERT INTO work_item_assignees(work_item_id,user_id,assigned_by) VALUES($1,$2,$3)`, id, a, userID); err != nil {
				return empty, err
			}
			if err = recordWorkActivity(ctx, tx, id, userID, "ASSIGN", "assignees", nil, a); err != nil {
				return empty, err
			}
			added = append(added, a)
		}
	}
	fieldsChanged := in.QAStage == "COMPLETE" && old.QAValidatedAt == nil && old.Status != model.WorkDone
	if old.ID == "" {
		err = recordWorkActivity(ctx, tx, id, userID, "CREATE", "", nil, in.Title)
	} else {
		for _, change := range []struct {
			action, field string
			old, new      any
		}{{"UPDATE", "title", old.Title, in.Title}, {"UPDATE", "description", old.Description, in.Description}, {"UPDATE", "relatedUrl", old.RelatedURL, in.RelatedURL}, {"UPDATE", "jobId", old.JobID, in.JobID}, {"STATUS_CHANGE", "status", old.Status, in.Status}, {"PRIORITY_CHANGE", "priority", old.Priority, in.Priority}, {"DATE_CHANGE", "startDate", old.StartDate, in.StartDate}, {"DATE_CHANGE", "dueDate", old.DueDate, in.DueDate}} {
			a, _ := json.Marshal(change.old)
			b, _ := json.Marshal(change.new)
			if string(a) != string(b) {
				fieldsChanged = true
				err = recordWorkActivity(ctx, tx, id, userID, change.action, change.field, change.old, change.new)
				if err != nil {
					return empty, err
				}
			}
		}
	}
	if err != nil {
		return empty, err
	}
	if len(added) > 0 {
		err = workAssignmentNotification(ctx, tx, id, userID, added)
	} else if fieldsChanged {
		err = workspaceNotification(ctx, tx, id, userID, "Pekerjaan diperbarui")
	}
	if err != nil {
		return empty, err
	}
	if err = tx.Commit(); err != nil {
		return empty, err
	}
	return db.WorkItem(ctx, userID, id)
}
func (db *Database) ArchiveWorkItem(ctx context.Context, userID, id string, version int) error {
	t, err := db.WorkItem(ctx, userID, id)
	if err != nil {
		return err
	}
	if !t.CanDelete {
		return ErrWorkspaceForbidden
	}
	if version != t.Version {
		return ErrWorkspaceConflict
	}
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE work_items SET archived_at=NOW(),updated_at=NOW(),version=version+1 WHERE id=$1 AND version=$2 AND archived_at IS NULL`, id, version)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrWorkspaceConflict
	}
	if err = recordWorkActivity(ctx, tx, id, userID, "ARCHIVE", "", t.Title, nil); err != nil {
		return err
	}
	return tx.Commit()
}

// Stored in the existing feed, with explicit recipients and live membership checks on read.
func workspaceNotification(ctx context.Context, tx *sql.Tx, id, actor, message string, extraRecipients ...string) error {
	notificationID := NewID()
	_, err := tx.ExecContext(ctx, `INSERT INTO notifications(id,type,actor_user_id,entity_type,entity_id,workspace_division_id,data)
 SELECT $1,'workspace_task',$2,'work_item',id,division_id,jsonb_build_object('title',title,'workspaceId',division_id,'message',$4::text) FROM work_items WHERE id=$3`, notificationID, actor, id, message)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO notification_recipients(notification_id,user_id)
 SELECT $1,a.user_id FROM (SELECT user_id FROM work_item_assignees WHERE work_item_id=$2 UNION SELECT created_by FROM work_items WHERE id=$2) a
 WHERE a.user_id<>$3`, notificationID, id, actor)
	if err != nil {
		return err
	}
	for _, recipient := range extraRecipients {
		if recipient == actor {
			continue
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO notification_recipients(notification_id,user_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, notificationID, recipient); err != nil {
			return err
		}
	}
	return queueWorkNotificationEmail(ctx, tx, notificationID)
}

func (db *Database) WorkComments(ctx context.Context, userID, id string) ([]model.WorkComment, error) {
	if _, err := db.WorkItem(ctx, userID, id); err != nil {
		return nil, err
	}
	rows, err := db.Conn.QueryContext(ctx, `SELECT c.id,COALESCE(c.user_id::text,''),COALESCE(u.name,''),c.content,c.created_at FROM work_item_comments c LEFT JOIN users u ON u.id=c.user_id WHERE c.work_item_id=$1 ORDER BY c.created_at,c.id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.WorkComment{}
	for rows.Next() {
		var c model.WorkComment
		if err = rows.Scan(&c.ID, &c.UserID, &c.Name, &c.Content, &c.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}
func (db *Database) AddWorkComment(ctx context.Context, userID, id, content string, mentionIDs []string) error {
	if strings.TrimSpace(content) == "" || len(content) > 4000 || len(mentionIDs) > 20 {
		return ErrWorkspaceInvalid
	}
	task, err := db.WorkItem(ctx, userID, id)
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, mention := range mentionIDs {
		if !workspaceUUID(mention) || seen[mention] {
			return ErrWorkspaceInvalid
		}
		seen[mention] = true
		var name string
		if err = db.Conn.QueryRowContext(ctx, `SELECT u.name FROM workspace_members m JOIN users u ON u.id=m.user_id WHERE m.division_id=$1 AND m.user_id=$2 AND u.is_active`, task.WorkspaceID, mention).Scan(&name); errors.Is(err, sql.ErrNoRows) {
			return ErrWorkspaceForbidden
		} else if err != nil {
			return err
		}
		if !strings.Contains(content, "@"+name) {
			return ErrWorkspaceInvalid
		}
	}
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var locked string
	if err = tx.QueryRowContext(ctx, `SELECT id FROM work_items WHERE id=$1 AND archived_at IS NULL FOR UPDATE`, id).Scan(&locked); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO work_item_comments(id,work_item_id,user_id,content) VALUES($1,$2,$3,$4)`, NewID(), id, userID, strings.TrimSpace(content)); err != nil {
		return err
	}
	if err = recordWorkActivity(ctx, tx, id, userID, "COMMENT", "", nil, "Komentar ditambahkan"); err != nil {
		return err
	}
	if err = workspaceNotification(ctx, tx, id, userID, "Komentar baru", mentionIDs...); err != nil {
		return err
	}
	return tx.Commit()
}
func (db *Database) WorkActivity(ctx context.Context, userID string, division int, id string) ([]model.WorkActivity, error) {
	if _, err := db.WorkspaceAccess(ctx, userID, division); err != nil {
		return nil, err
	}
	if id != "" {
		t, err := db.WorkItem(ctx, userID, id)
		if err != nil {
			return nil, err
		}
		if t.WorkspaceID != division {
			return nil, ErrWorkspaceForbidden
		}
	}
	rows, err := db.Conn.QueryContext(ctx, `SELECT a.id,a.work_item_id,COALESCE(u.name,''),a.action,a.field_name,COALESCE(a.old_value,'null'),COALESCE(a.new_value,'null'),a.created_at
 FROM work_item_activity a JOIN work_items t ON t.id=a.work_item_id LEFT JOIN users u ON u.id=a.actor_user_id WHERE t.division_id=$1 AND ($2='' OR t.id=NULLIF($2,'')::uuid) AND ($2<>'' OR a.created_at >= $3) ORDER BY a.created_at DESC,a.id DESC LIMIT 100`, division, id, db.Now().Add(-7*24*time.Hour))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.WorkActivity{}
	for rows.Next() {
		var a model.WorkActivity
		if err = rows.Scan(&a.ID, &a.TaskID, &a.ActorName, &a.Action, &a.Field, &a.OldValue, &a.NewValue, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("activity: %w", err)
		}
		items = append(items, a)
	}
	return items, rows.Err()
}
