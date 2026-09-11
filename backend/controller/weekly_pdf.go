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

type exportProgress struct {
	Number  int                `json:"number"`
	Content string             `json:"content"`
	Report  model.WeeklyReport `json:"report"`
}
type exportJob struct {
	Key      string           `json:"key"`
	Name     string           `json:"name"`
	PICs     model.JobPICs    `json:"pics,omitempty"`
	Progress []exportProgress `json:"progress"`
}
type exportCategory struct {
	Name string      `json:"name"`
	Jobs []exportJob `json:"jobs"`
}
type exportRole struct {
	Name       string           `json:"name"`
	Categories []exportCategory `json:"categories"`
}

func exportGroups(reports []model.WeeklyReport) []exportRole {
	reports = documentReports(reports)
	groups := []exportRole{}
	for _, report := range reports {
		if len(groups) == 0 || groups[len(groups)-1].Name != report.Role {
			groups = append(groups, exportRole{Name: report.Role})
		}
		role := &groups[len(groups)-1]
		if len(role.Categories) == 0 || role.Categories[len(role.Categories)-1].Name != report.WorkDivision {
			role.Categories = append(role.Categories, exportCategory{Name: report.WorkDivision})
		}
		category := &role.Categories[len(role.Categories)-1]
		key := report.Work
		if report.JobID != nil {
			key = fmt.Sprint(*report.JobID)
		}
		if len(category.Jobs) == 0 || category.Jobs[len(category.Jobs)-1].Key != key {
			category.Jobs = append(category.Jobs, exportJob{Key: key, Name: report.Work, PICs: report.PICs})
		}
		job := &category.Jobs[len(category.Jobs)-1]
		job.Progress = append(job.Progress, exportProgress{Number: len(job.Progress) + 1, Content: report.Description, Report: report})
	}
	return groups
}

func reportHeading(work string, pics model.JobPICs) string {
	names := []string{}
	for _, pic := range pics {
		name := pic.Name
		if !pic.IsActive {
			name += " (Nonaktif)"
		}
		names = append(names, name)
	}
	heading := work
	if len(names) > 0 {
		heading += "    PIC: " + strings.Join(names, ", ")
	}
	return heading
}

