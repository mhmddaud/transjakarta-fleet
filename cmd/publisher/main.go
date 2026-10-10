package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/mhmddaud/transjakarta-fleet/internal/model"
)

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	broker := env("MQTT_BROKER", "tcp://localhost:1883")
	vehicleID := env("VEHICLE_ID", "B1234XYZ")

	interval := 2 * time.Second
	if value := os.Getenv("PUBLISH_INTERVAL_SECONDS"); value != "" {
		seconds, err := strconv.Atoi(value)
		if err != nil || seconds <= 0 {
			log.Fatal("PUBLISH_INTERVAL_SECONDS must be a positive integer")
		}
		interval = time.Duration(seconds) * time.Second
	}

	opts := paho.NewClientOptions().
		AddBroker(broker).
		SetClientID("fleet-mock-publisher-" + vehicleID)

	client := paho.NewClient(opts)
	token := client.Connect()

	if !token.WaitTimeout(10 * time.Second) {
		log.Fatal("MQTT connection timed out")
	}
	if token.Error() != nil {
		log.Fatalf("connect MQTT: %v", token.Error())
	}
	defer client.Disconnect(250)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var count int64

	publish := func() {
		// Bergantian masuk dan keluar dari geofence.
		latitude, longitude := -6.2088, 106.8456

		if count%2 == 1 {
			latitude, longitude = -6.3000, 106.9000
		}

		location := model.VehicleLocation{
			VehicleID: vehicleID,
			Latitude:  latitude,
			Longitude: longitude,
			Timestamp: time.Now().Unix(),
		}

		payload, err := json.Marshal(location)
		if err != nil {
			log.Printf("encode location: %v", err)
			return
		}

		topic := "/fleet/vehicle/" + vehicleID + "/location"
		token := client.Publish(topic, 1, false, payload)

		if !token.WaitTimeout(5 * time.Second) {
			log.Printf("MQTT publish timed out")
			return
		}
		if token.Error() != nil {
			log.Printf("MQTT publish failed: %v", token.Error())
			return
		}

		log.Printf(
			"published vehicle=%s lat=%.6f lon=%.6f",
			vehicleID, latitude, longitude,
		)
		count++
	}

	publish()

	for {
		select {
		case <-stop:
			log.Println("MQTT publisher stopped")
			return
		case <-ticker.C:
			publish()
		}
	}
}
