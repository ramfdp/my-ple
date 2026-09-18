package controller

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"my-ple-backend/model"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestChecklistPDFIncludesAllTablesAndLongNotes(t *testing.T) {
	tables := []*model.DocumentChecklist{
		{Title: "First checklist", RequireDigitalSignature: true, Signers: []model.ChecklistSigner{{ID: 1, Name: "Signer One", Position: "Manager", Type: "internal"}}, Items: []model.ChecklistItem{{DocumentName: "FIRSTDOCUMENT", DigitalSigned: true, Signatures: map[int]bool{1: true}, Notes: strings.Repeat("Long note\n", 140) + "ENDOFNOTES"}}},
		{Title: "Second checklist", Items: []model.ChecklistItem{{DocumentName: "SECONDDOCUMENT"}}},
		{Title: "Empty checklist"},
	}
	data, err := checklistPDF("JOBMARKER", tables)
	if err != nil || !validPDF(data) {
		t.Fatalf("invalid PDF: %v", err)
	}
	content := checklistPDFText(t, data)
	for _, marker := range []string{"JOBMARKER", "FIRSTDOCUMENT", "SECONDDOCUMENT", "ENDOFNOTES", "TTD Hardcopy", "Signer One", "Belum ada dokumen."} {
		if !bytes.Contains(content, utf16BE(marker)) {
			t.Fatalf("missing PDF content %q", marker)
		}
	}
	if bytes.Count(data, []byte("/Type /Page\n")) < 5 {
		t.Fatal("long notes did not paginate")
	}
}

func checklistPDFText(t *testing.T, data []byte) []byte {
	t.Helper()
	var content []byte
	for _, match := range regexp.MustCompile(`(?s)stream\r?\n(.*?)\r?\nendstream`).FindAllSubmatch(data, -1) {
		reader, err := zlib.NewReader(bytes.NewReader(match[1]))
		if err != nil {
			continue
		}
		decoded, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		content = append(content, decoded...)
	}
	return content
}

func TestChecklistPDFTableLayout(t *testing.T) {
	tables := []*model.DocumentChecklist{
		{Title: "Checklist Pertama", RequireDigitalSignature: true, Signers: []model.ChecklistSigner{{ID: 1, Name: "Nama Internal", Position: "Project Manager", Type: "internal"}, {ID: 2, Name: "Nama External", Position: "Direktur", Type: "external"}, {ID: 3, Name: "ARCHIVED", Position: "Old", Type: "internal", Archived: true}}, Items: []model.ChecklistItem{{DocumentName: "Berita Acara", DigitalSigned: true, Signatures: map[int]bool{1: true}, Notes: "Dokumen siap diperiksa."}}},
		{Title: "Checklist Kedua", Items: []model.ChecklistItem{{DocumentName: "Lampiran", Notes: "Catatan berikutnya"}}},
	}
	data, err := checklistPDF("Pekerjaan Pengujian", tables)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Count(data, []byte("/Type /Page\n")) != 1 {
		t.Fatal("short tables should share one page")
	}
	text := checklistPDFText(t, data)
	for _, label := range []string{"TTD Hardcopy", "Internal", "External", "Nama Internal", "Nama External", "Project Manager", "Checklist Kedua", "Digital"} {
		if !bytes.Contains(text, utf16BE(label)) {
			t.Fatalf("missing %s", label)
		}
	}
	if bytes.Contains(text, utf16BE("ARCHIVED")) {
		t.Fatal("archived signer exported")
	}
	if dir := os.Getenv("DOCUMENT_TEST_OUTPUT"); dir != "" {
		if err = os.WriteFile(filepath.Join(dir, "checklist-table-layout.pdf"), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	prior := &model.DocumentChecklist{Title: "Previous"}
	for i := 0; i < 12; i++ {
		prior.Items = append(prior.Items, model.ChecklistItem{DocumentName: "Short"})
	}
	next := &model.DocumentChecklist{Title: "NOORPHAN", Items: []model.ChecklistItem{{DocumentName: "First row", Notes: strings.Repeat("First row note\n", 20)}}}
	packed, err := checklistPDF("Job", []*model.DocumentChecklist{prior, next})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Count(checklistPDFText(t, packed), utf16BE("NOORPHAN")) != 1 {
		t.Fatal("table heading stranded without its first row")
	}
	tables[0].RequireDigitalSignature = false
	tables[0].Signers = nil
	for i := 1; i <= 16; i++ {
		tables[0].Signers = append(tables[0].Signers, model.ChecklistSigner{ID: i, Name: fmt.Sprintf("SIGNER%02d", i), Position: "Manager", Type: "internal"})
	}
	data, err = checklistPDF("Wide checklist", tables[:1])
	if err != nil {
		t.Fatal(err)
	}
	text = checklistPDFText(t, data)
	for i := 1; i <= 16; i++ {
		if !bytes.Contains(text, utf16BE(fmt.Sprintf("SIGNER%02d", i))) {
			t.Fatal("signer lost", i)
		}
	}
	if bytes.Contains(text, utf16BE("Digital")) {
		t.Fatal("disabled digital column exported")
	}
	if dir := os.Getenv("DOCUMENT_TEST_OUTPUT"); dir != "" {
		if err = os.WriteFile(filepath.Join(dir, "checklist-wide-layout.pdf"), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
