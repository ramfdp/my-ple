package workspace

import (
	"my-ple-backend/model"
	"time"
)

// Indicators, not employee productivity. Weights deliberately remain in one service.
const (
	DueSoonDays     = 7
	DueSoonWeight   = 2
	OverdueWeight   = 3
	BusyScore       = 10
	OverloadedScore = 18
)

type Workload struct {
	Member            model.WorkspaceMember `json:"member"`
	ActiveTaskCount   int                   `json:"activeTaskCount"`
	HighPriorityCount int                   `json:"highPriorityCount"`
	DueSoonCount      int                   `json:"dueSoonCount"`
	OverdueCount      int                   `json:"overdueCount"`
	OverlapCount      int                   `json:"overlapCount"`
	WorkloadScore     int                   `json:"workloadScore"`
	WorkloadLevel     string                `json:"workloadLevel"`
	Tasks             []model.WorkItem      `json:"tasks"`
}

func Calculate(tasks []model.WorkItem, now time.Time) Workload {
	start, _ := model.ReportingWeek(now)
	today := now.In(start.Location()).Format("2006-01-02")
	soon := now.In(start.Location()).AddDate(0, 0, DueSoonDays).Format("2006-01-02")
	result := Workload{Tasks: []model.WorkItem{}, WorkloadLevel: "AVAILABLE"}
	overlaps := 0
	for _, t := range tasks {
		if t.Status == model.WorkDone {
			continue
		}
		result.Tasks = append(result.Tasks, t)
		result.ActiveTaskCount++
		weight := map[string]int{model.PriorityLow: 1, model.PriorityMedium: 2, model.PriorityHigh: 3, model.PriorityUrgent: 5}[t.Priority]
		result.WorkloadScore += weight
		if t.Priority == model.PriorityHigh || t.Priority == model.PriorityUrgent {
			result.HighPriorityCount++
		}
		if t.DueDate != "" && t.DueDate < today {
			result.OverdueCount++
			result.WorkloadScore += OverdueWeight
		} else if t.DueDate != "" && t.DueDate <= soon {
			result.DueSoonCount++
			result.WorkloadScore += DueSoonWeight
		}
		if t.StartDate != "" && t.DueDate != "" {
			a, e1 := time.Parse("2006-01-02", t.StartDate)
			b, e2 := time.Parse("2006-01-02", t.DueDate)
			if e1 == nil && e2 == nil {
				result.WorkloadScore += min(4, int(b.Sub(a).Hours()/24)/14)
			}
		}
		if (t.StartDate == "" || t.StartDate <= today) && (t.DueDate == "" || t.DueDate >= today) {
			overlaps++
		}
	}
	result.OverlapCount = max(0, overlaps-1)
	result.WorkloadScore += min(4, result.OverlapCount)
	switch {
	case result.WorkloadScore >= OverloadedScore:
		result.WorkloadLevel = "OVERLOADED"
	case result.WorkloadScore >= BusyScore:
		result.WorkloadLevel = "BUSY"
	case result.ActiveTaskCount > 0:
		result.WorkloadLevel = "NORMAL"
	}
	return result
}
