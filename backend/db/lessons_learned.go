package db

import (
	"context"
	"database/sql"
	"my-ple-backend/model"
)

const lessonColumns = `id,job_id,kind,COALESCE(to_char(found_on,'YYYY-MM-DD'),''),title,location,internal_factor,external_factor,impact,action,status,notes,COALESCE(to_char(resolved_on,'YYYY-MM-DD'),''),version`

func scanLesson(row interface{ Scan(...any) error }) (model.LessonLearned, error) {
	var l model.LessonLearned
	err := row.Scan(&l.ID, &l.JobID, &l.Kind, &l.FoundOn, &l.Title, &l.Location, &l.InternalFactor, &l.ExternalFactor, &l.Impact, &l.Action, &l.Status, &l.Notes, &l.ResolvedOn, &l.Version)
	return l, err
}

func (db *Database) LessonsLearned(ctx context.Context) ([]model.LessonLearned, error) {
	rows, err := db.Conn.QueryContext(ctx, `SELECT `+lessonColumns+` FROM lessons_learned WHERE deleted_at IS NULL ORDER BY job_id,kind,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.LessonLearned{}
	for rows.Next() {
		l, err := scanLesson(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, l)
	}
	return items, rows.Err()
}

func (db *Database) SaveLessonLearned(ctx context.Context, id int, l model.LessonLearned) (model.LessonLearned, error) {
	if err := l.Validate(); err != nil {
		return l, err
	}
	var row *sql.Row
	if id == 0 {
		row = db.Conn.QueryRowContext(ctx, `INSERT INTO lessons_learned(job_id,kind,found_on,title,location,internal_factor,external_factor,impact,action,status,notes,resolved_on)
   VALUES($1,$2,NULLIF($3,'')::date,$4,$5,$6,$7,$8,$9,$10,$11,NULLIF($12,'')::date) RETURNING `+lessonColumns,
			l.JobID, l.Kind, l.FoundOn, l.Title, l.Location, l.InternalFactor, l.ExternalFactor, l.Impact, l.Action, l.Status, l.Notes, l.ResolvedOn)
	} else {
		row = db.Conn.QueryRowContext(ctx, `UPDATE lessons_learned SET job_id=$2,kind=$3,found_on=NULLIF($4,'')::date,title=$5,location=$6,internal_factor=$7,external_factor=$8,impact=$9,action=$10,status=$11,notes=$12,resolved_on=NULLIF($13,'')::date,version=version+1,updated_at=NOW()
   WHERE id=$1 AND version=$14 AND deleted_at IS NULL RETURNING `+lessonColumns,
			id, l.JobID, l.Kind, l.FoundOn, l.Title, l.Location, l.InternalFactor, l.ExternalFactor, l.Impact, l.Action, l.Status, l.Notes, l.ResolvedOn, l.Version)
	}
	saved, err := scanLesson(row)
	if err == sql.ErrNoRows && id != 0 {
		return saved, model.ErrLessonConflict
	}
	return saved, err
}

func (db *Database) DeleteLessonLearned(ctx context.Context, id, version int) error {
	result, err := db.Conn.ExecContext(ctx, `UPDATE lessons_learned SET deleted_at=NOW(),updated_at=NOW(),version=version+1 WHERE id=$1 AND version=$2 AND deleted_at IS NULL`, id, version)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return model.ErrLessonConflict
	}
	return nil
}
