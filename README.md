# Transjakarta Fleet Tracking

Backend service untuk menerima lokasi kendaraan, menyimpan riwayat lokasi ke PostgreSQL, mendeteksi perpindahan kendaraan masuk atau keluar dari area geofence, serta mengirimkan event melalui RabbitMQ untuk diproses secara asynchronous.

## Features

- REST API untuk menerima lokasi kendaraan.
- Penyimpanan riwayat lokasi kendaraan menggunakan PostgreSQL.
- Deteksi event `vehicle.entered_geofence` dan `vehicle.exited_geofence`.
- Asynchronous event publishing dan consumption menggunakan RabbitMQ.
- Retry pemrosesan pesan dengan jeda 5 detik.
- Dead Letter Queue (DLQ) untuk pesan yang gagal diproses setelah batas percobaan.
- Docker Compose untuk menjalankan service dan dependensinya.
- Unit test dan integration test untuk service, repository, dan RabbitMQ.

## Technology Stack

- Go
- PostgreSQL
- RabbitMQ
- Docker dan Docker Compose
- MQTT integration components

## Architecture

1. Client mengirimkan data lokasi kendaraan melalui REST API.
2. Server memvalidasi request dan memproses lokasi.
3. Data lokasi disimpan ke PostgreSQL.
4. Service geofence menentukan apakah kendaraan masuk atau keluar dari area geofence.
5. Event transisi dipublikasikan ke RabbitMQ.
6. Consumer menerima event dan memproses notifikasi.
7. Jika pemrosesan gagal, pesan dijadwalkan untuk retry. Pesan yang tetap gagal akan diarahkan ke DLQ.

## Prerequisites

- Go sesuai versi pada `go.mod`.
- Docker Desktop dengan Docker Compose.
- Git.

## Configuration

Buat file `.env` berdasarkan `.env.example`:

```powershell
Copy-Item .env.example .env
```

Periksa dan sesuaikan nilai konfigurasi di `.env`, khususnya PostgreSQL, RabbitMQ, port aplikasi, dan geofence.

Jangan commit file `.env` atau menyimpan kredensial asli ke repository.

## Run with Docker Compose

Build dan jalankan seluruh service:

```powershell
docker compose up -d --build
```

Periksa status container:

```powershell
docker compose ps
```

Lihat log server dan consumer:

```powershell
docker compose logs -f server
docker compose logs -f consumer
```

### Database Migration

Setelah PostgreSQL berjalan, jalankan migrasi awal:

```powershell
Get-Content .\migrations\001_create_vehicle_locations.sql |
  docker compose exec -T postgres psql -U postgres -d fleet
```

Perintah tersebut menggunakan nama database dan user default dari Docker Compose. Jika konfigurasi diubah, sesuaikan parameter `psql`.

### Stop Services

```powershell
docker compose down
```

Perintah di atas mempertahankan named volume database. Untuk menghapus volume dan seluruh data PostgreSQL secara permanen, gunakan `docker compose down -v` hanya jika memang ingin menghapus data.

## API

### Health Check

```http
GET /health
```

URL lokal:

```text
http://localhost:8080/health
```

### Submit Vehicle Location

```http
POST /api/v1/locations
Content-Type: application/json
```

Request body:

```json
{
  "vehicle_id": "TJ001",
  "latitude": -6.2088,
  "longitude": 106.8456,
  "timestamp": 1791618914
}
```

`timestamp` menggunakan Unix timestamp dalam detik.

Contoh respons saat event geofence terdeteksi:

```json
{
  "event": {
    "vehicle_id": "TJ001",
    "event_type": "vehicle.entered_geofence",
    "latitude": -6.2088,
    "longitude": 106.8456,
    "timestamp": 1791618914
  },
  "message": "location processed"
}
```

Lokasi pertama kendaraan belum tentu menghasilkan event karena belum ada lokasi sebelumnya untuk dibandingkan.

## RabbitMQ

Konfigurasi default:

- Main exchange: `fleet.events`
- Main queue: `geofence_alerts`
- Retry exchange: `fleet.events.retry`
- Retry queue: `geofence_alerts.retry`
- Dead-letter exchange: `fleet.events.dlx`
- Dead-letter queue: `geofence_alerts.dlq`
- Retry delay: 5 detik
- Maximum retry attempts: 3 kali setelah pemrosesan awal

RabbitMQ Management UI tersedia di:

```text
http://localhost:15672
```

Gunakan kredensial yang dikonfigurasi pada environment Docker Compose.

## Testing and Validation

Jalankan seluruh test:

```powershell
go test ./... -count=1
```

Jalankan integration test RabbitMQ:

```powershell
$env:RABBITMQ_URL = "amqp://fleet:fleet_dev_password@localhost:5672/"
go test -v ./internal/rabbitmq -run '^TestConsumerRejectsFailedEventToDLQ$' -count=1
```

Sesuaikan URL dengan kredensial RabbitMQ yang digunakan.

Static analysis:

```powershell
go vet ./...
```

Build:

```powershell
go build ./...
```

Integration test memerlukan dependensi yang sesuai dan konfigurasi environment yang valid.

## Project Structure

```text
.
├── cmd/
│   ├── consumer/
│   ├── publisher/
│   └── server/
├── internal/
│   ├── config/
│   ├── database/
│   ├── handler/
│   ├── model/
│   ├── mqtt/
│   ├── rabbitmq/
│   ├── repository/
│   └── service/
├── migrations/
├── mosquitto/
├── Dockerfile
├── docker-compose.yml
├── go.mod
└── README.md
```

## Notes

- Pastikan database dan RabbitMQ tersedia sebelum menjalankan service di luar Docker.
- Untuk koneksi antar-container, gunakan nama service Compose seperti `postgres` dan `rabbitmq`, bukan `localhost`.
- Gunakan `.env.example` sebagai template konfigurasi tanpa kredensial rahasia.
- Sesuaikan pengaturan geofence melalui konfigurasi aplikasi yang tersedia.