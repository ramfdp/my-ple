package model

import "testing"

func TestNormalizeWeeklyLabels(t *testing.T) {
	tests := []struct {
		role, category, work, wantRole, wantCategory, wantWork string
	}{
		{"PE", "MK supervisi", "Road", "Project Engineer", "MK/Supervisi", "Road"},
		{"QA", "QA1", "Audit", "Quality Assurance", "QA 1", "Audit"},
		{"QA 2", "QA2", "Audit", "Quality Assurance", "QA 2", "Audit"},
		{"CIHPK", "Jember", "Jember Station Area Redevelopment", "CIHPK", "MK/Supervisi", "Supervisi Penataan Kawasan Stasiun Jember (Tanggal SPMK: 30 Desember 2025)"},
	}
	for _, tt := range tests {
		r := WeeklyReport{Role: tt.role, WorkDivision: tt.category, Work: tt.work}
		NormalizeWeeklyReport(&r)
		if r.Role != tt.wantRole || r.WorkDivision != tt.wantCategory || r.Work != tt.wantWork {
			t.Fatalf("normalize (%q, %q, %q) = (%q, %q, %q)", tt.role, tt.category, tt.work, r.Role, r.WorkDivision, r.Work)
		}
	}
}

func TestNormalizeWeeklyStorageKeys(t *testing.T) {
	for _, tt := range []struct{ role, category, wantRole, wantCategory string }{
		{"Project Engineer", "MK/Supervisi", "PE", "MK supervisi"},
		{"Project Engineer", "Non MK/Non Supervisi", "PE", "Non-MK supervisi"},
		{"Quality Assurance", "QA 1", "QA", "QA1"},
		{"Quality Assurance", "QA 2", "QA", "QA2"},
	} {
		role, category := NormalizeWeeklyRoleCategory(tt.role, tt.category)
		if role != tt.wantRole || category != tt.wantCategory {
			t.Fatalf("storage keys (%q, %q) = (%q, %q)", tt.role, tt.category, role, category)
		}
	}
}
