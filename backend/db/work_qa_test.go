package db

import (
	"my-ple-backend/model"
	"testing"
)

func TestQATransitions(t *testing.T) {
	old := model.WorkItem{ID: "task", RequiresQA: true, Status: model.WorkInProgress}
	in := model.WorkItemInput{RequiresQA: true, Status: model.WorkDone}
	if validateQATransition(old, in, false) == nil {
		t.Fatal("QA bypass")
	}
	in.Status = model.WorkInProgress
	in.QAStage = "WAITING"
	if err := validateQATransition(old, in, false); err != nil {
		t.Fatal(err)
	}
	old.QAStage = "WAITING"
	in.QAStage = "CHECKING"
	if validateQATransition(old, in, false) == nil {
		t.Fatal("non-QA reviewer")
	}
	if err := validateQATransition(old, in, true); err != nil {
		t.Fatal(err)
	}
	old.QAStage = "CHECKING"
	in.QAStage = "COMPLETE"
	in.Status = model.WorkInProgress
	if err := validateQATransition(old, in, true); err != nil {
		t.Fatal(err)
	}
	in.RequiresQA = false
	in.QAStage = ""
	in.Status = model.WorkDone
	if err := validateQATransition(old, in, false); err != nil {
		t.Fatal("QA cannot be disabled", err)
	}
	if validateQATransition(model.WorkItem{}, model.WorkItemInput{Status: model.WorkDone}, false) == nil {
		t.Fatal("created directly done")
	}
	if err := validateQATransition(model.WorkItem{ID: "legacy", Status: model.WorkDone}, model.WorkItemInput{Status: model.WorkDone}, false); err != nil {
		t.Fatal(err)
	}
}

func TestOptionalQAAllowsDirectCompletion(t *testing.T) {
	old := model.WorkItem{ID: "task", Status: model.WorkInProgress}
	in := model.WorkItemInput{Status: model.WorkDone}
	if err := validateQATransition(old, in, false); err != nil {
		t.Fatal(err)
	}
}

func TestQADecisionCompletesOrRevisesTask(t *testing.T) {
	old := model.WorkItem{ID: "task", RequiresQA: true, Status: model.WorkInProgress}
	for _, in := range []model.WorkItemInput{
		{RequiresQA: true, Status: model.WorkDone, QAStage: "COMPLETE", QAOnly: true},
		{RequiresQA: true, Status: model.WorkNeedRevision, QAOnly: true},
	} {
		if err := validateQATransition(old, in, false); err != nil {
			t.Fatal(err)
		}
	}
}

func TestEnableQARequiresValidation(t *testing.T) {
	for _, status := range []string{model.WorkInProgress, model.WorkDone} {
		old := model.WorkItem{ID: "task", Status: status}
		if err := validateQATransition(old, model.WorkItemInput{RequiresQA: true, Status: model.WorkDone}, false); err != ErrQARequired {
			t.Fatalf("enabling QA must require validation: %v", err)
		}
	}
	old := model.WorkItem{ID: "task", Status: model.WorkInProgress}
	if err := validateQATransition(old, model.WorkItemInput{RequiresQA: true, Status: model.WorkInProgress}, false); err != nil {
		t.Fatal(err)
	}
}

func TestWorkAssigneeCanEdit(t *testing.T) {
	task := model.WorkItem{CreatedBy: "creator", Assignees: []model.WorkspaceMember{{ID: "assignee"}}}
	workPermissions(&task, "assignee", false)
	if !task.CanEdit || task.CanDelete {
		t.Fatal("assignee must edit without deletion rights")
	}
	workPermissions(&task, "member", false)
	if task.CanEdit {
		t.Fatal("unassigned member can edit")
	}
}
