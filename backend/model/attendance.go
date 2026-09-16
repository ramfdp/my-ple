package model

import "time"

type Attendance struct {
	ID        string     `json:"id"`
	Date      string     `json:"date"`
	Mode      string     `json:"mode"`
	ClockIn   time.Time  `json:"clockIn"`
	ClockOut  *time.Time `json:"clockOut"`
	Latitude  *float64   `json:"latitude"`
	Longitude *float64   `json:"longitude"`
}

type AttendanceInput struct {
	Mode      string   `json:"mode"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	Accuracy  *float64 `json:"accuracy"`
}

type AttendanceHistory struct {
	Records          []Attendance `json:"records"`
	Today            *Attendance  `json:"today"`
	Open             *Attendance  `json:"open"`
	TodayDate        string       `json:"todayDate"`
	OfficeConfigured bool         `json:"officeConfigured"`
}
