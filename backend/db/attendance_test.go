package db

import (
	"errors"
	"math"
	"my-ple-backend/model"
	"testing"
)

func TestAttendanceLocation(t *testing.T) {
	office := AttendanceOffice{Latitude: 0, Longitude: 0, Configured: true}
	for _, tc := range []struct {
		name, mode       string
		meters, accuracy float64
		want             error
	}{
		{"inside", "WFO", 99, 10, nil}, {"boundary", "WFO", 100, 100, nil},
		{"outside", "WFO", 100.01, 10, ErrAttendanceOutside},
		{"inaccurate", "WFO", 0, 101, ErrAttendanceAccuracy},
		{"remote", "WFA", 10000, 1000, nil},
		{"invalid mode", "WFH", 0, 10, ErrAttendanceInvalid},
		{"negative accuracy", "WFA", 0, -1, ErrAttendanceInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lat, lng := tc.meters/6371000*180/math.Pi, 0.0
			in := model.AttendanceInput{Mode: tc.mode, Latitude: &lat, Longitude: &lng, Accuracy: &tc.accuracy}
			if err := office.Validate(in); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
	zero, accuracy := 0.0, 5.0
	in := model.AttendanceInput{Mode: "WFO", Latitude: &zero, Longitude: &zero, Accuracy: &accuracy}
	if err := (AttendanceOffice{}).Validate(in); !errors.Is(err, ErrAttendanceOffice) {
		t.Fatal(err)
	}
	in.Mode = "WFA"
	if err := (AttendanceOffice{}).Validate(in); err != nil {
		t.Fatal(err)
	}
	for _, value := range []float64{91, math.NaN(), math.Inf(1)} {
		in.Latitude = &value
		if !errors.Is(office.Validate(in), ErrAttendanceInvalid) {
			t.Fatal("invalid latitude accepted")
		}
	}
	in.Latitude = nil
	if !errors.Is(office.Validate(in), ErrAttendanceInvalid) {
		t.Fatal("missing latitude accepted")
	}
	// Verify the configured office pin with a non-zero latitude and longitude.
	office = AttendanceOffice{Latitude: -6.896211064380826, Longitude: 107.69234020127173, Configured: true}
	for _, meters := range []float64{0, 99, 101} {
		lat, lng := office.Latitude+meters/6371000*180/math.Pi, office.Longitude
		in = model.AttendanceInput{Mode: "WFO", Latitude: &lat, Longitude: &lng, Accuracy: &accuracy}
		err := office.Validate(in)
		if (meters <= 100 && err != nil) || (meters > 100 && !errors.Is(err, ErrAttendanceOutside)) {
			t.Fatalf("office pin at %v meters: %v", meters, err)
		}
	}
}
