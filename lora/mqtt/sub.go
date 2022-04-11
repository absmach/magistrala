package mqtt

// LoraSubscribe subscribe to lora server messages
import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/lora"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Subscriber represents the MQTT broker.
type Subscriber interface {
	// Subscribes to given subject and receives events.
	Subscribe(string) error
}

type broker struct {
	svc     lora.Service
	client  mqtt.Client
	logger  logger.Logger
	timeout time.Duration
}

// NewBroker returns new MQTT broker instance.
func NewBroker(svc lora.Service, client mqtt.Client, t time.Duration, log logger.Logger) Subscriber {
	return broker{
		svc:     svc,
		client:  client,
		logger:  log,
		timeout: t,
	}
}

// Subscribe subscribes to the Lora MQTT message broker
func (b broker) Subscribe(subject string) error {
	s := b.client.Subscribe(subject, 0, b.handleMsg)
	if err := s.Error(); s.WaitTimeout(b.timeout) && err != nil {
		return err
	}

	return nil
}

// handleMsg triggered when new message is received on Lora MQTT broker
func (b broker) handleMsg(c mqtt.Client, msg mqtt.Message) {
	m := lora.Message{}
	if err := json.Unmarshal(msg.Payload(), &m); err != nil {
		b.logger.Warn(fmt.Sprintf("Failed to unmarshal message: %s", err.Error()))
		return
	}

	b.svc.Publish(context.Background(), m)
	return
}
