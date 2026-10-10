package service

import "math"

const earthRadiusMeters = 6371000

type GeofenceService struct {
	centerLatitude  float64
	centerLongitude float64
	radiusMeters    float64
}

func NewGeofenceService(
	centerLatitude float64,
	centerLongitude float64,
	radiusMeters float64,
) *GeofenceService {
	return &GeofenceService{
		centerLatitude:  centerLatitude,
		centerLongitude: centerLongitude,
		radiusMeters:    radiusMeters,
	}
}

// DistanceMeters menghitung jarak antara dua koordinat menggunakan
// rumus Haversine, dengan hasil dalam meter.
func (s *GeofenceService) DistanceMeters(
	latitude float64,
	longitude float64,
) float64 {
	lat1 := s.centerLatitude * math.Pi / 180
	lat2 := latitude * math.Pi / 180

	deltaLat := (latitude - s.centerLatitude) * math.Pi / 180
	deltaLon := (longitude - s.centerLongitude) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)

	// Menghindari nilai di luar rentang akibat pembulatan floating point.
	a = math.Max(0, math.Min(1, a))

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusMeters * c
}

// IsInside mengembalikan true jika koordinat berada di dalam
// atau tepat pada batas radius geofence.
func (s *GeofenceService) IsInside(
	latitude float64,
	longitude float64,
) bool {
	if s.radiusMeters < 0 {
		return false
	}

	return s.DistanceMeters(latitude, longitude) <= s.radiusMeters
}
