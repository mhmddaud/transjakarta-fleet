package service

import (
	"math"
	"testing"
)

func TestGeofenceService_IsInside(t *testing.T) {
	// Titik pusat dan radius geofence 50 meter.
	const (
		centerLat = -6.2088
		centerLon = 106.8456
		radius    = 50.0
	)

	geofence := NewGeofenceService(centerLat, centerLon, radius)

	tests := []struct {
		name      string
		latitude  float64
		longitude float64
		want      bool
	}{
		{
			name:      "tepat di titik pusat",
			latitude:  centerLat,
			longitude: centerLon,
			want:      true,
		},
		{
			name:      "di dalam radius",
			latitude:  centerLat + 0.0001,
			longitude: centerLon,
			want:      true,
		},
		{
			name:      "di luar radius",
			latitude:  centerLat + 0.01,
			longitude: centerLon,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := geofence.IsInside(tt.latitude, tt.longitude)

			if got != tt.want {
				t.Errorf("IsInside() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGeofenceService_DistanceMeters(t *testing.T) {
	geofence := NewGeofenceService(-6.2088, 106.8456, 50)

	// Jarak titik terhadap dirinya sendiri harus nol.
	got := geofence.DistanceMeters(-6.2088, 106.8456)

	if math.Abs(got) > 0.001 {
		t.Errorf("DistanceMeters() = %f, want approximately 0", got)
	}

	// Perpindahan 0.001 derajat latitude sekitar 111 meter.
	got = geofence.DistanceMeters(-6.2078, 106.8456)

	if got < 100 || got > 120 {
		t.Errorf("DistanceMeters() = %f, expected between 100 and 120 meters", got)
	}
}
