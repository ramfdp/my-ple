package db

import (
	"context"
	"my-ple-backend/model"
)

// QA identity comes from existing QA job PICs in the same division.
func (db *Database) WorkspaceQA(ctx context.Context, userID string, division int) (bool, error) {
	var allowed bool
	err := db.Conn.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM job_pics p JOIN job_context j ON j.id=p.job_id JOIN workspace_access a ON a.user_id=p.user_id AND a.division_id=j.division_id WHERE p.user_id=$1 AND j.division_id=$2 AND j.role_name='QA')`, userID, division).Scan(&allowed)
	return allowed, err
}

func validateQATransition(old model.WorkItem, in model.WorkItemInput, checker bool) error {
	if old.ID == "" {
		if in.Status != model.WorkTodo || in.QAStage != "" {
			return ErrWorkspaceInvalid
		}
		return nil
	}
	if old.RequiresQA != in.RequiresQA {
		return ErrWorkspaceInvalid
	} // The creation choice cannot be removed to bypass checking.
	if !in.RequiresQA {
		if in.QAStage != "" {
			return ErrWorkspaceInvalid
		}
		return nil
	}
	if in.QAStage != old.QAStage {
		switch {
		case old.QAStage == "" && in.QAStage == "WAITING" && old.Status == model.WorkInProgress && in.Status == model.WorkInProgress:
		case old.QAStage == "WAITING" && in.QAStage == "CHECKING" && in.Status == model.WorkInProgress && checker:
		case old.QAStage == "CHECKING" && in.QAStage == "COMPLETE" && in.Status == model.WorkDone && checker:
		default:
			return ErrWorkspaceForbidden
		}
	}
	if in.Status == model.WorkDone && in.QAStage != "COMPLETE" {
		return ErrWorkspaceInvalid
	}
	if (in.QAStage == "WAITING" || in.QAStage == "CHECKING") && in.Status != model.WorkInProgress {
		return ErrWorkspaceInvalid
	}
	if in.QAStage == "COMPLETE" && in.Status != model.WorkDone {
		return ErrWorkspaceInvalid
	}
	return nil
}
