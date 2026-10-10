package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mhmddaud/transjakarta-fleet/internal/config"
	"github.com/mhmddaud/transjakarta-fleet/internal/database"
	"github.com/mhmddaud/transjakarta-fleet/internal/model"
	"github.com/mhmddaud/transjakarta-fleet/internal/repository"
)

func TestLocationRepository_PostgreSQL(t *testing.T) {
	// Load konfigurasi aplikasi.
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("gagal load konfigurasi: %v", err)
	}

	// Buat koneksi ke PostgreSQL.
	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	db, err := database.NewPostgresPool(ctx, cfg)
	if err != nil {
		t.Fatalf("gagal konek ke PostgreSQL: %v", err)
	}
	defer db.Close()

	repo := repository.NewLocationRepository(db)

	// Gunakan ID unik agar tidak bentrok dengan data lain.
	vehicleID := fmt.Sprintf("test-%d", time.Now().UnixNano())
	baseTimestamp := time.Now().Unix()

	// Hapus data test setelah test selesai.
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cleanupCancel()

		_, err := db.Exec(
			cleanupCtx,
			`DELETE FROM vehicle_locations WHERE vehicle_id = $1`,
			vehicleID,
		)
		if err != nil {
			t.Logf("peringatan: gagal membersihkan data test: %v", err)
		}
	}()

	// Data untuk pengujian.
	locations := []model.VehicleLocation{
		{
			VehicleID: vehicleID,
			Latitude:  -6.2000,
			Longitude: 106.8000,
			Timestamp: baseTimestamp - 120,
		},
		{
			VehicleID: vehicleID,
			Latitude:  -6.2100,
			Longitude: 106.8100,
			Timestamp: baseTimestamp - 60,
		},
		{
			VehicleID: vehicleID,
			Latitude:  -6.2200,
			Longitude: 106.8200,
			Timestamp: baseTimestamp,
		},
	}

	// 1. Test Insert.
	for _, location := range locations {
		if err := repo.Insert(ctx, &location); err != nil {
			t.Fatalf("Insert gagal: %v", err)
		}
	}
	t.Log("Insert: PASS - 3 lokasi berhasil disimpan")

	// 2. Test GetLatest.
	latest, err := repo.GetLatest(ctx, vehicleID)
	if err != nil {
		t.Fatalf("GetLatest gagal: %v", err)
	}
	if latest == nil {
		t.Fatal("GetLatest mengembalikan nil")
	}

	if latest.Timestamp != baseTimestamp {
		t.Errorf(
			"timestamp terbaru salah: got %d, want %d",
			latest.Timestamp,
			baseTimestamp,
		)
	}
	if latest.Latitude != -6.2200 || latest.Longitude != 106.8200 {
		t.Errorf(
			"koordinat terbaru salah: got (%f, %f)",
			latest.Latitude,
			latest.Longitude,
		)
	}
	t.Log("GetLatest: PASS - lokasi terbaru sesuai")

	// 3. Test GetHistory.
	history, err := repo.GetHistory(
		ctx,
		vehicleID,
		baseTimestamp-120,
		baseTimestamp-60,
	)
	if err != nil {
		t.Fatalf("GetHistory gagal: %v", err)
	}

	if len(history) != 2 {
		t.Fatalf(
			"jumlah riwayat salah: got %d, want 2",
			len(history),
		)
	}

	if history[0].Timestamp != baseTimestamp-120 {
		t.Errorf("timestamp riwayat pertama tidak sesuai")
	}
	if history[1].Timestamp != baseTimestamp-60 {
		t.Errorf("timestamp riwayat kedua tidak sesuai")
	}

	t.Log("GetHistory: PASS - 2 lokasi ditemukan dalam rentang waktu")
}
