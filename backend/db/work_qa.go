package db

import (
	"context"
	"errors"
	"my-ple-backend/model"
)

// QA identity comes from existing QA job PICs in the same division.
func (db *Database) WorkspaceQA(ctx context.Context, userID string, division int) (bool, error) {
	var allowed bool
	err := db.Conn.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM job_pics p JOIN job_context j ON j.id=p.job_id JOIN workspace_access a ON a.user_id=p.user_id AND a.division_id=j.division_id WHERE p.user_id=$1 AND j.division_id=$2 AND j.role_name='QA')`, userID, division).Scan(&allowed)
	return allowed, err
}

var ErrQARequired = errors.New("Pekerjaan harus divalidasi QA sebelum menjadi Done.")

func validateQATransition(old model.WorkItem, in model.WorkItemInput, checker bool) error {
	if old.ID == "" {
		if in.Status != model.WorkTodo || in.QAStage != "" {
			return ErrWorkspaceInvalid
		}
		return nil
	}
	if in.QAOnly {
		if !old.RequiresQA || (old.Status != model.WorkInProgress && old.Status != model.WorkNeedRevision) {
			return ErrWorkspaceForbidden
		}
		if (in.Status == model.WorkDone && in.QAStage == "COMPLETE") || (in.Status == model.WorkNeedRevision && in.QAStage == "") {
			return nil
		}
		return ErrWorkspaceInvalid
	}
	// Historical Done records are retained, not backfilled with fictitious approvals.
	if in.RequiresQA && in.Status == model.WorkDone && (old.Status != model.WorkDone || !old.RequiresQA) {
		return ErrQARequired
	}
	if !in.RequiresQA && in.QAStage == "" {
		return nil
	}
	if in.QAStage == "COMPLETE" && old.QAValidatedAt == nil && old.Status != model.WorkDone {
		if !checker {
			return ErrWorkspaceForbidden
		}
		if old.Status != model.WorkInProgress || in.Status != model.WorkInProgress {
			return ErrWorkspaceInvalid
		}
	}
	if in.QAStage != old.QAStage {
		switch {
		case old.QAStage == "" && in.QAStage == "WAITING" && old.Status == model.WorkInProgress && in.Status == model.WorkInProgress:
		case old.QAStage == "WAITING" && in.QAStage == "CHECKING" && in.Status == model.WorkInProgress && checker:
		case in.QAStage == "COMPLETE" && old.Status == model.WorkInProgress && in.Status == model.WorkInProgress && checker:
		default:
			return ErrWorkspaceForbidden
		}
	}
	if (in.QAStage == "WAITING" || in.QAStage == "CHECKING") && in.Status != model.WorkInProgress {
		return ErrWorkspaceInvalid
	}
	return nil
}
