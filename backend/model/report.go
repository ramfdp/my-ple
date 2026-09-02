package model

import "time"

var WeeklyWorkDivisions = map[string]map[string]bool{
	"Project Engineer":     {"Supervisory": true, "Non-Supervisory": true, "QA 1": true, "QA 2": true, "JAKG": true, "Manggarai Hydrant": true, "Jember": true},
	"AI Dev":               {"Pengembangan": true, "Pengujian Model": true, "Deployment": true, "Dokumentasi": true},
	"Keuangan":             {"Pembukuan": true, "Pembayaran": true, "Anggaran": true, "Pelaporan": true},
	"Business Development": {"Prospek": true, "Proposal": true, "Kemitraan": true, "Tindak Lanjut": true},
	"HR":                   {"Kehadiran": true, "Rekrutmen": true, "Administrasi": true, "Pengembangan SDM": true},
	"PMO":                  {"Milestone": true, "Rapat": true, "Risiko": true, "Pelaporan": true},
}

var ProjectEngineerJobs = map[string]map[string]map[string]bool{
	"PE":    {"Supervisory": {"Jember Station Area Planning": true, "Supervision of the JAKG Large Warehouse Renovation": true, "Supervision of the BY Manggarai Hydrant Installation": true}, "Non-Supervisory": {"MPS PIT": true, "MPS SGR": true, "MPS SGP": true, "SHL JN": true, "SOP Governance": true, "Natural Gas Purchasing Power Study": true, "Waipirit Port FS and DED Review": true, "Project Management Office": true, "Reimagine IPS Housing": true}},
	"QA 1":  {"QA 1": {"MPS PIT": true, "MPS SGR": true, "MPS SGP": true, "SHL JN": true, "Governance SOP": true, "Natural Gas Purchasing Power Study": true, "Waipirit Port FS and DED Review": true, "Project Management Office": true, "Reimagine IPS House": true, "Jember Station Area Redevelopment": true, "Supervision of JAKG Large Warehouse Renovation": true, "Supervision of BY Manggarai Hydrant Installation": true, "Internal": true}},
	"QA 2":  {"QA 2": {"MPS PIT": true, "MPS SGR": true, "MPS SGP": true, "SHL JN": true, "Governance SOP": true, "Natural Gas Purchasing Power Study": true, "Waipirit Port FS and DED Review": true, "Project Management Office": true, "Reimagine IPS House": true, "Jember Station Area Redevelopment": true, "Supervision of JAKG Large Warehouse Renovation": true, "Supervision of BY Manggarai Hydrant Installation": true, "Internal": true}},
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
