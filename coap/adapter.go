package adapter

import (
	"encoding/json"
	"log"
	"net"

	"github.com/dustin/go-coap"
	"github.com/mainflux/mainflux/writer"
	broker "github.com/nats-io/go-nats"
	"go.uber.org/zap"
)

const protocol string = "coap"

type Observer struct {
	conn    *net.UDPConn
	addr    *net.UDPAddr
	message *coap.Message
}

type CoAPAdapter struct {
	obsMap map[string][]Observer
	logger *zap.Logger
	repo   writer.MessageRepository
}

// NewCoAPAdapter creates new CoAP adapter struct
func NewCoAPAdapter(logger *zap.Logger, repo writer.MessageRepository) *CoAPAdapter {
	ca := &CoAPAdapter{
		logger: logger,
		repo:   repo,
	}

	ca.obsMap = make(map[string][]Observer)

	return ca
}

// Serve function starts CoAP server
func (ca *CoAPAdapter) Serve(addr string) error {
	ca.logger.Info("Starting CoAP server", zap.String("address", addr))
	return coap.ListenAndServe("udp", addr, ca.COAPServer())
}

// BridgeHandler functions is a handler for messages recieved via NATS
func (ca *CoAPAdapter) BridgeHandler(nm *broker.Msg) {
	log.Printf("Received a message: %s\n", string(nm.Data))

	// And write it into the database
	m := writer.RawMessage{}
	if len(nm.Data) > 0 {
		if err := json.Unmarshal(nm.Data, &m); err != nil {
			log.Println("Can not decode adapter msg")
			return
		}
	}

	log.Println("Calling obsTransmit()")
	log.Println(m.Publisher, m.Protocol, m.Channel, m.Payload)
	ca.obsTransmit(m)
}
