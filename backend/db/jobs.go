package db

import (
	"context"
	"database/sql"
	"errors"
	"my-ple-backend/model"
	"strings"
)

const jobColumns = `id,division_id,role_id,job_category_id,name,description,is_active,sort_order,division_name,role_name,category_name,used_at IS NOT NULL,(SELECT pics FROM job_pic_summary WHERE job_id=job_context.id)`

func scanJob(row interface{ Scan(...any) error }) (model.Job, error) {
	var j model.Job
	err := row.Scan(&j.ID, &j.DivisionID, &j.RoleID, &j.JobCategoryID, &j.Name, &j.Description, &j.IsActive, &j.SortOrder, &j.Division, &j.Role, &j.Category, &j.Referenced, &j.PICs)
	return j, err
}

func normalizeJob(j model.Job) model.Job {
	r := model.WeeklyReport{Role: j.Role, WorkDivision: j.Category, Work: j.Name}
	model.NormalizeWeeklyReport(&r)
	j.Role, j.Category, j.Name = r.Role, r.WorkDivision, r.Work
	return j
}

func (db *Database) ListJobs(ctx context.Context, division, role, category *int, active *bool) ([]model.Job, error) {
	rows, err := db.Conn.QueryContext(ctx, `SELECT `+jobColumns+` FROM job_context WHERE ($1::int IS NULL OR division_id=$1) AND ($2::int IS NULL OR COALESCE(role_id,0)=$2) AND ($3::int IS NULL OR COALESCE(job_category_id,0)=$3) AND ($4::boolean IS NULL OR is_active=$4) ORDER BY division_id,role_id NULLS FIRST,job_category_id NULLS FIRST,sort_order,id`, division, role, category, active)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	jobs := []model.Job{}
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, normalizeJob(j))
	}
	return jobs, rows.Err()
}

func (db *Database) Job(ctx context.Context, id int) (model.Job, error) {
	j, err := scanJob(db.Conn.QueryRowContext(ctx, `SELECT `+jobColumns+` FROM job_context WHERE id=$1`, id))
	return normalizeJob(j), err
}

func (db *Database) JobMasters(ctx context.Context) (map[string][]model.JobMaster, error) {
	result := map[string][]model.JobMaster{"divisions": {}, "roles": {}, "categories": {}}
	rows, err := db.Conn.QueryContext(ctx, `SELECT 'divisions',id,name,NULL::int,NULL::int,sort_order FROM divisions UNION ALL SELECT 'roles',id,name,division_id,NULL::int,sort_order FROM report_roles UNION ALL SELECT 'categories',id,name,division_id,role_id,sort_order FROM job_categories ORDER BY 1,6,2`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var kind string
		var m model.JobMaster
		var order int
		if err := rows.Scan(&kind, &m.ID, &m.Name, &m.DivisionID, &m.RoleID, &order); err != nil {
			return nil, err
		}
		result[kind] = append(result[kind], m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	roles := map[int]string{}
	for i := range result["roles"] {
		role, _ := model.CanonicalWeeklyRoleCategory(result["roles"][i].Name, "")
		result["roles"][i].Name, roles[result["roles"][i].ID] = role, role
	}
	for i := range result["categories"] {
		role := ""
		if id := result["categories"][i].RoleID; id != nil {
			role = roles[*id]
		}
		_, result["categories"][i].Name = model.CanonicalWeeklyRoleCategory(role, result["categories"][i].Name)
	}
	return result, nil
}

func (db *Database) SaveJob(ctx context.Context, id int, j model.Job, userID string) (model.Job, error) {
	j.Name = strings.TrimSpace(j.Name)
	j.Description = strings.TrimSpace(j.Description)
	if j.Name == "" || len([]rune(j.Name)) > 255 || len([]rune(j.Description)) > 2000 || j.DivisionID <= 0 || j.SortOrder < 0 {
		return j, ErrInvalidReport
	}
	var err error
	if id == 0 {
		err = db.Conn.QueryRowContext(ctx, `INSERT INTO jobs(division_id,role_id,job_category_id,name,description,is_active,sort_order,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`, j.DivisionID, j.RoleID, j.JobCategoryID, j.Name, j.Description, j.IsActive, j.SortOrder, userID).Scan(&id)
	} else {
		err = db.Conn.QueryRowContext(ctx, `UPDATE jobs SET division_id=$2,role_id=$3,job_category_id=$4,name=$5,description=$6,is_active=$7,sort_order=$8,updated_at=NOW() WHERE id=$1 RETURNING id`, id, j.DivisionID, j.RoleID, j.JobCategoryID, j.Name, j.Description, j.IsActive, j.SortOrder).Scan(&id)
	}
	if err != nil {
		return j, err
	}
	return db.Job(ctx, id)
}

func (db *Database) SetJobStatus(ctx context.Context, id int, active bool) (model.Job, error) {
	var found int
	err := db.Conn.QueryRowContext(ctx, `UPDATE jobs SET is_active=$2,updated_at=NOW() WHERE id=$1 RETURNING id`, id, active).Scan(&found)
	if err != nil {
		return model.Job{}, err
	}
	return db.Job(ctx, id)
}

func (db *Database) DeleteJob(ctx context.Context, id int) error {
	var found int
	return db.Conn.QueryRowContext(ctx, `DELETE FROM jobs WHERE id=$1 RETURNING id`, id).Scan(&found)
}

// Resolve at the write boundary, inside the report transaction. Locking serializes
// selection with deactivation, relocation and deletion. Legacy clients resolve by full context.
func resolveReportJob(ctx context.Context, tx *sql.Tx, input *model.WeeklyReportInput, old *model.WeeklyReport) error {
	if strings.TrimSpace(input.Description) == "" {
		return ErrInvalidReport
	}
	input.Role, input.WorkDivision = model.NormalizeWeeklyRoleCategory(input.Role, input.WorkDivision)
	if old != nil && input.JobID == nil && old.JobID == nil && input.PrimaryDivision == old.PrimaryDivision && input.Role == old.Role && input.WorkDivision == old.WorkDivision && input.Work == old.Work {
		return nil
	}
	var id int
	if input.JobID != nil {
		id = *input.JobID
	} else {
		var matched sql.NullInt64
		if err := tx.QueryRowContext(ctx, `SELECT legacy_job_match($1,$2,$3,$4)`, input.PrimaryDivision, input.Role, input.WorkDivision, input.Work).Scan(&matched); err != nil {
			return err
		}
		if !matched.Valid {
			return ErrInvalidReport
		}
		id = int(matched.Int64)
	}
	if err := tx.QueryRowContext(ctx, `SELECT id FROM jobs WHERE id=$1 FOR UPDATE`, id).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInvalidReport
		}
		return err
	}
	j, err := scanJob(tx.QueryRowContext(ctx, `SELECT `+jobColumns+` FROM job_context WHERE id=$1`, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInvalidReport
		}
		return err
	}
	same := old != nil && old.JobID != nil && *old.JobID == id
	inputRole, inputCategory := model.CanonicalWeeklyRoleCategory(input.Role, input.WorkDivision)
	jobRole, jobCategory := model.CanonicalWeeklyRoleCategory(j.Role, j.Category)
	if (!j.IsActive && !same) || input.PrimaryDivision != j.Division || inputRole != jobRole || inputCategory != jobCategory {
		return ErrInvalidReport
	}
	input.JobID = &id
	input.Role, input.WorkDivision = j.Role, j.Category
	input.Work = j.Name
	_, err = tx.ExecContext(ctx, `UPDATE jobs SET used_at=COALESCE(used_at,NOW()) WHERE id=$1`, id)
	return err
}
