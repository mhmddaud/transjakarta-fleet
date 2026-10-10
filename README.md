# Transjakarta Fleet Management Backend

Aplikasi backend untuk memantau lokasi kendaraan Transjakarta secara real-time, menyimpan riwayat lokasi, serta mendeteksi kendaraan yang memasuki atau meninggalkan area geofence.

Aplikasi dibangun menggunakan Go dengan integrasi MQTT, PostgreSQL, RabbitMQ, dan Docker Compose.

## 1. Fitur Utama

- Menerima data lokasi kendaraan secara real-time melalui MQTT.
- Menyimpan data lokasi kendaraan ke PostgreSQL.
- Mengambil lokasi terakhir kendaraan melalui REST API.
- Mengambil riwayat lokasi kendaraan berdasarkan rentang waktu.
- Mendeteksi kendaraan yang memasuki dan meninggalkan area geofence.
- Mengirimkan event geofence ke RabbitMQ.
- Memproses event geofence melalui consumer.
- Menyediakan mock publisher untuk simulasi pengiriman lokasi kendaraan.
- Menjalankan layanan aplikasi menggunakan Docker Compose.

## 2. Teknologi yang Digunakan

| Teknologi | Kegunaan |
|---|---|
| Go | Bahasa pemrograman backend |
| PostgreSQL | Penyimpanan data lokasi kendaraan |
| MQTT | Komunikasi data lokasi kendaraan |
| Eclipse Mosquitto | MQTT broker |
| RabbitMQ | Pengiriman dan distribusi event geofence |
| Docker | Menjalankan aplikasi dalam container |
| Docker Compose | Mengelola beberapa layanan aplikasi |
| REST API | Akses lokasi terakhir dan riwayat lokasi kendaraan |

## 3. Arsitektur Sistem

```text
             Mock Publisher
                    |
                    | MQTT
                    v
                Mosquitto
                    |
                    v
                Go Backend
                 /       \
                v         v
           PostgreSQL   RabbitMQ
           Data Lokasi  fleet.events
                           |
                           v
                        Consumer
                           |
                           v
                     Event Geofence
```

Alur proses aplikasi:

1. Mock publisher mengirimkan data lokasi kendaraan melalui MQTT.
2. Mosquitto menerima pesan MQTT dan meneruskannya kepada backend yang berlangganan pada topik lokasi kendaraan.
3. Backend memvalidasi dan memproses pesan, kemudian menyimpan data lokasi ke PostgreSQL.
4. Backend memeriksa apakah lokasi kendaraan menyebabkan perubahan status geofence.
5. Jika terjadi kendaraan masuk atau keluar dari area geofence, backend mengirimkan event ke RabbitMQ.
6. Consumer mengambil dan memproses event dari antrean `geofence_alerts`.
7. REST API menyediakan akses ke lokasi terakhir dan riwayat lokasi kendaraan.

## 4. Persyaratan Menjalankan Aplikasi

Pastikan perangkat telah memiliki:

- Docker Desktop.
- Docker Compose yang tersedia melalui Docker Desktop.
- Go, apabila ingin menjalankan pengujian atau build langsung dari komputer.

## 5. Struktur Project

Struktur direktori utama:

```text
transjakarta-fleet/
├── cmd/
│   ├── server/
│   │   └── main.go
│   ├── consumer/
│   └── publisher/
├── internal/
│   ├── config/
│   ├── handler/
│   ├── model/
│   ├── repository/
│   └── service/
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
├── .env
└── README.md
```

Struktur di atas menggambarkan komponen utama project. Sesuaikan dengan direktori aktual apabila terdapat file atau folder tambahan.

## 6. Konfigurasi Aplikasi

Konfigurasi aplikasi menggunakan environment variable yang didefinisikan dalam project dan Docker Compose.

Pastikan konfigurasi koneksi berikut sesuai dengan layanan yang berjalan:

- PostgreSQL untuk penyimpanan data.
- MQTT broker untuk menerima lokasi kendaraan.
- RabbitMQ untuk pengiriman event geofence.
- HTTP server untuk REST API.

Jika project menggunakan file `.env`, buat file tersebut berdasarkan konfigurasi yang diperlukan oleh aplikasi.

**Catatan keamanan:** Jangan memasukkan password, credential, atau informasi rahasia ke repository Git.

## 7. Menjalankan Aplikasi

### 7.1 Masuk ke direktori project

Buka PowerShell:

```powershell
cd D:\Project\transjakarta-fleet
```

### 7.2 Build dan jalankan seluruh layanan

```powershell
docker compose up -d --build
```

Perintah tersebut membangun image dan menjalankan layanan yang didefinisikan dalam Docker Compose.

### 7.3 Periksa status layanan

```powershell
docker compose ps
```

Pastikan layanan utama berjalan dengan baik, termasuk backend, PostgreSQL, Mosquitto, RabbitMQ, dan consumer.

### 7.4 Periksa log backend

```powershell
docker compose logs --tail 100 server
```

### 7.5 Periksa log consumer

```powershell
docker compose logs --tail 100 consumer
```

### 7.6 Jalankan mock publisher

Buka terminal PowerShell terpisah dan jalankan:

