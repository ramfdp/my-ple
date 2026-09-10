package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-pdf/fpdf"
	"my-ple-backend/db"
	"my-ple-backend/middleware"
	"my-ple-backend/model"
)

var unsafeFilename = regexp.MustCompile(`[^A-Za-z0-9_-]+`)

func (c *WeeklyController) ExportPDF(w http.ResponseWriter, r *http.Request) {
	_, ok := middleware.GetClaims(r.Context())
	if !ok {
		WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Autentikasi diperlukan."})
		return
	}
	division := strings.TrimSpace(r.URL.Query().Get("primaryDivision"))
	if division == "" || len(division) > 100 {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Divisi utama tidak valid."})
		return
	}
	var valid bool
	if err := c.DB.Conn.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM divisions WHERE name=$1)`, division).Scan(&valid); err != nil {
		WriteJSON(w, 500, map[string]string{"error": "Divisi tidak dapat dimuat."})
		return
	}
	if !valid {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Divisi utama tidak valid."})
		return
	}
	period, err := c.currentPeriod(r.Context(), division)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Laporan tidak dapat diekspor."})
		return
	}
	reports, err := c.DB.PeriodReports(r.Context(), division, period)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Laporan tidak dapat diekspor."})
		return
	}
	if len(reports) == 0 {
		WriteJSON(w, http.StatusConflict, map[string]string{"error": "Belum ada laporan pekerjaan untuk periode ini."})
		return
	}
	data, err := weeklyPDF(division, period, reports)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Laporan tidak dapat diekspor."})
		return
	}
	filenameDivision := strings.Trim(unsafeFilename.ReplaceAllString(strings.ReplaceAll(division, " ", "_"), "_"), "_")
	filename := fmt.Sprintf("Weekly_Report_%s_%s_%s.pdf", filenameDivision, period.Start.Format("2006-01-02"), period.End.Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(data)
}

func weeklyPDF(division string, period model.ReportingPeriod, reports []model.WeeklyReport) ([]byte, error) {
	return weeklyPDFDocument(division, period, reports, true)
}

func weeklyPDFDocument(division string, period model.ReportingPeriod, reports []model.WeeklyReport, compression bool) ([]byte, error) {
	reports = append([]model.WeeklyReport(nil), reports...)
	for i := range reports {
		model.NormalizeWeeklyReport(&reports[i])
	}
	db.SortReports(reports)
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetCompression(compression)
	text := pdf.UnicodeTranslatorFromDescriptor("")
	const left, roleWidth, categoryWidth, numberWidth, reportWidth = 15.0, 20.0, 42.0, 14.0, 104.0
	pdf.SetMargins(left, 15, left)
	pdf.SetAutoPageBreak(false, 15)
	sectionLabel := "Weekly Report"
	pdf.SetHeaderFunc(func() {
		pdf.SetXY(left, 15)
		pdf.SetFont("Helvetica", "", 8)
		pdf.SetTextColor(100, 116, 139)
		pdf.CellFormat(90, 5, "Weekly Report", "", 0, "L", false, 0, "")
		pdf.CellFormat(90, 5, sectionLabel, "", 1, "R", false, 0, "")
		pdf.Ln(6)
		pdf.SetTextColor(16, 42, 67)
		pdf.SetFont("Helvetica", "B", 17)
		pdf.CellFormat(180, 9, text("Weekly Report "+division), "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
		pdf.CellFormat(180, 8, text("Tanggal: "+period.Label), "", 1, "L", false, 0, "")
		pdf.Ln(4)
		pdf.SetFillColor(18, 58, 90)
		pdf.SetTextColor(255, 255, 255)
		pdf.SetFont("Helvetica", "B", 9)
		pdf.CellFormat(roleWidth, 9, "Role", "1", 0, "C", true, 0, "")
		pdf.CellFormat(categoryWidth, 9, "Kategori", "1", 0, "C", true, 0, "")
		pdf.CellFormat(numberWidth, 9, "No", "1", 0, "C", true, 0, "")
		pdf.CellFormat(reportWidth, 9, "Laporan Pekerjaan", "1", 1, "C", true, 0, "")
		pdf.SetTextColor(41, 70, 93)
	})
	for index := range reports {
		model.NormalizeWeeklyReport(&reports[index])
	}

	itemHeight := func(description string) float64 {
		height := float64(len(pdf.SplitLines([]byte(text(description)), reportWidth-4)))*5 + 4
		if height < 10 {
			return 10
		}
		return height
	}
	renderReports := func(reports []model.WeeklyReport) {
		pdf.AddPage()
		for roleStart := 0; roleStart < len(reports); {
			roleEnd := roleStart + 1
			for roleEnd < len(reports) && reports[roleEnd].Role == reports[roleStart].Role {
				roleEnd++
			}
			role, roleSegmentStart, showRole := reports[roleStart].Role, pdf.GetY(), true
			finishRoleSegment := func() {
				endY, height := pdf.GetY(), pdf.GetY()-roleSegmentStart
				if height <= 0 {
					return
				}
				pdf.Rect(left, roleSegmentStart, roleWidth, height, "D")
				if showRole {
					lines := pdf.SplitLines([]byte(text(role)), roleWidth-4)
					pdf.SetXY(left+2, roleSegmentStart+(height-float64(len(lines))*5)/2)
					pdf.SetFont("Helvetica", "", 9)
					pdf.MultiCell(roleWidth-4, 5, text(role), "", "C", false)
					showRole = false
				}
				pdf.SetXY(left, endY)
			}
			for categoryStart := roleStart; categoryStart < roleEnd; {
				categoryEnd := categoryStart + 1
				for categoryEnd < roleEnd && reports[categoryEnd].WorkDivision == reports[categoryStart].WorkDivision {
					categoryEnd++
				}
				category := reports[categoryStart].WorkDivision
				categorySegmentStart, showCategory := pdf.GetY(), true
				finishCategorySegment := func() {
					endY, height := pdf.GetY(), pdf.GetY()-categorySegmentStart
					if height <= 0 {
						return
					}
					pdf.Rect(left+roleWidth, categorySegmentStart, categoryWidth, height, "D")
					if showCategory {
						lines := pdf.SplitLines([]byte(text(category)), categoryWidth-4)
						pdf.SetXY(left+roleWidth+2, categorySegmentStart+(height-float64(len(lines))*5)/2)
						pdf.SetFont("Helvetica", "", 9)
						pdf.MultiCell(categoryWidth-4, 5, text(category), "", "C", false)
						showCategory = false
					}
					pdf.SetXY(left, endY)
				}
				newPage := func() {
					finishCategorySegment()
					finishRoleSegment()
					pdf.AddPage()
					roleSegmentStart, categorySegmentStart = pdf.GetY(), pdf.GetY()
				}
				group := ""
				for index := categoryStart; index < categoryEnd; index++ {
					report := reports[index]
					key := report.PrimaryDivision + "\x00" + report.WorkDivision + "\x00" + report.Role + "\x00" + report.Work
					if key != group {
						group = key
						heading := report.Work
						names := []string{}
						for _, pic := range report.PICs {
							name := pic.Name
							if !pic.IsActive {
								name += " (Nonaktif)"
							}
							names = append(names, name)
						}
						if len(names) > 0 {
							heading += "    PIC: " + strings.Join(names, ", ")
						}
						pdf.SetFont("Helvetica", "B", 9)
						lines := pdf.SplitLines([]byte(text(heading)), numberWidth+reportWidth-4)
						headerHeight := float64(len(lines))*5 + 4
						if pdf.GetY()+headerHeight+itemHeight(report.Description) > 282 {
							newPage()
						}
						for len(lines) > 0 {
							count := int((282 - pdf.GetY() - 4) / 5)
							if count < 1 {
								newPage()
								continue
							}
							if count > len(lines) {
								count = len(lines)
							}
							height := float64(count)*5 + 4
							x, y := left+roleWidth+categoryWidth, pdf.GetY()
							pdf.SetFillColor(223, 242, 235)
							pdf.SetFont("Helvetica", "B", 9)
							pdf.Rect(x, y, numberWidth+reportWidth, height, "DF")
							for lineIndex, line := range lines[:count] {
								pdf.SetXY(x+2, y+2+float64(lineIndex)*5)
								pdf.CellFormat(numberWidth+reportWidth-4, 5, string(line), "", 0, "L", false, 0, "")
							}
							pdf.SetXY(left, y+height)
							lines = lines[count:]
							if len(lines) > 0 {
								newPage()
							}
						}
					}
					height := itemHeight(report.Description)
					if pdf.GetY()+height > 282 {
						newPage()
					}
					x, y := left+roleWidth+categoryWidth, pdf.GetY()
					pdf.SetFont("Helvetica", "", 9)
					pdf.SetXY(x, y)
					pdf.CellFormat(numberWidth, height, fmt.Sprint(report.DisplayNumber), "1", 0, "C", false, 0, "")
					pdf.SetXY(x+numberWidth, y)
					pdf.Rect(x+numberWidth, y, reportWidth, height, "D")
					pdf.SetXY(x+numberWidth+2, y+2)
					pdf.MultiCell(reportWidth-4, 5, text(report.Description), "", "L", false)
					pdf.SetXY(left, y+height)
				}
				finishCategorySegment()
				categoryStart = categoryEnd
			}
			finishRoleSegment()
			roleStart = roleEnd
		}
	}
	if division == "Project Engineer" {
		peQA, cihpk := make([]model.WeeklyReport, 0, len(reports)), make([]model.WeeklyReport, 0, len(reports))
		for _, report := range reports {
			if report.Role == "CIHPK" {
				cihpk = append(cihpk, report)
			} else {
				peQA = append(peQA, report)
			}
		}
		if len(peQA) > 0 {
			sectionLabel = "PE & QA"
			renderReports(peQA)
		}
		if len(cihpk) > 0 {
			sectionLabel = "CIHPK"
			renderReports(cihpk)
		}
	} else {
		renderReports(reports)
	}

	var output bytes.Buffer
	err := pdf.Output(&output)
	return output.Bytes(), err
}
