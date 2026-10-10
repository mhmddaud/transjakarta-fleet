package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mhmddaud/transjakarta-fleet/internal/model"
)

type LocationRepository struct {
	db *pgxpool.Pool
}

func NewLocationRepository(db *pgxpool.Pool) *LocationRepository {
	return &LocationRepository{
		db: db,
	}
}

func (r *LocationRepository) Insert(
	ctx context.Context,
	location *model.VehicleLocation,
) error {
	query := `
		INSERT INTO vehicle_locations (
			vehicle_id,
			latitude,
			longitude,
			timestamp
		)
		VALUES ($1, $2, $3, $4)
	`

	_, err := r.db.Exec(
		ctx,
		query,
		location.VehicleID,
		location.Latitude,
		location.Longitude,
		location.Timestamp,
	)

	return err
}

func (r *LocationRepository) GetLatest(
	ctx context.Context,
	vehicleID string,
) (*model.VehicleLocation, error) {
	query := `
		SELECT
			vehicle_id,
			latitude,
			longitude,
			timestamp
		FROM vehicle_locations
		WHERE vehicle_id = $1
		ORDER BY timestamp DESC
		LIMIT 1
	`

	var location model.VehicleLocation

	err := r.db.QueryRow(
		ctx,
		query,
		vehicleID,
	).Scan(
		&location.VehicleID,
		&location.Latitude,
		&location.Longitude,
		&location.Timestamp,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &location, nil
}

func (r *LocationRepository) GetHistory(
	ctx context.Context,
	vehicleID string,
	start int64,
	end int64,
) ([]model.VehicleLocation, error) {
	query := `
		SELECT
			vehicle_id,
			latitude,
			longitude,
			timestamp
		FROM vehicle_locations
		WHERE vehicle_id = $1
		  AND timestamp BETWEEN $2 AND $3
		ORDER BY timestamp ASC
	`

	rows, err := r.db.Query(
		ctx,
		query,
		vehicleID,
		start,
		end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	locations := make([]model.VehicleLocation, 0)

	for rows.Next() {
		var location model.VehicleLocation

		err := rows.Scan(
			&location.VehicleID,
			&location.Latitude,
			&location.Longitude,
			&location.Timestamp,
		)
		if err != nil {
			return nil, err
		}

		locations = append(locations, location)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return locations, nil
}
