package model

import "time"

const NotificationZoomBookingCreated = "zoom_booking_created"

type NotificationActor struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type NotificationBooking struct {
	ID       string    `json:"id"`
	ZoomName string    `json:"zoomName"`
	StartAt  time.Time `json:"startAt"`
	EndAt    time.Time `json:"endAt"`
}

type Notification struct {
	ID         string               `json:"id"`
	Type       string               `json:"type"`
	Actor      NotificationActor    `json:"actor"`
	EntityType string               `json:"entityType"`
	EntityID   string               `json:"entityId"`
	Booking    *NotificationBooking `json:"booking,omitempty"`
	CreatedAt  time.Time            `json:"createdAt"`
	ReadAt     *time.Time           `json:"readAt"`
}

type NotificationEvent struct {
	Event        string       `json:"event"`
	Notification Notification `json:"notification"`
}
