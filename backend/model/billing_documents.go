package model

import (
	"errors"
	"slices"
	"strings"
	"time"
)

var BillingStatuses = []string{"BELUM ADA", "PEMBUATAN", "KURANG LENGKAP", "SIRKULIR TTD", "OK"}
var ErrBillingInvalid = errors.New("Data penagihan tidak valid.")
var ErrBillingConflict = errors.New("Data penagihan sudah berubah. Muat ulang dan bandingkan draf Anda.")

func CanUseBilling(role string) bool {
	return slices.Contains([]string{"admin", "finance", "project engineer", "management"}, strings.ToLower(role))
}

type BillingDocument struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updatedAt"`
}
type BillingMilestone struct {
	ID        int               `json:"id"`
	Name      string            `json:"name"`
	Notes     string            `json:"notes"`
	UpdatedAt time.Time         `json:"updatedAt"`
	Documents []BillingDocument `json:"documents"`
	Readiness string            `json:"readiness"`
}
type BillingStage struct {
	ID          int                `json:"id"`
	JobID       int                `json:"jobId"`
	Name        string             `json:"name"`
	PeriodLabel string             `json:"periodLabel"`
	Notes       string             `json:"notes"`
	UpdatedAt   time.Time          `json:"updatedAt"`
	Version     int                `json:"version"`
	Milestones  []BillingMilestone `json:"milestones"`
	Readiness   string             `json:"readiness"`
}

func (s *BillingStage) Calculate() {
	s.Readiness = "SIAP"
	if len(s.Milestones) == 0 {
		s.Readiness = "BELUM SIAP"
	}
	for i := range s.Milestones {
		m := &s.Milestones[i]
		m.Readiness = "SIAP"
		if len(m.Documents) == 0 {
			m.Readiness = "BELUM SIAP"
		}
		for _, d := range m.Documents {
			if d.Status != "OK" {
				m.Readiness = "BELUM SIAP"
			}
		}
		if m.Readiness != "SIAP" {
			s.Readiness = "BELUM SIAP"
		}
	}
}

type BillingRequirement struct {
	ID        int                  `json:"id"`
	Name      string               `json:"name"`
	Documents []BillingRequirement `json:"documents,omitempty"`
}
type BillingConfiguration struct {
	JobID       int                  `json:"jobId"`
	Name        string               `json:"name"`
	PeriodLabel string               `json:"periodLabel"`
	Version     int                  `json:"version"`
	Milestones  []BillingRequirement `json:"milestones"`
}

func (c *BillingConfiguration) Validate() error {
	c.Name = strings.TrimSpace(c.Name)
	c.PeriodLabel = strings.TrimSpace(c.PeriodLabel)
	validName := func(s string) bool { return s != "" && len([]rune(s)) <= 255 }
	if c.JobID <= 0 || c.Version < 0 || !validName(c.Name) || !validName(c.PeriodLabel) || len(c.Milestones) == 0 {
		return ErrBillingInvalid
	}
	ids := map[int]bool{}
	docs := map[int]bool{}
	for i := range c.Milestones {
		m := &c.Milestones[i]
		m.Name = strings.TrimSpace(m.Name)
		if m.ID < 0 || !validName(m.Name) || len(m.Documents) == 0 || m.ID > 0 && ids[m.ID] {
			return ErrBillingInvalid
		}
		ids[m.ID] = true
		names := map[string]bool{}
		for j := range m.Documents {
			d := &m.Documents[j]
			d.Name = strings.TrimSpace(d.Name)
			key := strings.ToLower(d.Name)
			if d.ID < 0 || !validName(d.Name) || len(d.Documents) > 0 || names[key] || d.ID > 0 && docs[d.ID] {
				return ErrBillingInvalid
			}
			docs[d.ID] = true
			names[key] = true
		}
	}
	return nil
}

type BillingPatch struct {
	Version     int     `json:"version"`
	DocumentID  int     `json:"documentId"`
	MilestoneID int     `json:"milestoneId"`
	Status      *string `json:"status"`
	Notes       *string `json:"notes"`
}

func (p BillingPatch) Validate() error {
	if p.Version < 1 || p.DocumentID < 0 || p.MilestoneID < 0 {
		return ErrBillingInvalid
	}
	if p.Status != nil {
		if p.DocumentID <= 0 || p.MilestoneID != 0 || p.Notes != nil || !slices.Contains(BillingStatuses, *p.Status) {
			return ErrBillingInvalid
		}
	} else if p.Notes == nil || p.DocumentID != 0 || len([]rune(*p.Notes)) > 4000 {
		return ErrBillingInvalid
	}
	return nil
}
