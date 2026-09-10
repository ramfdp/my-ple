package controller

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"

	"my-ple-backend/model"
)

// A real Open Packaging Conventions document, using the standard library.
// Rows may split across pages; no cantSplit/rowspan restrictions on long text.
func weeklyDOCX(division string, period model.ReportingPeriod, reports []model.WeeklyReport) ([]byte, error) {
	var document strings.Builder
	paragraph := func(value string, bold bool) string {
		var p strings.Builder
		p.WriteString("<w:p><w:pPr><w:spacing w:after=\"0\"/></w:pPr><w:r><w:rPr><w:sz w:val=\"20\"/>")
		if bold {
			p.WriteString("<w:b/>")
		}
		p.WriteString("</w:rPr>")
		for i, line := range strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n") {
			if i > 0 {
				p.WriteString("<w:br/>")
			}
			p.WriteString("<w:t xml:space=\"preserve\">")
			_ = xml.EscapeText(&p, []byte(line))
			p.WriteString("</w:t>")
		}
		p.WriteString("</w:r></w:p>")
		return p.String()
	}
	document.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`)
	document.WriteString(paragraph("Weekly Report "+division, true))
	document.WriteString(paragraph("Tanggal: "+period.Label, false))
	document.WriteString(`<w:tbl><w:tblPr><w:tblW w:w="10206" w:type="dxa"/><w:tblLayout w:type="fixed"/><w:tblBorders><w:top w:val="single" w:sz="4"/><w:left w:val="single" w:sz="4"/><w:bottom w:val="single" w:sz="4"/><w:right w:val="single" w:sz="4"/><w:insideH w:val="single" w:sz="4"/><w:insideV w:val="single" w:sz="4"/></w:tblBorders></w:tblPr><w:tblGrid><w:gridCol w:w="1134"/><w:gridCol w:w="2381"/><w:gridCol w:w="794"/><w:gridCol w:w="5897"/></w:tblGrid>`)
	row := func(values []string, header, group bool) {
		document.WriteString("<w:tr>")
		if header {
			document.WriteString("<w:trPr><w:tblHeader/></w:trPr>")
		}
		sizes := []int{1134, 2381, 794, 5897}
		for i, value := range values {
			size := sizes[i]
			span := ""
			if group && i == 2 {
				size = 6691
				span = `<w:gridSpan w:val="2"/>`
			}
			fmt.Fprintf(&document, `<w:tc><w:tcPr><w:tcW w:w="%d" w:type="dxa"/>%s<w:vAlign w:val="top"/></w:tcPr>%s</w:tc>`, size, span, paragraph(value, header || group))
		}
		document.WriteString("</w:tr>")
	}
	row([]string{"Role", "Kategori", "No", "Laporan Pekerjaan"}, true, false)
	group := ""
	for _, r := range documentReports(reports) {
		if key := reportGroup(r); key != group {
			group = key
			row([]string{r.Role, r.WorkDivision, reportHeading(r)}, false, true)
		}
		row([]string{r.Role, r.WorkDivision, fmt.Sprint(r.DisplayNumber), r.Description}, false, false)
	}
	document.WriteString(`</w:tbl><w:p/><w:sectPr><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="850" w:right="850" w:bottom="850" w:left="850" w:header="425" w:footer="425" w:gutter="0"/></w:sectPr></w:body></w:document>`)
	var output bytes.Buffer
	archive := zip.NewWriter(&output)
	for _, part := range []struct{ name, body string }{
		{"[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`},
		{"_rels/.rels", `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`},
		{"word/document.xml", document.String()},
	} {
		entry, err := archive.Create(part.name)
		if err != nil {
			return nil, err
		}
		if _, err = entry.Write([]byte(part.body)); err != nil {
			return nil, err
		}
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}
