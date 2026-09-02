package model

import "time"

var WeeklyWorkDivisions = map[string]map[string]bool{
	"Project Engineer":     {"MK supervisi": true, "Non-MK supervisi": true, "QA 1": true, "QA 2": true, "JAKG": true, "Manggarai Hydrant": true, "Jember": true},
	"AI Dev":               {"Pengembangan": true, "Pengujian Model": true, "Deployment": true, "Dokumentasi": true},
	"Keuangan":             {"Pembukuan": true, "Pembayaran": true, "Anggaran": true, "Pelaporan": true},
	"Business Development": {"Prospek": true, "Proposal": true, "Kemitraan": true, "Tindak Lanjut": true},
	"HR":                   {"Kehadiran": true, "Rekrutmen": true, "Administrasi": true, "Pengembangan SDM": true},
	"PMO":                  {"Milestone": true, "Rapat": true, "Risiko": true, "Pelaporan": true},
}

var ProjectEngineerJobs = map[string]map[string]map[string]bool{
	"PE":    {"MK supervisi": {"Penataan Kawasan Stasiun Jember": true, "Pengawasan Renovasi Gudang Besar JAKG": true, "Pengawasan Instalasi Hidran BY Manggarai": true}, "Non-MK supervisi": {"MPS PIT": true, "MPS SGR": true, "MPS SGP": true, "SHL JN": true, "SOP Tata Kelola": true, "Studi Daya Beli Gas Bumi": true, "Review FS dan DED Pelabuhan Waipirit": true, "Project Management Office": true, "Reimagine Rumah IPS": true}},
	"QA 1":  {"QA 1": {"MPS PIT": true, "MPS SGR": true, "MPS SGP": true, "SHL JN": true, "SOP Tata Kelola": true, "Studi Daya Beli Gas Bumi": true, "Review FS dan DED Pelabuhan Waipirit": true, "Project Management Office": true, "Reimagine Rumah IPS": true, "Penataan Kawasan Stasiun Jember": true, "Pengawasan Renovasi Gudang Besar JAKG": true, "Pengawasan Instalasi Hidran BY Manggarai": true, "Internal": true}},
	"QA 2":  {"QA 2": {"MPS PIT": true, "MPS SGR": true, "MPS SGP": true, "SHL JN": true, "SOP Tata Kelola": true, "Studi Daya Beli Gas Bumi": true, "Review FS dan DED Pelabuhan Waipirit": true, "Project Management Office": true, "Reimagine Rumah IPS": true, "Penataan Kawasan Stasiun Jember": true, "Pengawasan Renovasi Gudang Besar JAKG": true, "Pengawasan Instalasi Hidran BY Manggarai": true, "Internal": true}},
	"CIHPK": {"JAKG": {"JAKG Supervising Consultant (SPMK Date: March 12, 2026)": true}, "Manggarai Hydrant": {"Supervising Consultant for the Manggarai Balai Yasa Hydrant Installation (SPMK Date: June 8, 2026)": true}, "Jember": {"Jember Station Area Redevelopment": true}},
}

func ValidWeeklyReport(primaryDivision, jobCategory, role, job, description string) bool {
	if description == "" || len(description) > 500 {
		return false
	}
	if primaryDivision != "Project Engineer" {
		return WeeklyWorkDivisions[primaryDivision][jobCategory] && role == "" && job == ""
	}
	return ProjectEngineerJobs[role][jobCategory][job]
}

type WeeklyReportInput struct {
	PrimaryDivision string `json:"primaryDivision"`
	WorkDivision    string `json:"workDivision"`
	Role            string `json:"role"`
	Work            string `json:"work"`
	Description     string `json:"description"`
}
type WeeklyReport struct {
	ID              string    `json:"id"`
	UserID          string    `json:"userId"`
	PrimaryDivision string    `json:"primaryDivision"`
	WorkDivision    string    `json:"workDivision"`
	Role            string    `json:"role"`
	Work            string    `json:"work"`
	Number          int       `json:"number"`
	Description     string    `json:"description"`
	CompletedAt     time.Time `json:"completedAt"`
}
type WeeklyReportUpdateInput struct {
	PrimaryDivision string `json:"primaryDivision"`
	WorkDivision    string `json:"workDivision"`
	Role            string `json:"role"`
	Work            string `json:"work"`
	Description     string `json:"description"`
}
