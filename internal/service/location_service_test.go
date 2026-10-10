package service

import (
	"context"
	"errors"
	"testing"

	"github.com/mhmddaud/transjakarta-fleet/internal/model"
)

type fakeLocationRepository struct {
	latest *model.VehicleLocation
	items  []model.VehicleLocation
	err    error
}

func (f *fakeLocationRepository) GetLatest(
	ctx context.Context,
	vehicleID string,
) (*model.VehicleLocation, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.latest, nil
}

func (f *fakeLocationRepository) Insert(
	ctx context.Context,
	location *model.VehicleLocation,
) error {
	if f.err != nil {
		return f.err
	}

	copyLocation := *location
	f.items = append(f.items, copyLocation)
	f.latest = &copyLocation
	return nil
}

func TestLocationService_ProcessLocation(t *testing.T) {
	const (
		centerLat = -6.2088
		centerLon = 106.8456
	)

	tests := []struct {
		name          string
		previous      *model.VehicleLocation
		current       model.VehicleLocation
		wantEventType string
	}{
		{
			name: "lokasi pertama tidak menghasilkan event",
			current: model.VehicleLocation{
				VehicleID: "BUS-001",
				Latitude:  centerLat,
				Longitude: centerLon,
				Timestamp: 100,
			},
		},
		{
			name: "kendaraan masuk geofence",
			previous: &model.VehicleLocation{
				VehicleID: "BUS-002",
				Latitude:  centerLat + 0.01,
				Longitude: centerLon,
				Timestamp: 100,
			},
			current: model.VehicleLocation{
				VehicleID: "BUS-002",
				Latitude:  centerLat,
				Longitude: centerLon,
				Timestamp: 101,
			},
			wantEventType: "vehicle.entered_geofence",
		},
		{
			name: "kendaraan keluar geofence",
			previous: &model.VehicleLocation{
				VehicleID: "BUS-003",
				Latitude:  centerLat,
				Longitude: centerLon,
				Timestamp: 100,
			},
			current: model.VehicleLocation{
				VehicleID: "BUS-003",
				Latitude:  centerLat + 0.01,
				Longitude: centerLon,
				Timestamp: 101,
			},
			wantEventType: "vehicle.exited_geofence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeLocationRepository{
				latest: tt.previous,
			}

			geofence := NewGeofenceService(centerLat, centerLon, 50)
			svc := NewLocationService(repo, geofence)

			event, err := svc.ProcessLocation(
				context.Background(),
				&tt.current,
			)
			if err != nil {
				t.Fatalf("ProcessLocation error: %v", err)
			}

			if tt.wantEventType == "" {
				if event != nil {
					t.Fatalf("expected no event, got %s", event.EventType)
				}
			} else {
				if event == nil {
					t.Fatalf("expected event %s, got nil", tt.wantEventType)
				}
				if event.EventType != tt.wantEventType {
					t.Errorf(
						"event type = %s, want %s",
						event.EventType,
						tt.wantEventType,
					)
				}
			}

			if len(repo.items) != 1 {
				t.Errorf("inserted locations = %d, want 1", len(repo.items))
			}
		})
	}
}

func TestLocationService_RejectsInvalidLocation(t *testing.T) {
	repo := &fakeLocationRepository{}
	geofence := NewGeofenceService(-6.2088, 106.8456, 50)
	svc := NewLocationService(repo, geofence)

	location := &model.VehicleLocation{
		VehicleID: "BUS-INVALID",
		Latitude:  100,
		Longitude: 106.8456,
		Timestamp: 100,
	}

	_, err := svc.ProcessLocation(context.Background(), location)
	if err == nil {
		t.Fatal("expected validation error")
	}

	if len(repo.items) != 0 {
		t.Fatal("invalid location must not be inserted")
	}
}

func TestLocationService_RepositoryError(t *testing.T) {
	repo := &fakeLocationRepository{
		err: errors.New("database unavailable"),
	}
	geofence := NewGeofenceService(-6.2088, 106.8456, 50)
	svc := NewLocationService(repo, geofence)

	location := &model.VehicleLocation{
		VehicleID: "BUS-ERROR",
		Latitude:  -6.2088,
		Longitude: 106.8456,
		Timestamp: 100,
	}

	_, err := svc.ProcessLocation(context.Background(), location)
	if err == nil {
		t.Fatal("expected repository error")
	}
}
