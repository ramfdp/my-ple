package controller

import (
	"archive/zip"
	"bytes"
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
		if y < 42 || y > 800 {
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
