package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort string

	PostgresHost     string
	PostgresPort     string
	PostgresDatabase string
	PostgresUser     string
	PostgresPassword string

	MQTTBroker   string
	MQTTClientID string

	RabbitMQURL      string
	RabbitMQExchange string
	RabbitMQQueue    string

	GeofenceLatitude  float64
	GeofenceLongitude float64
	GeofenceRadius    float64
}

func Load() (*Config, error) {
	// Ignore error because .env may not exist
	// when running inside Docker.
	_ = godotenv.Load()

	config := &Config{
		AppPort:          getEnv("APP_PORT", "8080"),
		PostgresHost:     getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:     getEnv("POSTGRES_PORT", "5432"),
		PostgresDatabase: getEnv("POSTGRES_DB", "fleet"),
		PostgresUser:     getEnv("POSTGRES_USER", "postgres"),
		PostgresPassword: getEnv("POSTGRES_PASSWORD", "postgres"),

		MQTTBroker:   getEnv("MQTT_BROKER", "tcp://localhost:1883"),
		MQTTClientID: getEnv("MQTT_CLIENT_ID", "fleet-backend"),

		RabbitMQURL:      getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		RabbitMQExchange: getEnv("RABBITMQ_EXCHANGE", "fleet.events"),
		RabbitMQQueue:    getEnv("RABBITMQ_QUEUE", "geofence_alerts"),

		GeofenceLatitude:  -6.2088,
		GeofenceLongitude: 106.8456,
		GeofenceRadius:    50,
	}

	return config, nil
}

func getEnv(key string, defaultValue string) string {
	value := os.Getenv(key)

	if value == "" {
		return defaultValue
	}

	return value
}

func (c *Config) PostgresDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		c.PostgresUser,
		c.PostgresPassword,
		c.PostgresHost,
		c.PostgresPort,
		c.PostgresDatabase,
	)
}
