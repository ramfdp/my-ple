package model

import "testing"

func TestLessonLearnedValidate(t *testing.T) {
	if !CanUseLessonsLearned("Project Engineer") || !CanUseLessonsLearned("Admin") || CanUseLessonsLearned("Finance") {
		t.Fatal("role gate")
	}
	valid := LessonLearned{JobID: 1, Kind: "failure", Title: "  Temuan  ", Status: "resolved", ResolvedOn: "2026-10-16"}
	if err := valid.Validate(); err != nil || valid.Title != "  Temuan  " {
		t.Fatalf("valid lesson: %v, title %q", err, valid.Title)
	}
	imported := LessonLearned{SourceRow: 101, SourceGroup: "NONPENGAWASAN", SourceProject: "Studi Daya Beli Gas Bumi PGN", Kind: "failure", Title: "  Temuan  ", Status: "open"}
	if err := imported.Validate(); err != nil {
		t.Fatalf("imported lesson: %v", err)
	}
	for name, change := range map[string]func(*LessonLearned){
		"empty title":             func(l *LessonLearned) { l.Title = " " },
		"invalid kind":            func(l *LessonLearned) { l.Kind = "other" },
		"invalid date":            func(l *LessonLearned) { l.ResolvedOn = "2026-02-30" },
		"missing resolution date": func(l *LessonLearned) { l.ResolvedOn = "" },
		"success with status":     func(l *LessonLearned) { l.Kind = "success" },
	} {
		t.Run(name, func(t *testing.T) {
			l := valid
			change(&l)
			if err := l.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
