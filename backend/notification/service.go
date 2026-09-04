package notification

import (
	"context"

	"my-ple-backend/db"
	"my-ple-backend/model"
	"my-ple-backend/realtime"
)

type Service struct {
	DB  *db.Database
	Hub *realtime.Hub
}

func (s *Service) CreateZoomBooking(ctx context.Context, userID string, input model.ZoomBookingInput) (model.ZoomBooking, error) {
	booking, event, err := s.DB.CreateZoomBookingWithNotification(ctx, userID, input)
	if err != nil {
		return model.ZoomBooking{}, err
	}
	if s.Hub != nil {
		s.Hub.Broadcast(event, userID)
	}
	return booking, nil
}
