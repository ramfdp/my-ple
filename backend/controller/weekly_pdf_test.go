package controller

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"my-ple-backend/model"
)

func TestWeeklyPDFGroupsProjectsAndCreatesMultiplePages(t *testing.T) {
	start, next := model.ReportingWeek(time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC))
	period := model.ReportingPeriod{Start: start, End: next.AddDate(0, 0, -1), Label: model.IndonesianWeekLabel(start, next)}
	reports := make([]model.WeeklyReport, 70)
	for i := range reports {
		reports[i] = model.WeeklyReport{PrimaryDivision: "Project Engineer", WorkDivision: "MK supervisi", Role: "PE", Work: "Project A", Description: strings.Repeat("Laporan pekerjaan panjang ", 8)}
	}
	data, err := weeklyPDFDocument("Project Engineer", period, reports, false)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("output is not a PDF")
	}
	pages := bytes.Count(data, []byte("/Type /Page\n"))
	if pages < 2 {
		t.Fatal("long report did not create multiple pages")
	}
	if got := bytes.Count(data, []byte("CONFIDENTIAL")); got != pages {
		t.Fatalf("watermarks = %d, pages = %d", got, pages)
	}
}

func TestWeeklyPDFUsesDivisionTitleAndMergedRoleAndCategoryCells(t *testing.T) {
	start, next := model.ReportingWeek(time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC))
	period := model.ReportingPeriod{Start: start, End: next.AddDate(0, 0, -1), Label: model.IndonesianWeekLabel(start, next)}
	reports := []model.WeeklyReport{
		{PrimaryDivision: "Project Engineer", WorkDivision: "MK supervisi", Role: "PE", Work: "Penataan Kawasan Stasiun Jember", Description: "anjayy"},
		{PrimaryDivision: "Project Engineer", WorkDivision: "MK supervisi", Role: "PE", Work: "Penataan Kawasan Stasiun Jember", Description: "gokil"},
		{PrimaryDivision: "Project Engineer", WorkDivision: "MK supervisi", Role: "PE", Work: "Penataan Kawasan Stasiun Jember", Description: "awas awas remmm cittt diyarr"},
		{PrimaryDivision: "Project Engineer", WorkDivision: "MK supervisi", Role: "PE", Work: "Pengawasan Renovasi Gudang Besar JAKG", Description: "brem bremmm"},
		{PrimaryDivision: "Project Engineer", WorkDivision: "Non-MK supervisi", Role: "PE", Work: "Studi Daya Beli Gas Bumi", Description: "gas nguenggg anjay"},
		{PrimaryDivision: "Project Engineer", WorkDivision: "QA 1", Role: "QA 1", Work: "Internal", Description: "rama tidur"},
		{PrimaryDivision: "Project Engineer", WorkDivision: "QA 2", Role: "QA 2", Work: "Project Management Office", Description: "rama tidur"},
		{PrimaryDivision: "Project Engineer", WorkDivision: "JAKG", Role: "CIHPK", Work: "JAKG Supervising Consultant", Description: "test JAKG CIHPK"},
		{PrimaryDivision: "Project Engineer", WorkDivision: "JAKG", Role: "CIHPK", Work: "JAKG Supervising Consultant", Description: "testing chipk"},
		{PrimaryDivision: "Project Engineer", WorkDivision: "Jember", Role: "CIHPK", Work: "Jember Station Area Redevelopment", Description: "cikini ke gondang dia"},
	}

	data, err := weeklyPDFDocument("Project Engineer", period, reports, false)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("Weekly Report Project Engineer")) {
		t.Fatal("selected division is missing from title")
	}
	for _, category := range []string{"MK supervisi", "Non-MK supervisi", "JAKG", "Jember"} {
		if got := bytes.Count(data, []byte("("+category+")Tj")); got != 1 {
			t.Fatalf("category %q rendered %d times", category, got)
		}
	}
	if bytes.Contains(data, []byte("QA 1")) || bytes.Contains(data, []byte("QA 2")) || bytes.Count(data, []byte("(QA)Tj")) != 2 {
		t.Fatal("legacy QA role/category values were not merged into canonical QA")
	}
	for _, role := range []string{"PE", "CIHPK"} {
		if got := bytes.Count(data, []byte("("+role+")Tj")); got != 1 {
			t.Fatalf("role %q rendered %d times", role, got)
		}
	}
	numberCommands := regexp.MustCompile(`(?m)^([0-9.]+) [^\r\n]*\([0-9]+\)Tj`).FindAllSubmatch(data, -1)
	expectedX := fmt.Sprintf("%.2f", (15.0+20.0+42.0)*72/25.4)
	if len(numberCommands) != len(reports) {
		t.Fatalf("number cells = %d, reports = %d", len(numberCommands), len(reports))
	}
	for _, command := range numberCommands {
		if string(command[1]) != expectedX {
			t.Fatalf("No column starts at %s, want %s", command[1], expectedX)
		}
	}
	if got := bytes.Count(data, []byte("(1)Tj")); got != 7 {
		t.Fatalf("project starts numbered 1 = %d, want 7", got)
	}
}
