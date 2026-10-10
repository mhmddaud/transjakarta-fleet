package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/mhmddaud/transjakarta-fleet/internal/model"
)

type Processor interface {
	HandleMQTTLocation(context.Context, *model.VehicleLocation) error
}

type Subscriber struct {
	client paho.Client
	topic  string
	proc   Processor
}

func NewSubscriber(
	broker string,
	clientID string,
	topic string,
	proc Processor,
) (*Subscriber, error) {
	if strings.TrimSpace(broker) == "" {
		return nil, fmt.Errorf("MQTT broker is required")
	}

	s := &Subscriber{
		topic: topic,
		proc:  proc,
	}

	opts := paho.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(2 * time.Second)

	opts.SetOnConnectHandler(func(client paho.Client) {
		token := client.Subscribe(topic, 1, s.onMessage)
		if !token.WaitTimeout(10 * time.Second) {
			log.Printf("MQTT subscribe timed out: %s", topic)
			return
		}
		if token.Error() != nil {
			log.Printf("MQTT subscribe failed: %v", token.Error())
			return
		}
		log.Printf("MQTT subscribed to %s", topic)
	})

	s.client = paho.NewClient(opts)
	return s, nil
}

func (s *Subscriber) Connect(ctx context.Context) error {
	token := s.client.Connect()

	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("MQTT connection timed out")
	}
	if token.Error() != nil {
		return fmt.Errorf("connect MQTT: %w", token.Error())
	}

	return nil
}

func (s *Subscriber) Close() {
	if s.client != nil && s.client.IsConnected() {
		s.client.Disconnect(250)
	}
}

func (s *Subscriber) onMessage(
	_ paho.Client,
	msg paho.Message,
) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	var location model.VehicleLocation
	if err := json.Unmarshal(msg.Payload(), &location); err != nil {
		log.Printf("invalid MQTT JSON topic=%s: %v", msg.Topic(), err)
		return
	}

	// Topic format: /fleet/vehicle/{vehicle_id}/location
	parts := strings.Split(msg.Topic(), "/")
	if len(parts) != 5 ||
		parts[0] != "" ||
		parts[1] != "fleet" ||
		parts[2] != "vehicle" ||
		parts[3] == "" ||
		parts[4] != "location" {
		log.Printf("invalid MQTT topic: %s", msg.Topic())
		return
	}

	topicVehicleID := parts[3]

	if location.VehicleID == "" {
		location.VehicleID = topicVehicleID
	} else if location.VehicleID != topicVehicleID {
		log.Printf("vehicle_id does not match topic: %s", msg.Topic())
		return
	}

	if err := s.proc.HandleMQTTLocation(ctx, &location); err != nil {
		log.Printf("process MQTT location failed: %v", err)
		return
	}

	log.Printf(
		"MQTT location processed vehicle=%s topic=%s",
		location.VehicleID,
		msg.Topic(),
	)
}
