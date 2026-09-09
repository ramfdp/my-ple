package realtime

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"my-ple-backend/model"
)

func TestZoomBroadcastExcludesCreator(t *testing.T) {
	hub := NewHub("")
	ready := make(chan struct{}, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		hub.add(r.URL.Query().Get("id"), c)
		ready <- struct{}{}
	}))
	defer server.Close()
	connect := func(id string) *websocket.Conn {
		t.Helper()
		c, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"?id="+id, nil)
		if err != nil {
			t.Fatal(err)
		}
		<-ready
		t.Cleanup(func() { c.Close(); hub.CloseUser(id) })
		return c
	}
	creator, other := connect("creator"), connect("other")
	booking := model.Notification{ID: "booking-event", Type: model.NotificationZoomBookingCreated, Actor: model.NotificationActor{ID: "creator", Name: "Budi"}, Booking: &model.NotificationBooking{ZoomName: "Zoom 1 - NP", Description: "Coordination", StartAt: time.Now(), EndAt: time.Now().Add(time.Hour)}}
	hub.Broadcast(booking, "creator")
	other.SetReadDeadline(time.Now().Add(time.Second))
	var event model.NotificationEvent
	if err := other.ReadJSON(&event); err != nil {
		t.Fatal(err)
	}
	if event.Event != "notification.created" || event.Notification.Booking.Description != "Coordination" {
		t.Fatal(event)
	}
	// A subsequent marker must be the creator's first received message.
	hub.Broadcast(model.Notification{ID: "marker"}, "other")
	creator.SetReadDeadline(time.Now().Add(time.Second))
	if err := creator.ReadJSON(&event); err != nil {
		t.Fatal(err)
	}
	if event.Notification.ID != "marker" {
		t.Fatal("creator received own booking")
	}
}
