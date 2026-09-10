package controller

import (
	"bytes"
	"my-ple-backend/model"
	"testing"
)

func TestPDFWithoutWatermark(t *testing.T) {
	data, err := weeklyPDFDocument("AI Dev", model.ReportingPeriod{Label: "Test"}, []model.WeeklyReport{{ID: "x", Work: "Task", Description: "Test report", Number: 9}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(bytes.ToUpper(data), []byte("CONFIDENTIAL")) {
		t.Fatal("watermark retained")
	}
	if !bytes.Contains(data, []byte("(1)")) || bytes.Contains(data, []byte("(9)")) {
		t.Fatal("PDF display number not compact")
	}
}
