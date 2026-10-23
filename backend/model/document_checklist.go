package model

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrInvalidChecklist = errors.New("Data checklist tidak valid. Periksa nama, penandatangan, dan tanggal.")
var ErrChecklistConflict = errors.New("Struktur checklist sudah berubah. Muat ulang sebelum menyimpan.")

func CanUseDocumentChecklist(role string) bool {
	switch strings.ToLower(role) {
	case "project engineer", "finance", "admin", "management":
		return true
	}
	return false
}

type ChecklistSigner struct {
	ID           int    `json:"id"`
	Type         string `json:"type"`
	Name         string `json:"name"`
	Position     string `json:"position"`
	DisplayOrder int    `json:"displayOrder"`
	Archived     bool   `json:"archived"`
}
type ChecklistConfigInput struct {
	Title                   string             `json:"title"`
	RequireDigitalSignature bool               `json:"requireDigitalSignature"`
	Version                 int                `json:"version"`
	InternalCount           int                `json:"internalCount"`
	ExternalCount           int                `json:"externalCount"`
	Signers                 []ChecklistSigner  `json:"signers"`
	NewItems                []ChecklistNewItem `json:"newItems"`
}

type ChecklistNewItem struct {
	ClientID       string `json:"clientId"`
	ParentID       int    `json:"parentId"`
	ParentClientID string `json:"parentClientId"`
	DocumentName   string `json:"documentName"`
}

func (in *ChecklistConfigInput) Validate() error {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" || len([]rune(in.Title)) > 255 {
		return ErrInvalidChecklist
	}
	if in.InternalCount < 0 || in.ExternalCount < 0 || in.Version < 0 {
		return ErrInvalidChecklist
	}
	counts := map[string]int{"internal": 0, "external": 0}
	seen := map[int]bool{}
	for i := range in.Signers {
		s := &in.Signers[i]
		s.Name, s.Position = strings.TrimSpace(s.Name), strings.TrimSpace(s.Position)
		if _, ok := counts[s.Type]; !ok {
			return ErrInvalidChecklist
		}
		if s.ID < 0 || (s.ID > 0 && seen[s.ID]) || s.Archived || s.Name == "" || s.Position == "" || len([]rune(s.Name)) > 255 || len([]rune(s.Position)) > 255 {
			return ErrInvalidChecklist
		}
		seen[s.ID] = true
		s.DisplayOrder = i
		counts[s.Type]++
	}
	if counts["internal"] != in.InternalCount || counts["external"] != in.ExternalCount {
		return ErrInvalidChecklist
	}
	drafts := map[string]bool{}
	for i := range in.NewItems {
		item := &in.NewItems[i]
		item.ClientID, item.ParentClientID, item.DocumentName = strings.TrimSpace(item.ClientID), strings.TrimSpace(item.ParentClientID), strings.TrimSpace(item.DocumentName)
		if item.ClientID == "" || len(item.ClientID) > 100 || drafts[item.ClientID] || item.DocumentName == "" || len([]rune(item.DocumentName)) > 255 || item.ParentID < 0 || (item.ParentID > 0 && item.ParentClientID != "") || (item.ParentClientID != "" && !drafts[item.ParentClientID]) {
			return ErrInvalidChecklist
		}
		drafts[item.ClientID] = true
	}
	return nil
}

type DocumentChecklist struct {
	Title                   string            `json:"title"`
	HeaderLabels            map[string]string `json:"headerLabels"`
	ID                      int               `json:"id"`
	JobID                   int               `json:"jobId"`
	RequireDigitalSignature bool              `json:"requireDigitalSignature"`
	Version                 int               `json:"version"`
	Signers                 []ChecklistSigner `json:"signers"`
	Items                   []ChecklistItem   `json:"items"`
}

type ChecklistHeaderPatch struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

func (p *ChecklistHeaderPatch) Validate() error {
	p.Value = strings.TrimSpace(p.Value)
	if p.Value == "" || len([]rune(p.Value)) > 255 {
		return ErrInvalidChecklist
	}
	switch p.Field {
	case "title", "number", "documentName", "digitalSignature", "internalGroup", "externalGroup", "documentPosition", "sentDate", "notes":
		return nil
	default:
		return ErrInvalidChecklist
	}
}

type ChecklistItem struct {
	ID               int          `json:"id"`
	ParentID         *int         `json:"parentId"`
	Number           string       `json:"number"`
	DocumentName     string       `json:"documentName"`
	DigitalSigned    bool         `json:"digitalSigned"`
	DocumentPosition string       `json:"documentPosition"`
	SentDate         *string      `json:"sentDate"`
	Notes            string       `json:"notes"`
	CreatedAt        time.Time    `json:"createdAt"`
	UpdatedAt        time.Time    `json:"updatedAt"`
	Signatures       map[int]bool `json:"signatures"`
}

func NumberChecklistItems(items []ChecklistItem) {
	counts, numbers := map[int]int{}, map[int]string{}
	for i := range items {
		parent := 0
		if items[i].ParentID != nil {
			parent = *items[i].ParentID
		}
		counts[parent]++
		items[i].Number = fmt.Sprint(counts[parent])
		if parent != 0 {
			items[i].Number = numbers[parent] + "." + items[i].Number
		}
		numbers[items[i].ID] = items[i].Number
	}
}

// One field per PATCH prevents stale row snapshots from overwriting other cells.
type ChecklistItemPatch struct {
	Field    string `json:"field"`
	Value    string `json:"value"`
	Checked  bool   `json:"checked"`
	SignerID int    `json:"signerId"`
}

func (p *ChecklistItemPatch) Validate() error {
	switch p.Field {
	case "documentName":
		p.Value = strings.TrimSpace(p.Value)
		if p.Value == "" || len([]rune(p.Value)) > 255 {
			return ErrInvalidChecklist
		}
	case "documentPosition":
		if len([]rune(p.Value)) > 255 {
			return ErrInvalidChecklist
		}
	case "notes":
		if len([]rune(p.Value)) > 4000 {
			return ErrInvalidChecklist
		}
	case "sentDate":
		if p.Value != "" {
			if _, err := time.Parse("2006-01-02", p.Value); err != nil {
				return ErrInvalidChecklist
			}
		}
	case "digitalSigned":
	case "signature":
		if p.SignerID <= 0 {
			return ErrInvalidChecklist
		}
	default:
		return ErrInvalidChecklist
	}
	return nil
}
