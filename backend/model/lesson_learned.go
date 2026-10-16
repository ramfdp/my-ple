package model

import (
	"errors"
	"strings"
	"time"
)

var ErrInvalidLesson = errors.New("Data evaluasi tidak valid.")
var ErrLessonConflict = errors.New("Evaluasi sudah berubah. Muat ulang sebelum menyimpan.")

func CanUseLessonsLearned(role string) bool {
	switch strings.ToLower(role) {
	case "project engineer", "admin", "management":
		return true
	}
	return false
}

type LessonLearned struct {
	ID             int    `json:"id"`
	JobID          int    `json:"jobId"`
	Kind           string `json:"kind"`
	FoundOn        string `json:"foundOn"`
	Title          string `json:"title"`
	Location       string `json:"location"`
	InternalFactor string `json:"internalFactor"`
	ExternalFactor string `json:"externalFactor"`
	Impact         string `json:"impact"`
	Action         string `json:"action"`
	Status         string `json:"status"`
	Notes          string `json:"notes"`
	ResolvedOn     string `json:"resolvedOn"`
	Version        int    `json:"version"`
}

func (l *LessonLearned) Validate() error {
	l.Title = strings.TrimSpace(l.Title)
	if l.JobID <= 0 || (l.Kind != "failure" && l.Kind != "success") || l.Title == "" || len([]rune(l.Title)) > 2000 || l.Version < 0 {
		return ErrInvalidLesson
	}
	if l.Status != "open" && l.Status != "in_progress" && l.Status != "resolved" {
		return ErrInvalidLesson
	}
	for _, value := range []string{l.Location, l.InternalFactor, l.ExternalFactor, l.Impact, l.Action, l.Notes} {
		if len([]rune(value)) > 4000 {
			return ErrInvalidLesson
		}
	}
	for _, value := range []string{l.FoundOn, l.ResolvedOn} {
		if value != "" {
			if _, err := time.Parse("2006-01-02", value); err != nil {
				return ErrInvalidLesson
			}
		}
	}
	if l.Kind == "success" && (l.Status != "open" || l.ResolvedOn != "") {
		return ErrInvalidLesson
	}
	if l.Kind == "failure" && (l.Status == "resolved") != (l.ResolvedOn != "") {
		return ErrInvalidLesson
	}
	return nil
}
