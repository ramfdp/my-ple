package model

import "testing"

func TestBillingSharedDocuments(t *testing.T) {
	s := BillingStage{Documents: []BillingDocument{{Status: "SIRKULIR TTD"}}, Milestones: []BillingMilestone{{}, {Documents: []BillingDocument{{Status: "OK"}}}}}
	s.Calculate()
	if s.Readiness != "BELUM SIAP" || s.Milestones[0].Readiness != "BELUM SIAP" {
		t.Fatal(s)
	}
	s.Documents[0].Status = "OK"
	s.Calculate()
	if s.Readiness != "SIAP" || s.Milestones[0].Readiness != "SIAP" {
		t.Fatal(s)
	}
	c := BillingConfiguration{JobID: 1, Name: "TW-1", PeriodLabel: "2026", Documents: []BillingRequirement{{Name: "BAST"}}, Milestones: []BillingRequirement{{Name: "Bulan 1"}}}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	c.Milestones[0].Documents = []BillingRequirement{{Name: " bast "}}
	if c.Validate() == nil {
		t.Fatal("shared/local duplicate accepted")
	}
	c.Milestones[0].Documents = nil
	c.Documents = append(c.Documents, BillingRequirement{Name: "bast"})
	if c.Validate() == nil {
		t.Fatal("duplicate shared document accepted")
	}
}

func TestBillingReadiness(t *testing.T) {
	s := BillingStage{Milestones: []BillingMilestone{{Documents: []BillingDocument{{Status: "OK"}, {Status: "OK"}}}, {Documents: []BillingDocument{{Status: "OK"}}}}}
	s.Calculate()
	if s.Readiness != "SIAP" {
		t.Fatal(s)
	}
	for _, status := range []string{"BELUM ADA", "PEMBUATAN", "KURANG LENGKAP", "SIRKULIR TTD", ""} {
		s.Milestones[0].Documents[0].Status = status
		s.Calculate()
		if s.Readiness != "BELUM SIAP" || s.Milestones[0].Readiness != "BELUM SIAP" || s.Milestones[1].Readiness != "SIAP" {
			t.Fatal(status, s)
		}
	}
	s.Milestones[0].Documents[0].Status = "OK"
	s.Calculate()
	if s.Readiness != "SIAP" {
		t.Fatal(s)
	}
	s.Milestones[0].Documents = nil
	s.Calculate()
	if s.Readiness != "BELUM SIAP" {
		t.Fatal("empty milestone ready")
	}
	s.Milestones = nil
	s.Calculate()
	if s.Readiness != "BELUM SIAP" {
		t.Fatal("empty stage ready")
	}
}
