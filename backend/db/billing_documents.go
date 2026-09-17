package db

import (
	"context"
	"database/sql"
	"my-ple-backend/model"
)

func (db *Database) BillingStages(ctx context.Context) ([]model.BillingStage, error) {
	tx, err := db.Conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	stages := []model.BillingStage{}
	rows, err := tx.QueryContext(ctx, `SELECT id,job_id,name,period_label,notes,version,updated_at FROM billing_stages ORDER BY job_id,sort_order,id`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var s model.BillingStage
		if err = rows.Scan(&s.ID, &s.JobID, &s.Name, &s.PeriodLabel, &s.Notes, &s.Version, &s.UpdatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		s.Milestones = []model.BillingMilestone{}
		stages = append(stages, s)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for i := range stages {
		s := &stages[i]
		rows, err = tx.QueryContext(ctx, `SELECT id,name,notes,updated_at FROM billing_milestones WHERE stage_id=$1 AND archived_at IS NULL ORDER BY sort_order,id`, s.ID)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var m model.BillingMilestone
			if err = rows.Scan(&m.ID, &m.Name, &m.Notes, &m.UpdatedAt); err != nil {
				rows.Close()
				return nil, err
			}
			m.Documents = []model.BillingDocument{}
			s.Milestones = append(s.Milestones, m)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
		for j := range s.Milestones {
			m := &s.Milestones[j]
			rows, err = tx.QueryContext(ctx, `SELECT id,name,status,updated_at FROM billing_documents WHERE milestone_id=$1 AND archived_at IS NULL ORDER BY sort_order,id`, m.ID)
			if err != nil {
				return nil, err
			}
			for rows.Next() {
				var d model.BillingDocument
				if err = rows.Scan(&d.ID, &d.Name, &d.Status, &d.UpdatedAt); err != nil {
					rows.Close()
					return nil, err
				}
				m.Documents = append(m.Documents, d)
			}
			err = rows.Err()
			rows.Close()
			if err != nil {
				return nil, err
			}
		}
		s.Calculate()
	}
	return stages, tx.Commit()
}
func lockBilling(ctx context.Context, tx *sql.Tx, id, version int) error {
	var current int
	if err := tx.QueryRowContext(ctx, `SELECT version FROM billing_stages WHERE id=$1 FOR UPDATE`, id).Scan(&current); err != nil {
		return err
	}
	if current != version {
		return model.ErrBillingConflict
	}
	return nil
}
func (db *Database) ConfigureBilling(ctx context.Context, id int, c model.BillingConfiguration) (int, error) {
	if err := c.Validate(); err != nil {
		return 0, err
	}
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if id == 0 {
		if c.Version != 0 {
			return 0, model.ErrBillingInvalid
		}
		var job int
		if err = tx.QueryRowContext(ctx, `SELECT id FROM jobs WHERE id=$1 AND is_active FOR SHARE`, c.JobID).Scan(&job); err != nil {
			return 0, err
		}
		err = tx.QueryRowContext(ctx, `INSERT INTO billing_stages(job_id,name,period_label) VALUES($1,$2,$3) RETURNING id`, c.JobID, c.Name, c.PeriodLabel).Scan(&id)
	} else {
		if err = lockBilling(ctx, tx, id, c.Version); err != nil {
			return 0, err
		}
		var job int
		if err = tx.QueryRowContext(ctx, `SELECT job_id FROM billing_stages WHERE id=$1`, id).Scan(&job); err != nil {
			return 0, err
		}
		if job != c.JobID {
			return 0, model.ErrBillingInvalid
		}
		_, err = tx.ExecContext(ctx, `UPDATE billing_stages SET name=$2,period_label=$3,version=version+1,updated_at=now() WHERE id=$1`, id, c.Name, c.PeriodLabel)
	}
	if err != nil {
		return 0, err
	}
	// Archive omitted requirements; validate existing IDs against the original active hierarchy first.
	existing := map[int]map[int]bool{}
	rows, err := tx.QueryContext(ctx, `SELECT m.id,d.id FROM billing_milestones m LEFT JOIN billing_documents d ON d.milestone_id=m.id AND d.archived_at IS NULL WHERE m.stage_id=$1 AND m.archived_at IS NULL`, id)
	if err != nil {
		return 0, err
	}
	for rows.Next() {
		var m int
		var d sql.NullInt64
		if err = rows.Scan(&m, &d); err != nil {
			rows.Close()
			return 0, err
		}
		if existing[m] == nil {
			existing[m] = map[int]bool{}
		}
		if d.Valid {
			existing[m][int(d.Int64)] = true
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return 0, err
	}
	for _, m := range c.Milestones {
		if m.ID > 0 && existing[m.ID] == nil {
			return 0, model.ErrBillingInvalid
		}
		for _, d := range m.Documents {
			if d.ID > 0 && !existing[m.ID][d.ID] {
				return 0, model.ErrBillingInvalid
			}
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE billing_documents SET archived_at=now(),updated_at=now() WHERE milestone_id IN (SELECT id FROM billing_milestones WHERE stage_id=$1 AND archived_at IS NULL) AND archived_at IS NULL`, id); err != nil {
		return 0, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE billing_milestones SET archived_at=now(),updated_at=now() WHERE stage_id=$1 AND archived_at IS NULL`, id); err != nil {
		return 0, err
	}
	for i, m := range c.Milestones {
		if m.ID == 0 {
			err = tx.QueryRowContext(ctx, `INSERT INTO billing_milestones(stage_id,name,sort_order) VALUES($1,$2,$3) RETURNING id`, id, m.Name, i).Scan(&m.ID)
		} else {
			_, err = tx.ExecContext(ctx, `UPDATE billing_milestones SET name=$2,sort_order=$3,archived_at=NULL WHERE id=$1`, m.ID, m.Name, i)
		}
		if err != nil {
			return 0, err
		}
		for j, d := range m.Documents {
			if d.ID == 0 {
				_, err = tx.ExecContext(ctx, `INSERT INTO billing_documents(milestone_id,name,sort_order) VALUES($1,$2,$3)`, m.ID, d.Name, j)
			} else {
				_, err = tx.ExecContext(ctx, `UPDATE billing_documents SET name=$2,sort_order=$3,archived_at=NULL WHERE id=$1`, d.ID, d.Name, j)
			}
			if err != nil {
				return 0, err
			}
		}
	}
	return id, tx.Commit()
}
func (db *Database) PatchBilling(ctx context.Context, id int, p model.BillingPatch) error {
	if err := p.Validate(); err != nil {
		return err
	}
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = lockBilling(ctx, tx, id, p.Version); err != nil {
		return err
	}
	var result sql.Result
	if p.Status != nil {
		var mid int
		err = tx.QueryRowContext(ctx, `UPDATE billing_documents d SET status=$3,updated_at=now() FROM billing_milestones m WHERE d.id=$2 AND d.milestone_id=m.id AND m.stage_id=$1 AND m.archived_at IS NULL AND d.archived_at IS NULL RETURNING m.id`, id, p.DocumentID, *p.Status).Scan(&mid)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE billing_milestones SET updated_at=now() WHERE id=$1`, mid)
	} else if p.MilestoneID > 0 {
		result, err = tx.ExecContext(ctx, `UPDATE billing_milestones SET notes=$3,updated_at=now() WHERE stage_id=$1 AND id=$2 AND archived_at IS NULL`, id, p.MilestoneID, *p.Notes)
		if err == nil {
			n, e := result.RowsAffected()
			if e != nil {
				return e
			}
			if n == 0 {
				return sql.ErrNoRows
			}
		}
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE billing_stages SET notes=$2 WHERE id=$1`, id, *p.Notes)
	}
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE billing_stages SET version=version+1,updated_at=now() WHERE id=$1`, id); err != nil {
		return err
	}
	return tx.Commit()
}
