package model

import (
	"errors"
	"fmt"
	"time"
)

const (
	OvertimeWorkday = "WORKDAY"
	OvertimeHoliday = "HOLIDAY"
)

var ErrInvalidOvertime = errors.New("Data lembur tidak valid.")

func overtimeClock(value string) (int, error) {
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return 0, ErrInvalidOvertime
	}
	return parsed.Hour()*60 + parsed.Minute(), nil
}

func CalculateOvertime(start, end, dayType string) (actualMinutes, weightedMinutes int, err error) {
	startMinute, err := overtimeClock(start)
	if err != nil {
		return 0, 0, err
	}
	endMinute, err := overtimeClock(end)
	if err != nil || startMinute == endMinute || (dayType != OvertimeWorkday && dayType != OvertimeHoliday) {
		return 0, 0, ErrInvalidOvertime
	}
	if endMinute < startMinute {
		endMinute += 24 * 60
	}
	if dayType == OvertimeHoliday {
		actualMinutes = endMinute - startMinute
	} else {
		if startMinute < 8*60 {
			actualMinutes += min(endMinute, 8*60) - startMinute
		}
		if endMinute > 17*60 {
			actualMinutes += endMinute - max(startMinute, 17*60)
		}
	}
	actualMinutes = ((actualMinutes + 15) / 30) * 30
	weightedMinutes = actualMinutes
	if dayType == OvertimeHoliday {
		weightedMinutes *= 2
	}
	return actualMinutes, weightedMinutes, nil
}

func ParseOvertimePeriod(period string) (time.Time, error) {
	location, _ := time.LoadLocation(ReportingTimezone)
	parsed, err := time.ParseInLocation("2006-01", period, location)
	if err != nil || parsed.Format("2006-01") != period {
		return time.Time{}, fmt.Errorf("%w: periode", ErrInvalidOvertime)
	}
	return parsed, nil
}

func OvertimeSubmissionOpen(at time.Time, period string) bool {
	month, err := ParseOvertimePeriod(period)
	if err != nil {
		return false
	}
	location := month.Location()
	at = at.In(location)
	start := time.Date(month.Year(), month.Month(), 26, 0, 0, 0, 0, location)
	next := month.AddDate(0, 1, 0)
	end := time.Date(next.Year(), next.Month(), 6, 0, 0, 0, 0, location)
	return !at.Before(start) && at.Before(end)
}
