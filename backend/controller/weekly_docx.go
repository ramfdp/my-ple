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
	groups := exportGroups(reports)
	var document strings.Builder
	paragraph := func(value string, bold, centered bool, color string) string {
		var p strings.Builder
		p.WriteString("<w:p><w:pPr><w:spacing w:after=\"0\"/>")
		if centered {
			p.WriteString(`<w:jc w:val="center"/>`)
		}
		p.WriteString("</w:pPr><w:r><w:rPr><w:sz w:val=\"20\"/>")
		if bold {
			p.WriteString("<w:b/>")
		}
		if color != "" {
			fmt.Fprintf(&p, `<w:color w:val="%s"/>`, color)
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
	document.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><w:body>`)
	tableStart := `<w:tbl><w:tblPr><w:tblW w:w="10206" w:type="dxa"/><w:tblLayout w:type="fixed"/><w:tblBorders><w:top w:val="single" w:sz="4"/><w:left w:val="single" w:sz="4"/><w:bottom w:val="single" w:sz="4"/><w:right w:val="single" w:sz="4"/><w:insideH w:val="single" w:sz="4"/><w:insideV w:val="single" w:sz="4"/></w:tblBorders></w:tblPr><w:tblGrid><w:gridCol w:w="1361"/><w:gridCol w:w="1939"/><w:gridCol w:w="794"/><w:gridCol w:w="6112"/></w:tblGrid>`
	document.WriteString(tableStart)
	row := func(values []string, header, group bool, roleMerge, categoryMerge string) {
		document.WriteString("<w:tr>")
		if header {
			document.WriteString("<w:trPr><w:tblHeader/></w:trPr>")
		}
		sizes := []int{1361, 1939, 794, 6112}
		for i, value := range values {
			size := sizes[i]
			span := ""
			if group && i == 2 {
				size = 6906
				span = `<w:gridSpan w:val="2"/>`
			}
			merge := ""
			if !header && i == 0 {
				merge = `<w:vMerge w:val="` + roleMerge + `"/>`
			}
			if !header && i == 1 {
				merge = `<w:vMerge w:val="` + categoryMerge + `"/>`
			}
			shading := ""
			if header {
				shading = `<w:shd w:fill="123A5A"/>`
			} else if group && i >= 2 {
				shading = `<w:shd w:fill="DFF2EB"/>`
			}
			color := ""
			if header {
				color = "FFFFFF"
			}
			fmt.Fprintf(&document, `<w:tc><w:tcPr><w:tcW w:w="%d" w:type="dxa"/>%s%s%s<w:vAlign w:val="center"/></w:tcPr>%s</w:tc>`, size, span, merge, shading, paragraph(value, header || group, header || (!group && i < 3), color))
		}
		document.WriteString("</w:tr>")
	}
	row([]string{"Role", "Kategori", "No", "Laporan Pekerjaan"}, true, false, "", "")
	renderedRole := false
	for _, role := range groups {
		if role.Name == "CIHPK" && renderedRole {
			document.WriteString(`</w:tbl><w:p><w:pPr><w:sectPr><w:headerReference w:type="default" r:id="rId1"/><w:footerReference w:type="default" r:id="rId2"/><w:type w:val="nextPage"/></w:sectPr></w:pPr></w:p>`)
			document.WriteString(tableStart)
			row([]string{"Role", "Kategori", "No", "Laporan Pekerjaan"}, true, false, "", "")
		}
		roleFirst := true
		for _, category := range role.Categories {
			categoryFirst := true
			for _, job := range category.Jobs {
				roleMerge, categoryMerge := "continue", "continue"
				roleName, categoryName := "", ""
				if roleFirst {
					roleMerge, roleName, roleFirst = "restart", role.Name, false
				}
				if categoryFirst {
					categoryMerge, categoryName, categoryFirst = "restart", category.Name, false
				}
				row([]string{roleName, categoryName, reportHeading(job.Name, job.PICs)}, false, true, roleMerge, categoryMerge)
				for _, progress := range job.Progress {
					row([]string{"", "", fmt.Sprint(progress.Number), progress.Content}, false, false, "continue", "continue")
				}
			}
		}
		renderedRole = true
	}
	finalHeaderID := "rId1"
	if len(groups) > 1 && groups[len(groups)-1].Name == "CIHPK" {
		finalHeaderID = "rId3"
	}
	fmt.Fprintf(&document, `</w:tbl><w:p/><w:sectPr><w:headerReference w:type="default" r:id="%s"/><w:footerReference w:type="default" r:id="rId2"/><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="2400" w:right="850" w:bottom="850" w:left="850" w:header="425" w:footer="425" w:gutter="0"/></w:sectPr></w:body></w:document>`, finalHeaderID)
	headerLabel := "PE & QA"
	if len(groups) > 0 && groups[0].Name == "CIHPK" {
		headerLabel = "CIHPK"
	}
	escape := func(value string) string {
		var b strings.Builder
		_ = xml.EscapeText(&b, []byte(value))
		return b.String()
	}
	header := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:hdr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:tbl><w:tblPr><w:tblW w:w="10206" w:type="dxa"/></w:tblPr><w:tblGrid><w:gridCol w:w="5103"/><w:gridCol w:w="5103"/></w:tblGrid><w:tr><w:tc><w:p><w:r><w:rPr><w:color w:val="5B6F80"/><w:sz w:val="18"/></w:rPr><w:t>Weekly Report</w:t></w:r></w:p></w:tc><w:tc><w:p><w:pPr><w:jc w:val="right"/></w:pPr><w:r><w:rPr><w:color w:val="5B6F80"/><w:sz w:val="18"/></w:rPr><w:t>%s</w:t></w:r></w:p></w:tc></w:tr></w:tbl><w:p><w:r><w:rPr><w:b/><w:color w:val="102A43"/><w:sz w:val="32"/></w:rPr><w:t>Weekly Report %s</w:t></w:r></w:p><w:p><w:pPr><w:spacing w:after="120"/></w:pPr><w:r><w:rPr><w:color w:val="102A43"/><w:sz w:val="20"/></w:rPr><w:t>Tanggal: %s</w:t></w:r></w:p></w:hdr>`, escape(headerLabel), escape(division), escape(period.Label))
	cihpkHeader := strings.Replace(header, ">PE &amp; QA<", ">CIHPK<", 1)
	footer := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:ftr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:fldChar w:fldCharType="begin"/></w:r><w:r><w:instrText xml:space="preserve"> PAGE </w:instrText></w:r><w:r><w:fldChar w:fldCharType="end"/></w:r></w:p></w:ftr>`
	var output bytes.Buffer
	archive := zip.NewWriter(&output)
	for _, part := range []struct{ name, body string }{
		{"[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/><Override PartName="/word/header.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml"/><Override PartName="/word/header-cihpk.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml"/><Override PartName="/word/footer.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.footer+xml"/></Types>`},
		{"_rels/.rels", `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`},
		{"word/document.xml", document.String()},
		{"word/_rels/document.xml.rels", `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/header" Target="header.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/footer" Target="footer.xml"/><Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/header" Target="header-cihpk.xml"/></Relationships>`},
		{"word/header.xml", header},
		{"word/header-cihpk.xml", cihpkHeader},
		{"word/footer.xml", footer},
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
