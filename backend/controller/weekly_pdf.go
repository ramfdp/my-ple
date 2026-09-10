package controller

import (
	"bytes"
	_ "embed"
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
	c.exportReport(w, r, "pdf")
}

func (c *WeeklyController) ExportDOCX(w http.ResponseWriter, r *http.Request) {
	c.exportReport(w, r, "docx")
}

func (c *WeeklyController) exportReport(w http.ResponseWriter, r *http.Request, format string) {
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
	render, contentType := weeklyPDF, "application/pdf"
	if format == "docx" {
		render, contentType = weeklyDOCX, "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	}
	data, err := render(division, period, reports)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Laporan tidak dapat diekspor."})
		return
	}
	filenameDivision := strings.Trim(unsafeFilename.ReplaceAllString(strings.ReplaceAll(division, " ", "_"), "_"), "_")
	filename := fmt.Sprintf("Weekly_Report_%s_%s_%s.%s", filenameDivision, period.Start.Format("2006-01-02"), period.End.Format("2006-01-02"), format)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(data)
}

func weeklyPDF(division string, period model.ReportingPeriod, reports []model.WeeklyReport) ([]byte, error) {
	return weeklyPDFDocument(division, period, reports, true)
}

//go:embed fonts/DejaVuSansCondensed.ttf
var reportFont []byte

//go:embed fonts/DejaVuSansCondensed-Bold.ttf
var reportBoldFont []byte

// Shared preparation: never mutate the query result or maintain renderer-specific numbering.
func documentReports(reports []model.WeeklyReport) []model.WeeklyReport {
	reports = append([]model.WeeklyReport(nil), reports...)
	for i := range reports {
		model.NormalizeWeeklyReport(&reports[i])
	}
	db.SortReports(reports)
	return reports
}

func reportGroup(r model.WeeklyReport) string {
	return r.PrimaryDivision + "\x00" + r.Role + "\x00" + r.WorkDivision + "\x00" + r.Work
}
func reportHeading(r model.WeeklyReport) string {
	names := []string{}
	for _, pic := range r.PICs {
		name := pic.Name
		if !pic.IsActive {
			name += " (Nonaktif)"
		}
		names = append(names, name)
	}
	heading := r.Work
	if len(names) > 0 {
		heading += "    PIC: " + strings.Join(names, ", ")
	}
	return heading
}

func weeklyPDFDocument(division string, period model.ReportingPeriod, reports []model.WeeklyReport, compression bool) ([]byte, error) {
	reports = documentReports(reports)
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetCompression(compression)
	pdf.AddUTF8FontFromBytes("Report", "", reportFont)
	pdf.AddUTF8FontFromBytes("Report", "B", reportBoldFont)
	const left, bottom, lineHeight = 15.0, 282.0, 5.0
	widths := []float64{20, 42, 14, 104}
	pdf.SetMargins(left, 15, left)
	pdf.SetAutoPageBreak(false, 15)
	pdf.SetHeaderFunc(func() {
		pdf.SetXY(left, 15)
		pdf.SetTextColor(16, 42, 67)
		pdf.SetFont("Report", "B", 16)
		pdf.MultiCell(180, 8, "Weekly Report "+division, "", "L", false)
		pdf.SetFont("Report", "", 10)
		pdf.MultiCell(180, 7, "Tanggal: "+period.Label, "", "L", false)
		pdf.Ln(4)
		pdf.SetFillColor(18, 58, 90)
		pdf.SetTextColor(255, 255, 255)
		pdf.SetFont("Report", "B", 9)
		for i, label := range []string{"Role", "Kategori", "No", "Laporan Pekerjaan"} {
			pdf.CellFormat(widths[i], 9, label, "1", 0, "C", true, 0, "")
		}
		pdf.Ln(9)
		pdf.SetTextColor(41, 70, 93)
	})
	// PDF-specific physical rows, without rowspans. Each column can continue on
	// another page, including an individual entry taller than the entire page.
	row := func(values []string, sizes []float64, bold bool) {
		style := ""
		if bold {
			style = "B"
		}
		pdf.SetFont("Report", style, 9)
		lines := make([][]string, len(values))
		total := 1
		for i, value := range values {
			for _, paragraph := range strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n") {
				if paragraph == "" {
					lines[i] = append(lines[i], "")
				} else {
					lines[i] = append(lines[i], pdf.SplitText(paragraph, sizes[i]-4)...)
				}
			}
			total = max(total, len(lines[i]))
		}
		for offset := 0; offset < total; {
			available := int((bottom - pdf.GetY() - 4) / lineHeight)
			if available < 1 {
				pdf.AddPage()
				continue
			}
			count := min(available, total-offset)
			height := float64(count)*lineHeight + 4
			x, y := left, pdf.GetY()
			pdf.SetFont("Report", style, 9)
			for i, column := range lines {
				pdf.SetFillColor(223, 242, 235)
				fill := "D"
				if bold {
					fill = "DF"
				}
				pdf.Rect(x, y, sizes[i], height, fill)
				// Repeat short identifying cells on continuation pages; long cells split.
				start := offset
				if i < len(lines)-1 && offset >= len(column) {
					start = 0
				}
				for n := 0; n < count && start+n < len(column); n++ {
					pdf.SetXY(x+2, y+2+float64(n)*lineHeight)
					pdf.CellFormat(sizes[i]-4, lineHeight, column[start+n], "", 0, "L", false, 0, "")
				}
				x += sizes[i]
			}
			pdf.SetXY(left, y+height)
			offset += count
			if offset < total {
				pdf.AddPage()
			}
		}
	}
	pdf.AddPage()
	group := ""
	previousRole := ""
	for _, report := range reports {
		if division == "Project Engineer" && report.Role == "CIHPK" && previousRole != "" && previousRole != "CIHPK" {
			pdf.AddPage()
		}
		previousRole = report.Role
		if key := reportGroup(report); key != group {
			group = key
			// Reserve only the next line, never the full height of an arbitrarily long entry.
			if bottom-pdf.GetY() < 23 {
				pdf.AddPage()
			}
			row([]string{report.Role, report.WorkDivision, reportHeading(report)}, []float64{20, 42, 118}, true)
		}
		row([]string{report.Role, report.WorkDivision, fmt.Sprint(report.DisplayNumber), report.Description}, widths, false)
	}
	var output bytes.Buffer
	err := pdf.Output(&output)
	return output.Bytes(), err
}
