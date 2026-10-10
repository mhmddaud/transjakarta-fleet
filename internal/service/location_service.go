package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mhmddaud/transjakarta-fleet/internal/model"
)

type LocationRepository interface {
	Insert(ctx context.Context, location *model.VehicleLocation) error
	GetLatest(ctx context.Context, vehicleID string) (*model.VehicleLocation, error)
	GetHistory(
		ctx context.Context,
		vehicleID string,
		start int64,
		end int64,
	) ([]model.VehicleLocation, error)
}

type GeofenceChecker interface {
	IsInside(latitude, longitude float64) bool
}

type EventPublisher interface {
	PublishEvent(ctx context.Context, event GeofenceEvent) error
}

type GeofenceEvent struct {
	VehicleID string  `json:"vehicle_id"`
	EventType string  `json:"event_type"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timestamp int64   `json:"timestamp"`
}

type LocationService struct {
	repository LocationRepository
	geofence   GeofenceChecker
	publisher  EventPublisher
}

func NewLocationService(
	repository LocationRepository,
	geofence GeofenceChecker,
	publishers ...EventPublisher,
) *LocationService {
	s := &LocationService{
		repository: repository,
		geofence:   geofence,
	}

	if len(publishers) > 0 {
		s.publisher = publishers[0]
	}

	return s
}

// ProcessLocation menyimpan lokasi kendaraan dan mendeteksi
// transisi masuk atau keluar dari geofence.
func (s *LocationService) ProcessLocation(
	ctx context.Context,
	location *model.VehicleLocation,
) (*GeofenceEvent, error) {
	if location == nil {
		return nil, errors.New("location cannot be nil")
	}

	location.VehicleID = strings.TrimSpace(location.VehicleID)

	if location.VehicleID == "" {
		return nil, errors.New("vehicle_id is required")
	}
	if location.Latitude < -90 || location.Latitude > 90 {
		return nil, errors.New("latitude must be between -90 and 90")
	}
	if location.Longitude < -180 || location.Longitude > 180 {
		return nil, errors.New("longitude must be between -180 and 180")
	}
	if location.Timestamp <= 0 {
		return nil, errors.New("timestamp must be positive")
	}

	previous, err := s.repository.GetLatest(ctx, location.VehicleID)
	if err != nil {
		return nil, fmt.Errorf("get previous location: %w", err)
	}

	// Validasi timestamp hanya jika ada lokasi sebelumnya.
	if previous != nil && location.Timestamp <= previous.Timestamp {
		return nil, fmt.Errorf("location timestamp must be newer than previous location")
	}

	// Simpan lokasi tepat satu kali.
	if err := s.repository.Insert(ctx, location); err != nil {
		return nil, fmt.Errorf("insert location: %w", err)
	}

	// Lokasi pertama tidak memiliki lokasi pembanding untuk event.
	if previous == nil {
		return nil, nil
	}

	wasInside := s.geofence.IsInside(
		previous.Latitude,
		previous.Longitude,
	)
	isInside := s.geofence.IsInside(
		location.Latitude,
		location.Longitude,
	)

	if !wasInside && isInside {
		event := &GeofenceEvent{
			VehicleID: location.VehicleID,
			EventType: "vehicle.entered_geofence",
			Latitude:  location.Latitude,
			Longitude: location.Longitude,
			Timestamp: location.Timestamp,
		}

		if s.publisher != nil {
			if err := s.publisher.PublishEvent(ctx, *event); err != nil {
				return event, fmt.Errorf("publish geofence event: %w", err)
			}
		}

		return event, nil
	}

	if wasInside && !isInside {
		event := &GeofenceEvent{
			VehicleID: location.VehicleID,
			EventType: "vehicle.exited_geofence",
			Latitude:  location.Latitude,
			Longitude: location.Longitude,
			Timestamp: location.Timestamp,
		}

		if s.publisher != nil {
			if err := s.publisher.PublishEvent(ctx, *event); err != nil {
				return event, fmt.Errorf("publish geofence event: %w", err)
			}
		}

		return event, nil
	}

	return nil, nil
}

// HandleMQTTLocation memproses lokasi yang diterima dari MQTT.
func (s *LocationService) HandleMQTTLocation(
	ctx context.Context,
	location *model.VehicleLocation,
) error {
	_, err := s.ProcessLocation(ctx, location)
	return err
}

// GetLatestLocation mengambil lokasi terakhir kendaraan.
func (s *LocationService) GetLatestLocation(
	ctx context.Context,
	vehicleID string,
) (*model.VehicleLocation, error) {
	vehicleID = strings.TrimSpace(vehicleID)
	if vehicleID == "" {
		return nil, errors.New("vehicle_id is required")
	}

	return s.repository.GetLatest(ctx, vehicleID)
}

// GetLocationHistory mengambil riwayat lokasi kendaraan.
func (s *LocationService) GetLocationHistory(
	ctx context.Context,
	vehicleID string,
	start int64,
	end int64,
) ([]model.VehicleLocation, error) {
	vehicleID = strings.TrimSpace(vehicleID)
	if vehicleID == "" {
		return nil, errors.New("vehicle_id is required")
	}

	if start <= 0 || end <= 0 {
		return nil, errors.New("start and end must be positive Unix timestamps")
	}

	if start > end {
		return nil, errors.New("start must be less than or equal to end")
	}

	return s.repository.GetHistory(ctx, vehicleID, start, end)
}
