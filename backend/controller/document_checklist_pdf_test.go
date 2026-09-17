package controller

import (
	"bytes"
	"compress/zlib"
	"io"
	"my-ple-backend/model"
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
	for _, marker := range []string{"JOBMARKER", "FIRSTDOCUMENT", "SECONDDOCUMENT", "ENDOFNOTES", "TTD Hardcopy", "Signer One", "Belum ada dokumen."} {
		if !bytes.Contains(content, utf16BE(marker)) {
			t.Fatalf("missing PDF content %q", marker)
		}
	}
	if bytes.Count(data, []byte("/Type /Page\n")) < 5 {
		t.Fatal("long notes did not paginate")
	}
}