```powershell
docker compose --profile tools run --rm publisher
```

Publisher akan mengirimkan simulasi lokasi kendaraan melalui MQTT secara berkala.

Tekan `Ctrl+C` untuk menghentikan proses publisher.

## 8. Integrasi MQTT

### 8.1 Topik MQTT

Format topik yang digunakan:

```text
/fleet/vehicle/{vehicle_id}/location
```

Contoh:

```text
/fleet/vehicle/B1234XYZ/location
```

### 8.2 Format pesan MQTT

Contoh payload JSON:

```json
{
  "vehicle_id": "B1234XYZ",
  "latitude": -6.2088,
  "longitude": 106.8456,
  "timestamp": 1791651686
}
```

Keterangan:

| Field | Deskripsi |
|---|---|
| `vehicle_id` | Identitas kendaraan |
| `latitude` | Koordinat lintang kendaraan |
| `longitude` | Koordinat bujur kendaraan |
| `timestamp` | Waktu pengambilan lokasi dalam Unix timestamp, satuan detik |

## 9. REST API

Base URL:

```text
http://localhost:8080
```

### 9.1 Mendapatkan Lokasi Terakhir Kendaraan

**Endpoint**

```http
GET /vehicles/{vehicle_id}/location
```

**Contoh request**

```http
GET http://localhost:8080/vehicles/B1234XYZ/location
```

**Contoh respons**

```json
{
  "vehicle_id": "B1234XYZ",
  "latitude": -6.2088,
  "longitude": 106.8456,
  "timestamp": 1791651686
}
```

Endpoint ini digunakan untuk mengambil lokasi terakhir yang tersimpan untuk kendaraan dengan ID yang diminta.

### 9.2 Mendapatkan Riwayat Lokasi Kendaraan

**Endpoint**

```http
GET /vehicles/{vehicle_id}/history?start={timestamp_awal}&end={timestamp_akhir}
```

**Contoh request**

```http
GET http://localhost:8080/vehicles/B1234XYZ/history?start=1700000000&end=1900000000
```

Parameter:

| Parameter | Deskripsi |
|---|---|
| `vehicle_id` | Identitas kendaraan |
| `start` | Awal rentang waktu dalam Unix timestamp |
| `end` | Akhir rentang waktu dalam Unix timestamp |

Respons berisi kumpulan data lokasi kendaraan dalam rentang waktu yang diminta, diurutkan berdasarkan waktu secara menaik.

Pastikan nilai `start` dan `end` menggunakan Unix timestamp dalam satuan detik.

## 10. Deteksi Geofence

Aplikasi menggunakan area geofence berbentuk lingkaran dengan konfigurasi berikut:

| Parameter | Nilai |
|---|---|
| Latitude pusat | `-6.2088` |
| Longitude pusat | `106.8456` |
| Radius | `50 meter` |

Backend memeriksa lokasi kendaraan terhadap area geofence dan menghasilkan event ketika status kendaraan berubah.

Jenis event yang digunakan:

```text
vehicle.entered_geofence
vehicle.exited_geofence
```

### Integrasi RabbitMQ

Nama exchange:

```text
fleet.events
```

Nama queue:

```text
geofence_alerts
```

Alur event:

1. Backend mendeteksi perubahan status geofence.
2. Backend menerbitkan event ke exchange `fleet.events`.
3. RabbitMQ meneruskan event ke queue `geofence_alerts` sesuai konfigurasi binding.
4. Consumer menerima dan memproses event tersebut.

Untuk melihat event yang diproses:

```powershell
docker compose logs --tail 100 consumer
```

## 11. Pengujian Aplikasi

Jalankan perintah berikut dari direktori project.

### 11.1 Menjalankan unit test

```powershell
go test ./... -count=1
```

### 11.2 Menjalankan pemeriksaan kode

```powershell
go vet ./...
```

### 11.3 Build seluruh package Go

```powershell
go build ./...
```

Ketiga perintah tersebut digunakan untuk memeriksa pengujian, potensi masalah kode, dan keberhasilan proses build.

## 12. Troubleshooting

### 12.1 Memeriksa status container

```powershell
docker compose ps
```

### 12.2 Memeriksa log backend

```powershell
docker compose logs --tail 100 server
```

### 12.3 Memeriksa log consumer

```powershell
docker compose logs --tail 100 consumer
```

### 12.4 Memeriksa log seluruh layanan

```powershell
docker compose logs --tail 100
```

### 12.5 Menghentikan aplikasi

```powershell
docker compose down
```

Perintah ini menghentikan dan menghapus container serta jaringan yang dibuat oleh Compose. Volume persisten tidak dihapus kecuali menggunakan opsi penghapusan volume secara eksplisit.

## 13. Catatan Pengembangan

- Mock publisher digunakan untuk simulasi pengiriman lokasi kendaraan.
- Data lokasi disimpan di PostgreSQL.
- Komunikasi lokasi kendaraan menggunakan MQTT.
- RabbitMQ digunakan untuk distribusi event geofence.
- Consumer memproses event yang diterima dari queue.
- Konfigurasi koneksi dan geofence perlu disesuaikan dengan kebutuhan lingkungan pengujian.