func weeklyPDFDocument(division string, period model.ReportingPeriod, reports []model.WeeklyReport, compression bool) ([]byte, error) {
	groups := exportGroups(reports)
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetCompression(compression)
	pdf.AddUTF8FontFromBytes("Report", "", reportFont)
	pdf.AddUTF8FontFromBytes("Report", "B", reportBoldFont)
	const left, bottom, lineHeight = 15.0, 282.0, 5.0
	widths := []float64{24, 34, 14, 108}
	headerLabel := division
	if division == "Project Engineer" {
		headerLabel = "PE & QA"
	}
	if len(groups) > 0 && groups[0].Name == "CIHPK" {
		headerLabel = "CIHPK"
	}
	pdf.SetMargins(left, 15, left)
	pdf.SetAutoPageBreak(false, 15)
	pdf.AliasNbPages("{nb}")
	pdf.SetFooterFunc(func() {
		pdf.SetY(-10)
		pdf.SetTextColor(91, 111, 128)
		pdf.SetFont("Report", "", 8)
		pdf.CellFormat(180, 5, fmt.Sprint(pdf.PageNo()), "", 0, "C", false, 0, "")
	})
	pdf.SetHeaderFunc(func() {
		pdf.SetXY(left, 8)
		pdf.SetTextColor(91, 111, 128)
		pdf.SetFont("Report", "", 9)
		pdf.CellFormat(90, 5, "Weekly Report", "", 0, "L", false, 0, "")
		pdf.CellFormat(90, 5, headerLabel, "", 0, "R", false, 0, "")
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
	pdf.AddPage()
	contentTop := pdf.GetY()
	activeRole, activeCategory := "", ""
	roleStart, categoryStart := pdf.GetY(), pdf.GetY()
	drawMergedText := func(x, start, end, width float64, value string) {
		fontSize, mergedLineHeight := 9.0, lineHeight
		pdf.SetFont("Report", "", fontSize)
		lines := pdf.SplitText(value, width-4)
		for fontSize > 5 && float64(len(lines))*mergedLineHeight > end-start-2 {
			fontSize--
			mergedLineHeight = fontSize / 2
			pdf.SetFont("Report", "", fontSize)
			lines = pdf.SplitText(value, width-4)
		}
		y := start + max(1, (end-start-float64(len(lines))*mergedLineHeight)/2)
		for _, line := range lines {
			pdf.SetXY(x+2, y)
			pdf.CellFormat(width-4, mergedLineHeight, line, "", 0, "C", false, 0, "")
			y += mergedLineHeight
		}
	}
	closeCategory := func() {
		if activeCategory == "" {
			return
		}
		y := pdf.GetY()
		pdf.Line(left+widths[0], y, left+widths[0]+widths[1], y)
		drawMergedText(left+widths[0], categoryStart, y, widths[1], activeCategory)
		pdf.SetXY(left, y)
		activeCategory = ""
	}
	closeRole := func() {
		if activeRole == "" {
			return
		}
		y := pdf.GetY()
		pdf.Line(left, y, left+widths[0], y)
		drawMergedText(left, roleStart, y, widths[0], activeRole)
		pdf.SetXY(left, y)
		activeRole = ""
	}
	beginRole := func(value string) {
		activeRole, roleStart = value, pdf.GetY()
		pdf.Line(left, roleStart, left+widths[0], roleStart)
	}
	beginCategory := func(value string) {
		activeCategory, categoryStart = value, pdf.GetY()
		pdf.Line(left+widths[0], categoryStart, left+widths[0]+widths[1], categoryStart)
	}
	newPage := func() {
		role, category := activeRole, activeCategory
		closeCategory()
		closeRole()
		pdf.AddPage()
		beginRole(role)
		beginCategory(category)
	}
	rowLines := func(values []string, sizes []float64, bold bool) ([][]string, int) {
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
		return lines, total
	}
	rowHeight := func(values []string, sizes []float64, bold bool) float64 {
		_, lines := rowLines(values, sizes, bold)
		return float64(lines)*lineHeight + 4
	}
	// Role/category cells are merged only within a page. Long content splits into
	// physical rows and repeats its group context after each page break.
	row := func(values []string, sizes []float64, bold bool) {
		lines, total := rowLines(values, sizes, bold)
		style := ""
		if bold {
			style = "B"
		}
		for offset := 0; offset < total; {
			available := int((bottom - pdf.GetY() - 4) / lineHeight)
			if available < 1 {
				newPage()
				continue
			}
			count := min(available, total-offset)
			height := float64(count)*lineHeight + 4
			x, y := left, pdf.GetY()
			pdf.SetFont("Report", style, 9)
			pdf.SetFillColor(255, 255, 255)
			for _, width := range widths[:2] {
				pdf.Rect(x, y, width, height, "F")
				pdf.Line(x, y, x, y+height)
				x += width
			}
			pdf.Line(x, y, x, y+height)
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
				newPage()
			}
		}
	}
	renderedRole := false
	for _, role := range groups {
		if role.Name == "CIHPK" && renderedRole {
			headerLabel = "CIHPK"
			pdf.AddPage()
		}
		beginRole(role.Name)
		for _, category := range role.Categories {
			beginCategory(category.Name)
			for _, job := range category.Jobs {
				heading := reportHeading(job.Name, job.PICs)
				required := rowHeight([]string{heading}, []float64{widths[2] + widths[3]}, true)
				if len(job.Progress) > 0 {
					required += rowHeight([]string{"1", job.Progress[0].Content}, widths[2:], false)
				}
				if required <= bottom-contentTop && bottom-pdf.GetY() < required {
					newPage()
				}
				row([]string{heading}, []float64{widths[2] + widths[3]}, true)
				for _, progress := range job.Progress {
					row([]string{fmt.Sprint(progress.Number), progress.Content}, widths[2:], false)
				}
			}
			closeCategory()
		}
		closeRole()
		renderedRole = true
	}
	var output bytes.Buffer
	err := pdf.Output(&output)
	return output.Bytes(), err
}
