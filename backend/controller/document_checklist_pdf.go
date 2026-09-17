package controller

import (
	"bytes"
	"fmt"
	"my-ple-backend/model"
	"strings"

	"github.com/go-pdf/fpdf"
)

// Signature names stay in a wrapping cell so any number of signers fits the page.
func checklistPDF(job string, tables []*model.DocumentChecklist) ([]byte, error) {
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.AddUTF8FontFromBytes("Report", "", reportFont)
	pdf.AddUTF8FontFromBytes("Report", "B", reportBoldFont)
	pdf.SetMargins(10, 10, 10)
	pdf.SetAutoPageBreak(false, 10)
	widths := []float64{12, 53, 20, 77, 30, 25, 60}
	for _, table := range tables {
		newPage := func() {
			pdf.AddPage()
			pdf.SetFont("Report", "B", 12)
			pdf.MultiCell(277, 6, job, "", "L", false)
			pdf.SetFont("Report", "B", 10)
			pdf.MultiCell(277, 5, table.Title, "", "L", false)
			pdf.SetFont("Report", "B", 8)
			for i, label := range []string{"No.", "Dokumen", "TTD Digital", "TTD Hardcopy", "Posisi Dokumen", "Tanggal Kirim", "Catatan"} {
				pdf.CellFormat(widths[i], 8, label, "1", 0, "C", false, 0, "")
			}
			pdf.Ln(8)
			pdf.SetFont("Report", "", 8)
		}
		newPage()
		for index, item := range table.Items {
			mark := func(signed bool) string {
				if signed {
					return "V"
				}
				return "X"
			}
			digital := "-"
			if table.RequireDigitalSignature {
				digital = mark(item.DigitalSigned)
			}
			signatures := []string{}
			for _, signer := range table.Signers {
				if signer.Archived {
					continue
				}
				signatures = append(signatures, fmt.Sprintf("%s: %s — %s (%s)", mark(item.Signatures[signer.ID]), signer.Name, signer.Position, signer.Type))
			}
			date := ""
			if item.SentDate != nil {
				date = *item.SentDate
			}
			values := []string{fmt.Sprint(index + 1), item.DocumentName, digital, strings.Join(signatures, "\n"), item.DocumentPosition, date, item.Notes}
			lines := make([][]string, len(values))
			count := 1
			for i, value := range values {
				lines[i] = pdf.SplitText(value, widths[i]-4)
				count = max(count, len(lines[i]))
			}
			for offset := 0; offset < count; {
				available := int((195 - pdf.GetY() - 2) / 4)
				if available < 1 {
					newPage()
					available = int((195 - pdf.GetY() - 2) / 4)
				}
				if available < 1 {
					return nil, fmt.Errorf("checklist heading exceeds page height")
				}
				n := min(available, count-offset)
				x, y := 10.0, pdf.GetY()
				for i, column := range lines {
					pdf.Rect(x, y, widths[i], float64(n)*4+2, "D")
					for k := 0; k < n && offset+k < len(column); k++ {
						pdf.SetXY(x+2, y+1+float64(k)*4)
						pdf.CellFormat(widths[i]-4, 4, column[offset+k], "", 0, "L", false, 0, "")
					}
					x += widths[i]
				}
				pdf.SetXY(10, y+float64(n)*4+2)
				offset += n
			}
		}
		if len(table.Items) == 0 {
			pdf.MultiCell(277, 6, "Belum ada dokumen.", "1", "L", false)
		}
	}
	var output bytes.Buffer
	err := pdf.Output(&output)
	return output.Bytes(), err
}
