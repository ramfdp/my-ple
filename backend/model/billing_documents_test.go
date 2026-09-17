package model

import "testing"

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
