package paho

// LoraSubscribe subscribe to lora server messages
import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/lora"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MqttBroker represents the MQTT broker.
type MqttBroker interface {
	// Subscribes to geven subject and receives events.
	Subscribe(string) error
}

type broker struct {
	svc    lora.Service
	client mqtt.Client
	logger logger.Logger
}

// NewBroker returns new MQTT broker instance.
func NewBroker(svc lora.Service, client mqtt.Client, log logger.Logger) MqttBroker {
	return broker{
		svc:    svc,
		client: client,
		logger: log,
	}
}

// Subscribe subscribes to the Lora MQTT message broker
func (b broker) Subscribe(subject string) error {
	s := b.client.Subscribe(subject, 0, b.handleMsg)
	if err := s.Error(); s.Wait() && err != nil {
		return err
	}

	return nil
}

// handleMsg triggered when new message is received on Lora MQTT broker
func (b broker) handleMsg(c mqtt.Client, msg mqtt.Message) {
	m := lora.Message{}
	if err := json.Unmarshal(msg.Payload(), &m); err != nil {
		b.logger.Warn(fmt.Sprintf("Failed to Unmarshal message: %s", err.Error()))
		return
	}

	b.svc.Publish(context.Background(), "", m)
	return
}
