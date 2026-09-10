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
	in.Status = model.WorkDone
	if err := validateQATransition(old, in, true); err != nil {
		t.Fatal(err)
	}
	in.RequiresQA = false
	if validateQATransition(old, in, true) == nil {
		t.Fatal("QA disabled")
	}
	if validateQATransition(model.WorkItem{}, model.WorkItemInput{Status: model.WorkDone}, false) == nil {
		t.Fatal("created directly done")
	}
	if err := validateQATransition(model.WorkItem{ID: "legacy", Status: model.WorkDone}, model.WorkItemInput{Status: model.WorkDone}, false); err != nil {
		t.Fatal(err)
	}
}
