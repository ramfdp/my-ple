package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

type rekapFinding struct {
	Number         string `json:"number"`
	Title          string `json:"title"`
	InternalFactor string `json:"internalFactor"`
	ExternalFactor string `json:"externalFactor"`
	Impact         string `json:"impact"`
	Action         string `json:"action"`
	Notes          string `json:"notes"`
}

type rekapPair struct {
	Row     int           `json:"row"`
	Failure *rekapFinding `json:"failure"`
	Success *rekapFinding `json:"success"`
}

type rekapSection struct {
	Group      string      `json:"group"`
	Project    string      `json:"project"`
	HeadingRow int         `json:"headingRow"`
	Rows       []rekapPair `json:"rows"`
}

type RekapSectionLayout struct {
	Group      string `json:"group"`
	Project    string `json:"project"`
	HeadingRow int    `json:"headingRow"`
	Rows       []int  `json:"rows"`
}

func rekapSections() ([]rekapSection, error) {
	var source struct {
		Sections []rekapSection `json:"sections"`
	}
	if err := json.Unmarshal([]byte(rekapSeedJSON), &source); err != nil {
		return nil, err
	}
	return source.Sections, nil
}

func RekapLayout() ([]RekapSectionLayout, error) {
	sections, err := rekapSections()
	if err != nil {
		return nil, err
	}
	layout := make([]RekapSectionLayout, 0, len(sections))
	for _, section := range sections {
		item := RekapSectionLayout{Group: section.Group, Project: section.Project, HeadingRow: section.HeadingRow, Rows: []int{}}
		for _, pair := range section.Rows {
			item.Rows = append(item.Rows, pair.Row)
		}
		layout = append(layout, item)
	}
	return layout, nil
}

func seedRekap(ctx context.Context, tx *sql.Tx) error {
	sections, err := rekapSections()
	if err != nil {
		return err
	}
	for _, section := range sections {
		for _, pair := range section.Rows {
			for _, finding := range []struct {
				kind  string
				value *rekapFinding
			}{{"failure", pair.Failure}, {"success", pair.Success}} {
				if finding.value == nil {
					continue
				}
				v := finding.value
				_, err = tx.ExecContext(ctx, `INSERT INTO lessons_learned(job_id,kind,title,internal_factor,external_factor,impact,action,notes,source_group,source_project,source_row,source_number)
				 SELECT (SELECT CASE WHEN count(*)=1 THEN min(id) END FROM job_context WHERE division_name='Project Engineer' AND name=$1),$2,$3,$4,$5,$6,$7,$8,$9,$1,$10,$11
				 ON CONFLICT(source_row,kind) DO NOTHING`, section.Project, finding.kind, v.Title, v.InternalFactor, v.ExternalFactor, v.Impact, v.Action, v.Notes, section.Group, pair.Row, v.Number)
				if err != nil {
					return fmt.Errorf("REKAP row %d %s: %w", pair.Row, finding.kind, err)
				}
			}
		}
	}
	return nil
}
