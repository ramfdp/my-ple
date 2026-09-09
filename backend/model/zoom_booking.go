package model

import (
	"net/url"
	"strings"
	"time"
)

type ZoomBookingInput struct {
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	ZoomResourceID int       `json:"zoomResourceId"`
	StartAt        time.Time `json:"startAt"`
	EndAt          time.Time `json:"endAt"`
}

type ZoomBooking struct {
	ID           string       `json:"id"`
	UserID       string       `json:"userId"`
	CreatedBy    string       `json:"createdBy"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	ZoomResource ZoomResource `json:"zoomResource"`
	StartAt      time.Time    `json:"startAt"`
	EndAt        time.Time    `json:"endAt"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
}

type ZoomResource struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	MeetingURL  string    `json:"meetingUrl"`
	MeetingID   string    `json:"meetingId"`
	HostKey     string    `json:"hostKey"`
	Passcode    string    `json:"passcode"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type ZoomResourceUpdateInput struct {
	Description string `json:"description"`
	MeetingURL  string `json:"meetingUrl"`
	MeetingID   string `json:"meetingId"`
	HostKey     string `json:"hostKey"`
	Passcode    string `json:"passcode"`
}

func validHTTPSURL(value string) bool {
	if value == "" {
		return true
	}
	u, err := url.ParseRequestURI(value)
	return err == nil && u.Scheme == "https" && u.Host != "" && len(value) <= 2048
}

func ValidZoomBooking(input ZoomBookingInput) bool {
	return strings.TrimSpace(input.Title) != "" && len(input.Title) <= 200 && len(input.Description) <= 1000 && input.ZoomResourceID >= 1 && input.ZoomResourceID <= 3 && input.StartAt.Before(input.EndAt)
}

func ValidZoomResourceUpdate(input ZoomResourceUpdateInput) bool {
	return len(input.Description) <= 1000 && len(input.MeetingID) <= 100 && len(input.HostKey) <= 100 && len(input.Passcode) <= 100 && validHTTPSURL(input.MeetingURL)
}
