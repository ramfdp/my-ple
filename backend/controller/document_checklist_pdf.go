package controller

import (
	"bytes"
	"fmt"
	"my-ple-backend/model"
	"strings"

	"github.com/go-pdf/fpdf"
)

func checklistPDF(job string, tables []*model.DocumentChecklist) ([]byte, error) {
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.AddUTF8FontFromBytes("Report", "", reportFont)
	pdf.AddUTF8FontFromBytes("Report", "B", reportBoldFont)
	pdf.SetMargins(10, 10, 10)
	pdf.SetAutoPageBreak(false, 10)
	pdf.SetDrawColor(120, 135, 130)
	pdf.SetFillColor(221, 239, 229)
	const width, bottom, line = 277.0, 200.0, 4.0
	cell := func(x, y, w, h float64, text string, header bool) {
		style := "D"
		if header {
			style = "DF"
		}
		pdf.Rect(x, y, w, h, style)
		lines := pdf.SplitText(text, w-4)
		start := y + (h-float64(len(lines))*line)/2
		for i, value := range lines {
			pdf.SetXY(x+2, start+float64(i)*line)
			pdf.CellFormat(w-4, line, value, "", 0, "C", false, 0, "")
		}
	}
	signerCell := func(x, y, w, h float64, signer model.ChecklistSigner) {
		pdf.Rect(x, y, w, h, "DF")
		pdf.SetFont("Report", "B", 8)
		position := pdf.SplitText(signer.Position, w-4)
		pdf.SetFont("Report", "", 8)
		name := pdf.SplitText(signer.Name, w-4)
		start := y + (h-float64(len(name)+len(position))*line)/2
		pdf.SetFont("Report", "B", 8)
		for _, part := range [][]string{position, name} {
			for _, value := range part {
				pdf.SetXY(x+2, start)
				pdf.CellFormat(w-4, line, value, "", 0, "C", false, 0, "")
				start += line
			}
			pdf.SetFont("Report", "", 8)
		}
	}
	newPage := func() {
		pdf.AddPage()
		pdf.SetFont("Report", "B", 11)
		pdf.MultiCell(width, 6, job, "", "C", false)
		pdf.Ln(2)
	}
	for _, table := range tables {
		header := func(field, fallback string) string {
			if value := table.HeaderLabels[field]; value != "" {
				return value
			}
			return fallback
		}
		signers := []model.ChecklistSigner{}
		for _, kind := range []string{"internal", "external"} {
			for _, s := range table.Signers {
				if !s.Archived && s.Type == kind {
					signers = append(signers, s)
				}
			}
		}
		// Wide checklists continue in column panels instead of shrinking readable text.
		for first := 0; first < len(signers) || first == 0; {
			count := min(5, len(signers)-first)
			signatureSpace := 120.0
			if table.RequireDigitalSignature {
				signatureSpace -= 18
			}
			pdf.SetFont("Report", "B", 8)
			signerWidth, signerHeight := signatureSpace, 8.0
			for {
				if count > 0 {
					signerWidth = signatureSpace / float64(count)
				}
				signerHeight = 8
				for _, s := range signers[first : first+count] {
					pdf.SetFont("Report", "", 8)
					nameLines := len(pdf.SplitText(s.Name, signerWidth-4))
					pdf.SetFont("Report", "B", 8)
					signerHeight = max(signerHeight, float64(nameLines+len(pdf.SplitText(s.Position, signerWidth-4)))*line+4)
				}
				if signerHeight <= 90 || count <= 1 {
					break
				}
				count--
			}
			panel := signers[first : first+count]
			widths := []float64{12, 46}
			labels := []string{header("number", "No."), header("documentName", "Dokumen")}
			digitalIndex := -1
			if table.RequireDigitalSignature {
				digitalIndex = len(widths)
				widths = append(widths, 18)
				labels = append(labels, header("digitalSignature", "TTD Digital"))
			}
			signatureStart := len(widths)
			for _, s := range panel {
				widths = append(widths, signerWidth)
				labels = append(labels, s.Name+"\n"+s.Position)
			}
			widths = append(widths, 28, 26, 45)
			labels = append(labels, header("documentPosition", "Posisi Dokumen"), header("sentDate", "Tanggal Kirim Dokumen"), header("notes", "Catatan"))
			if count == 0 {
				widths[1] += signatureSpace
			}
			title := table.Title
			if first > 0 {
				title += fmt.Sprintf(" (lanjutan kolom %d-%d)", first+1, first+count)
			}
			titleHeight := max(8.0, float64(len(pdf.SplitText(title, width-4)))*line+4)
			headerHeight := 8.0
			for i, label := range labels {
				if i < signatureStart || i >= signatureStart+count {
					headerHeight = max(headerHeight, float64(len(pdf.SplitText(label, widths[i]-4)))*line+4)
				}
			}
			if count > 0 {
				headerHeight = max(headerHeight, 8+signerHeight)
			}
			drawHeader := func() {
				pdf.SetFont("Report", "B", 8)
				y := pdf.GetY()
				cell(10, y, width, titleHeight, title, true)
				y += titleHeight
				x := 10.0
				for i, label := range labels {
					if i < signatureStart || i >= signatureStart+count {
						cell(x, y, widths[i], headerHeight, label, true)
					}
					x += widths[i]
				}
				if count > 0 {
					x = 10
					for _, w := range widths[:signatureStart] {
						x += w
					}
					for start := 0; start < count; {
						end := start + 1
						for end < count && panel[end].Type == panel[start].Type {
							end++
						}
						group := header("internalGroup", "Paraf Hardcopy")
						if panel[start].Type == "external" {
							group = header("externalGroup", "TTD Hardcopy")
						}
						cell(x+float64(start)*signerWidth, y, float64(end-start)*signerWidth, 8, group, true)
						start = end
					}
					for i, s := range panel {
						signerCell(x+float64(i)*signerWidth, y+8, signerWidth, headerHeight-8, s)
					}
				}
				pdf.SetXY(10, y+headerHeight)
				pdf.SetFont("Report", "", 8)
			}
			valuesFor := func(item model.ChecklistItem) ([]string, map[int]bool) {
				values := []string{item.Number, item.DocumentName}
				checks := map[int]bool{}
				if digitalIndex >= 0 {
					checks[digitalIndex] = item.DigitalSigned
					values = append(values, "")
				}
				for _, s := range panel {
					checks[len(values)] = item.Signatures[s.ID]
					values = append(values, "")
				}
				date := ""
				if item.SentDate != nil {
					date = *item.SentDate
				}
				values = append(values, item.DocumentPosition, date, item.Notes)
				return values, checks
			}
			pdf.SetFont("Report", "B", 11)
			freshBodyStart := 12 + float64(len(pdf.SplitText(job, width-2)))*6 + titleHeight + headerHeight
			pdf.SetFont("Report", "", 8)
			firstHeight := 8.0
			if len(table.Items) > 0 {
				values, _ := valuesFor(table.Items[0])
				for i, value := range values {
					firstHeight = max(firstHeight, float64(len(pdf.SplitText(value, widths[i]-4)))*line+4)
				}
				firstHeight = min(firstHeight, bottom-freshBodyStart)
			}
			if pdf.PageNo() == 0 {
				newPage()
			} else {
				pdf.Ln(5)
			}
			if pdf.GetY()+titleHeight+headerHeight+firstHeight > bottom {
				newPage()
			}
			drawHeader()
			bodyStart := pdf.GetY()
			for _, item := range table.Items {
				values, checks := valuesFor(item)
				lines := make([][]string, len(values))
				longest := 1
				for i, value := range values {
					lines[i] = pdf.SplitText(value, widths[i]-4)
					longest = max(longest, len(lines[i]))
				}
				rowHeight := float64(longest)*line + 4
				// Keep a row together if it fits on a fresh page; only oversized rows split.
				if pdf.GetY()+rowHeight > bottom && (rowHeight <= bottom-freshBodyStart || pdf.GetY() > bodyStart) {
					newPage()
					drawHeader()
					bodyStart = pdf.GetY()
				}
				for offset := 0; offset < longest; {
					available := int((bottom - pdf.GetY() - 4) / line)
					if available < 1 {
						newPage()
						drawHeader()
						bodyStart = pdf.GetY()
						available = int((bottom - bodyStart - 4) / line)
					}
					if available < 1 {
						return nil, fmt.Errorf("checklist heading exceeds page height")
					}
					n := min(available, longest-offset)
					h := float64(n)*line + 4
					x, y := 10.0, pdf.GetY()
					for i, column := range lines {
						text := ""
						if offset < len(column) {
							text = strings.Join(column[offset:min(offset+n, len(column))], "\n")
						}
						cell(x, y, widths[i], h, text, false)
						if checked, ok := checks[i]; ok {
							cx, cy := x+widths[i]/2-1.6, y+h/2-1.6
							pdf.Rect(cx, cy, 3.2, 3.2, "D")
							if checked {
								red, green, blue := pdf.GetDrawColor()
								lineWidth := pdf.GetLineWidth()
								pdf.SetDrawColor(0, 0, 0)
								pdf.SetLineWidth(0.7)
								pdf.Line(cx+.5, cy+1.6, cx+1.3, cy+2.5)
								pdf.Line(cx+1.3, cy+2.5, cx+2.8, cy+.6)
								pdf.SetLineWidth(lineWidth)
								pdf.SetDrawColor(red, green, blue)
							}
						}
						x += widths[i]
					}
					pdf.SetXY(10, y+h)
					offset += n
				}
			}
			if len(table.Items) == 0 {
				y := pdf.GetY()
				cell(10, y, width, 8, "Belum ada dokumen.", false)
				pdf.SetXY(10, y+8)
			}
			if first+count >= len(signers) {
				break
			}
			first += count
		}
	}
	if pdf.PageNo() == 0 {
		newPage()
	}
	var output bytes.Buffer
	err := pdf.Output(&output)
	return output.Bytes(), err
}
