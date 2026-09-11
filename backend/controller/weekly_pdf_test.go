package controller

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"my-ple-backend/model"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestPDFAndDOCXReportOrder(t *testing.T) {
	reports := []model.WeeklyReport{
		{ID: "c", PrimaryDivision: "Project Engineer", Role: "QA", WorkDivision: "MK supervisi", Work: "Road", Number: 9, Description: "QAMARKER"},
		{ID: "b", PrimaryDivision: "Project Engineer", Role: "PE", WorkDivision: "MK supervisi", Work: "Road", Number: 7, Description: "PESECONDMARKER"},
		{ID: "a", PrimaryDivision: "Project Engineer", Role: "PE", WorkDivision: "MK supervisi", Work: "Road", Number: 2, Description: "PEFIRSTMARKER"},
	}
	period := model.ReportingPeriod{Label: "Same week"}
	pdf, err := weeklyPDFDocument("Project Engineer", period, reports, false)
	if err != nil {
		t.Fatal(err)
	}
	word, err := weeklyDOCX("Project Engineer", period, reports)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(word), int64(len(word)))
	if err != nil {
		t.Fatal(err)
	}
	var document []byte
	for _, file := range archive.File {
		if file.Name == "word/document.xml" {
			reader, err := file.Open()
			if err != nil {
				t.Fatal(err)
			}
			document, err = io.ReadAll(reader)
			reader.Close()
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	pdfPosition, wordPosition := -1, -1
	for _, marker := range []string{"PEFIRSTMARKER", "PESECONDMARKER", "QAMARKER"} {
		encoded := []byte{}
		for _, c := range []byte(marker) {
			encoded = append(encoded, 0, c)
		}
		p, w := bytes.Index(pdf, encoded), bytes.Index(document, []byte(marker))
		if p <= pdfPosition || w <= wordPosition || bytes.Count(pdf, encoded) != 1 || bytes.Count(document, []byte(marker)) != 1 {
			t.Fatal("export ordering/content differs", marker)
		}
		pdfPosition, wordPosition = p, w
	}
	if reports[0].Number != 9 || reports[1].Number != 7 || reports[2].Number != 2 {
		t.Fatal("export mutated source numbering")
	}
}

func TestPDFWithoutWatermark(t *testing.T) {
	data, err := weeklyPDFDocument("AI Dev", model.ReportingPeriod{Label: "Test"}, []model.WeeklyReport{{ID: "x", Work: "Task", Description: "Test report", Number: 9}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(bytes.ToUpper(data), []byte("CONFIDENTIAL")) {
		t.Fatal("watermark retained")
	}
	if !bytes.Contains(data, []byte("(\x001)")) || bytes.Contains(data, []byte("(\x009)")) {
		t.Fatal("PDF display number not compact")
	}
}

func TestPDFIncludesWebsiteTopHeader(t *testing.T) {
	pdf, err := weeklyPDFDocument("Project Engineer", model.ReportingPeriod{Label: "Test"}, []model.WeeklyReport{{ID: "x", Role: "PE", WorkDivision: "MK supervisi", Work: "Task", Description: "Progress"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(pdf, utf16BE("PE & QA")) {
		t.Fatal("PDF is missing the website top header")
	}
}

func TestDocumentsIncludePageNumbersAndMatchingHeader(t *testing.T) {
	reports := []model.WeeklyReport{{ID: "x", Role: "PE", WorkDivision: "MK supervisi", Work: "Task", Description: "Progress"}}
	period := model.ReportingPeriod{Label: "Test period"}
	pdf, err := weeklyPDFDocument("Project Engineer", period, reports, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"Page 1 / 1", "Weekly Report Project Engineer", "Test period"} {
		if !bytes.Contains(pdf, utf16BE(value)) {
			t.Fatalf("PDF is missing %q", value)
		}
	}
	docx, err := weeklyDOCX("Project Engineer", period, reports)
	if err != nil {
		t.Fatal(err)
	}
	for part, values := range map[string][]string{
		"word/header.xml": {"Weekly Report", "PE &amp; QA", "Weekly Report Project Engineer", "Test period"},
		"word/footer.xml": {`w:instrText xml:space="preserve"> PAGE `, `w:instrText xml:space="preserve"> NUMPAGES `},
	} {
		data := docxPart(t, docx, part)
		for _, value := range values {
			if !bytes.Contains(data, []byte(value)) {
				t.Fatalf("%s is missing %q", part, value)
			}
		}
	}
	document := docxDocument(t, docx)
	for _, value := range []string{`w:headerReference w:type="default" r:id="rId1"`, `w:footerReference w:type="default" r:id="rId2"`, `w:fill="123A5A"`} {
		if !bytes.Contains(document, []byte(value)) {
			t.Fatalf("DOCX template is missing %q", value)
		}
	}
}

func TestDocumentsGroupRepeatedRoleCategoryAndJobLabels(t *testing.T) {
	reports := []model.WeeklyReport{
		{ID: "a", PrimaryDivision: "Project Engineer", Role: "PE", WorkDivision: "MK supervisi", Work: "Same job", Number: 1, Description: "Progress one"},
		{ID: "b", PrimaryDivision: "Project Engineer", Role: "PE", WorkDivision: "MK supervisi", Work: "Same job", Number: 2, Description: "Progress two"},
		{ID: "c", PrimaryDivision: "Project Engineer", Role: "PE", WorkDivision: "MK supervisi", Work: "Other job", Number: 1, Description: "Progress three"},
	}
	period := model.ReportingPeriod{Label: "Test"}
	pdf, err := weeklyPDFDocument("Project Engineer", period, reports, false)
	if err != nil {
		t.Fatal(err)
	}
	docx, err := weeklyDOCX("Project Engineer", period, reports)
	if err != nil {
		t.Fatal(err)
	}
	document := docxDocument(t, docx)
	for label, counts := range map[string][2]int{"Project Engineer": {1, 1}, "MK/Supervisi": {1, 1}} {
		pdfCount, docxCount := bytes.Count(pdf, utf16BE(label)), bytes.Count(document, []byte(label))
		if pdfCount != counts[0] || docxCount != counts[1] {
			t.Fatalf("%q was not grouped in the table: PDF=%d DOCX=%d", label, pdfCount, docxCount)
		}
	}
	for _, job := range []string{"Same job", "Other job"} {
		if bytes.Count(pdf, utf16BE(job)) != 1 || bytes.Count(document, []byte(job)) != 1 {
			t.Fatalf("job %q was not grouped once", job)
		}
	}
}

func TestExportGroupsRequiredHierarchy(t *testing.T) {
	jobA, jobB, pit, sgr := 1, 2, 3, 4
	picsA := model.JobPICs{{UserID: "u1", Name: "mingse", IsActive: true}, {UserID: "u2", Name: "nasfa", IsActive: true}}
	picsB := model.JobPICs{{UserID: "u1", Name: "mingse", IsActive: true}}
	reports := []model.WeeklyReport{}
	for i, content := range []string{"PROGRESS-A", "PROGRESS-B", "PROGRESS-C", "PROGRESS-D", "PROGRESS-E"} {
		reports = append(reports, model.WeeklyReport{ID: fmt.Sprint("a", i), JobID: &jobA, PrimaryDivision: "Project Engineer", Role: "PE", WorkDivision: "MK supervisi", Work: "Penataan Kawasan Stasiun Jember", Number: i + 10, PICs: picsA, Description: content})
	}
	reports = append(reports,
		model.WeeklyReport{ID: "b", JobID: &jobB, PrimaryDivision: "Project Engineer", Role: "PE", WorkDivision: "MK supervisi", Work: "Pengawasan Renovasi Gudang Besar JAKG", Number: 40, PICs: picsB, Description: "JOB-B-PROGRESS"},
		model.WeeklyReport{ID: "pit", JobID: &pit, PrimaryDivision: "Project Engineer", Role: "PE", WorkDivision: "Non-MK supervisi", Work: "MPS PIT", Number: 50, Description: "PIT-PROGRESS"},
		model.WeeklyReport{ID: "sgr", JobID: &sgr, PrimaryDivision: "Project Engineer", Role: "PE", WorkDivision: "Non-MK supervisi", Work: "MPS SGR", Number: 60, Description: "SGR-PROGRESS"},
	)
	groups := exportGroups(reports)
	if len(groups) != 1 || groups[0].Name != "Project Engineer" || len(groups[0].Categories) != 2 {
		t.Fatalf("wrong role/category hierarchy: %#v", groups)
	}
	if got := groups[0].Categories[0]; got.Name != "MK/Supervisi" || len(got.Jobs) != 2 || len(got.Jobs[0].Progress) != 5 || got.Jobs[1].Progress[0].Number != 1 {
		t.Fatalf("wrong MK grouping or numbering: %#v", got)
	}
	if got := groups[0].Categories[1]; got.Name != "Non MK/Non Supervisi" || len(got.Jobs) != 2 || got.Jobs[0].Progress[0].Number != 1 || got.Jobs[1].Progress[0].Number != 1 {
		t.Fatalf("wrong Non-MK grouping or numbering: %#v", got)
	}
	pdf, err := weeklyPDFDocument("Project Engineer", model.ReportingPeriod{Label: "Test"}, reports, false)
	if err != nil {
		t.Fatal(err)
	}
	if dir := os.Getenv("DOCUMENT_TEST_OUTPUT"); dir != "" {
		if err := os.WriteFile(filepath.Join(dir, "weekly-report-layout.pdf"), pdf, 0600); err != nil {
			t.Fatal(err)
		}
	}
	positions := map[float64]string{}
	for _, marker := range []string{"PROGRESS-A", "PROGRESS-D", "PROGRESS-E", "JOB-B-PROGRESS", "PIT-PROGRESS", "SGR-PROGRESS"} {
		y := pdfTextY(t, pdf, marker)
		if previous := positions[y]; previous != "" {
			t.Fatalf("PDF rows overlap: %q and %q share y=%v", previous, marker, y)
		}
		positions[y] = marker
	}
	docx, err := weeklyDOCX("Project Engineer", model.ReportingPeriod{Label: "Test"}, reports)
	if err != nil {
		t.Fatal(err)
	}
	document := docxDocument(t, docx)
	for _, value := range []string{"MK/Supervisi", "Penataan Kawasan Stasiun Jember", "Pengawasan Renovasi Gudang Besar JAKG", "MPS PIT", "MPS SGR", "nasfa", "PIT-PROGRESS", "SGR-PROGRESS"} {
		if !bytes.Contains(pdf, utf16BE(value)) || !bytes.Contains(document, []byte(value)) {
			t.Fatalf("PDF/DOCX mismatch: missing %q", value)
		}
	}
	if !bytes.Contains(document, []byte("Non MK/Non Supervisi")) || !bytes.Contains(pdf, utf16BE("Non MK/Non")) || !bytes.Contains(pdf, utf16BE("Supervisi")) {
		t.Fatal("wrapped Non MK/Non Supervisi category missing")
	}
	if bytes.Count(document, []byte(`<w:gridSpan w:val="2"/>`)) != 4 || !bytes.Contains(document, []byte(`<w:vMerge w:val="restart"/>`)) || !bytes.Contains(document, []byte(`<w:vMerge w:val="continue"/>`)) {
		t.Fatal("DOCX is missing job or vertical cell merges")
	}
}

func TestCIHPKStartsOnNewPageOnlyAfterOtherRoles(t *testing.T) {
	report := func(id, role, category, marker string) model.WeeklyReport {
		job := len(id)
		return model.WeeklyReport{ID: id, JobID: &job, PrimaryDivision: "Project Engineer", Role: role, WorkDivision: category, Work: id + " job", Description: marker}
	}
	period := model.ReportingPeriod{Label: "Test"}
	all := []model.WeeklyReport{report("pe", "PE", "MK supervisi", "PEMARKER"), report("qa", "QA", "QA1", "QAMARKER"), report("cihpk", "CIHPK", "Jember", "CIHPKMARKER")}
	pdf, err := weeklyPDFDocument("Project Engineer", period, all, false)
	if err != nil {
		t.Fatal(err)
	}
	if dir := os.Getenv("DOCUMENT_TEST_OUTPUT"); dir != "" {
		if err := os.WriteFile(filepath.Join(dir, "weekly-report-cihpk.pdf"), pdf, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if pe, qa, cihpk := pdfPageOf(pdf, "PEMARKER"), pdfPageOf(pdf, "QAMARKER"), pdfPageOf(pdf, "CIHPKMARKER"); pe != qa || cihpk != qa+1 {
		t.Fatalf("wrong PDF role pages: PE=%d QA=%d CIHPK=%d", pe, qa, cihpk)
	}
	if page, y := pdfPageOf(pdf, "CIHPK"), pdfTextY(t, pdf, "CIHPK"); page != 2 || y < 800 {
		t.Fatalf("CIHPK page header missing: page=%d y=%v", page, y)
	}
	docx, err := weeklyDOCX("Project Engineer", period, all)
	if err != nil {
		t.Fatal(err)
	}
	document := docxDocument(t, docx)
	pageBreak := []byte(`<w:br w:type="page"/>`)
	if bytes.Count(document, pageBreak) != 1 || bytes.Index(document, []byte("QAMARKER")) > bytes.Index(document, pageBreak) || bytes.Index(document, pageBreak) > bytes.Index(document, []byte("CIHPKMARKER")) {
		t.Fatal("DOCX page break is not exactly between QA and CIHPK")
	}
	for name, reports := range map[string][]model.WeeklyReport{
		"only CIHPK": {report("cihpk", "CIHPK", "Jember", "ONLYCIHPK")},
		"no CIHPK":   {report("pe", "PE", "MK supervisi", "ONLYPE")},
	} {
		pdf, err := weeklyPDFDocument("Project Engineer", period, reports, false)
		if err != nil {
			t.Fatal(err)
		}
		docx, err := weeklyDOCX("Project Engineer", period, reports)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Count(pdf, []byte("/Type /Page\n")) != 1 || bytes.Contains(docxDocument(t, docx), pageBreak) {
			t.Fatalf("%s created an unnecessary page", name)
		}
	}
}

func TestCIHPKStartsAfterLongQACompletes(t *testing.T) {
	peJob, qaJob, cihpkJob := 1, 2, 3
	reports := []model.WeeklyReport{
		{ID: "pe", JobID: &peJob, PrimaryDivision: "Project Engineer", Role: "PE", WorkDivision: "MK supervisi", Work: "PE job", Description: strings.Repeat("PE progress ", 800)},
		{ID: "qa", JobID: &qaJob, PrimaryDivision: "Project Engineer", Role: "QA", WorkDivision: "QA1", Work: "QA job", Description: strings.Repeat("QA progress ", 2200) + "ENDQALONG"},
		{ID: "cihpk", JobID: &cihpkJob, PrimaryDivision: "Project Engineer", Role: "CIHPK", WorkDivision: "Jember", Work: "CIHPK job", Description: "CIHPKAFTERLONGQA"},
	}
	pdf, err := weeklyPDFDocument("Project Engineer", model.ReportingPeriod{Label: "Test"}, reports, false)
	if err != nil {
		t.Fatal(err)
	}
	qaPage, cihpkPage := pdfPageOf(pdf, "ENDQALONG"), pdfPageOf(pdf, "CIHPKAFTERLONGQA")
	if qaPage < 2 || cihpkPage != qaPage+1 {
		t.Fatalf("CIHPK page %d did not follow long QA page %d", cihpkPage, qaPage)
	}
}

func TestPDFKeepsJobHeadingWithFirstProgress(t *testing.T) {
	firstJob, nextJob := 1, 2
	reports := make([]model.WeeklyReport, 0, 25)
	for i := range 23 {
		reports = append(reports, model.WeeklyReport{ID: fmt.Sprint("first-", i), JobID: &firstJob, PrimaryDivision: "AI Dev", Role: "AI Dev", WorkDivision: "Development", Work: "First job", Description: fmt.Sprint("Progress ", i)})
	}
	reports = append(reports, model.WeeklyReport{ID: "next", JobID: &nextJob, PrimaryDivision: "AI Dev", Role: "AI Dev", WorkDivision: "Development", Work: strings.Repeat("WRAPPED JOB HEADING ", 8), Description: "FIRSTPROGRESSMARKER"})
	pdf, err := weeklyPDFDocument("AI Dev", model.ReportingPeriod{Label: "Test"}, reports, false)
	if err != nil {
		t.Fatal(err)
	}
	if headingPage, progressPage := pdfPageOf(pdf, "WRAPPED JOB HEADING"), pdfPageOf(pdf, "FIRSTPROGRESSMARKER"); headingPage == 0 || headingPage != progressPage {
		t.Fatalf("orphan job heading: heading page=%d first progress page=%d", headingPage, progressPage)
	}
}

func pdfPageOf(data []byte, marker string) int {
	position := bytes.Index(data, utf16BE(marker))
	if position < 0 {
		return 0
	}
	return bytes.Count(data[:position], []byte("/Type /Page\n"))
}

func pdfTextY(t *testing.T, data []byte, marker string) float64 {
	t.Helper()
	position := bytes.Index(data, utf16BE(marker))
	if position < 0 {
		t.Fatalf("PDF marker missing: %q", marker)
	}
	matches := regexp.MustCompile(`([0-9.]+) ([0-9.]+) Td`).FindAllSubmatch(data[:position], -1)
	if len(matches) == 0 {
		t.Fatalf("PDF position missing: %q", marker)
	}
	y, err := strconv.ParseFloat(string(matches[len(matches)-1][2]), 64)
	if err != nil {
		t.Fatal(err)
	}
	return y
}

func docxDocument(t *testing.T, data []byte) []byte {
	return docxPart(t, data, "word/document.xml")
}

func docxPart(t *testing.T, data []byte, name string) []byte {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range archive.File {
		if file.Name == name {
			reader, err := file.Open()
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			document, err := io.ReadAll(reader)
			if err != nil {
				t.Fatal(err)
			}
			return document
		}
	}
	t.Fatalf("%s missing", name)
	return nil
}

func utf16BE(value string) []byte {
	result := make([]byte, 0, len(value)*2)
	for _, r := range value {
		result = append(result, byte(r>>8), byte(r))
	}
	return result
}

func TestLongDocuments(t *testing.T) {
	content := strings.Repeat("Paragraf laporan panjang • café & <aman>\n\n", 350) + strings.Repeat("LONGTOKEN", 200) + "\nENDOFREPORT"
	reports := []model.WeeklyReport{{ID: "long", PrimaryDivision: "Project Engineer", Role: "PE", WorkDivision: "MK supervisi", Work: "Long work", Number: 9, Description: content}}
	period := model.ReportingPeriod{Label: "7–13 September 2026"}
	pdf, err := weeklyPDFDocument("Project Engineer", period, reports, false)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Count(pdf, []byte("/Type /Page\n")) < 10 {
		t.Fatal("long entry did not paginate")
	}
	if !bytes.Contains(pdf, []byte("\x00E\x00N\x00D\x00O\x00F\x00R\x00E\x00P\x00O\x00R\x00T")) {
		t.Fatal("PDF lost final content")
	}
	// FPDF writes one absolute Td position per text fragment (PDF points).
	positions := regexp.MustCompile(`([0-9.]+) ([0-9.]+) Td`).FindAllSubmatch(pdf, -1)
	if len(positions) == 0 {
		t.Fatal("missing PDF text positions")
	}
	for _, position := range positions {
		y, _ := strconv.ParseFloat(string(position[2]), 64)
		if y < 18 || y > 825 {
			t.Fatalf("PDF text outside printable page: %v", y)
		}
	}
	docx, err := weeklyDOCX("Project Engineer", period, reports)
	if err != nil {
		t.Fatal(err)
	}
	// Optional local artifacts for a real PDF parser and Microsoft Word verification.
	if dir := os.Getenv("DOCUMENT_TEST_OUTPUT"); dir != "" {
		for name, data := range map[string][]byte{"long-report.pdf": pdf, "long-report.docx": docx, "long-report.txt": []byte(content)} {
			if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
}
