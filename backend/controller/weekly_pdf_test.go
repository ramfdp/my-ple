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

	// Fixture numbers are supplied by the backend, not synthesized by the PDF.
	for i, n := range []int{1, 2, 3, 1, 1, 1, 1, 1, 2, 1} {
		reports[i].Number = n
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
	for _, category := range []string{"QA1", "QA2"} {
		if got := bytes.Count(data, []byte("("+category+")Tj")); got != 1 {
			t.Fatalf("category %q rendered %d times", category, got)
		}
	}
	for role, want := range map[string]int{"PE": 1, "QA": 1, "CIHPK": 2} {
		if got := bytes.Count(data, []byte("("+role+")Tj")); got != want {
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

func TestWeeklyPDFSeparatesPEQAAndCIHPKSections(t *testing.T) {
	start, next := model.ReportingWeek(time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC))
	period := model.ReportingPeriod{Start: start, End: next.AddDate(0, 0, -1), Label: model.IndonesianWeekLabel(start, next)}
	reports := []model.WeeklyReport{
		{PrimaryDivision: "Project Engineer", WorkDivision: "MK supervisi", Role: "PE", Work: "Project PE", Description: "PE work"},
		{PrimaryDivision: "Project Engineer", WorkDivision: "QA1", Role: "QA", Work: "Internal", Description: "QA1 work"},
		{PrimaryDivision: "Project Engineer", WorkDivision: "QA2", Role: "QA", Work: "Project Management Office", Description: "QA2 work"},
		{PrimaryDivision: "Project Engineer", WorkDivision: "JAKG", Role: "CIHPK", Work: "Project CIHPK", Description: "CIHPK work"},
	}

	data, err := weeklyPDFDocument("Project Engineer", period, reports, false)
	if err != nil {
		t.Fatal(err)
	}
	if pages := bytes.Count(data, []byte("/Type /Page\n")); pages != 2 {
		t.Fatalf("pages = %d, want 2", pages)
	}
	for label, count := range map[string]int{"PE & QA": 1, "CIHPK": 2, "CONFIDENTIAL": 2} {
		if got := bytes.Count(data, []byte("("+label+")Tj")); got != count {
			t.Fatalf("%q rendered %d times, want %d", label, got, count)
		}
	}
	if got := bytes.Count(data, []byte(period.Label)); got != 2 {
		t.Fatalf("date range rendered %d times, want 2", got)
	}
}

func TestWeeklyPDFUsesStoredNumbersIncludingGaps(t *testing.T) {
	start, next := model.ReportingWeek(time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC))
	reports := []model.WeeklyReport{
		{PrimaryDivision: "Project Engineer", Role: "PE", WorkDivision: "MK supervisi", Work: "Job A", Number: 8, Description: "first"},
		{PrimaryDivision: "Project Engineer", Role: "PE", WorkDivision: "MK supervisi", Work: "Job A", Number: 10, Description: "after deletion"},
		{PrimaryDivision: "Project Engineer", Role: "PE", WorkDivision: "MK supervisi", Work: "Job B", Number: 1, Description: "independent"},
	}
	data, err := weeklyPDFDocument("Project Engineer", model.ReportingPeriod{Start: start, End: next.AddDate(0, 0, -1)}, reports, false)
	if err != nil {
		t.Fatal(err)
	}
	commands := regexp.MustCompile(`\(([0-9]+)\)Tj`).FindAllSubmatch(data, -1)
	if len(commands) != 3 {
		t.Fatalf("number cells: %d", len(commands))
	}
	for i, want := range []string{"8", "10", "1"} {
		if string(commands[i][1]) != want {
			t.Fatalf("number %d: %s want %s", i, commands[i][1], want)
		}
	}
}
