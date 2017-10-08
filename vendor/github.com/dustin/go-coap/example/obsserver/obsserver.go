package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/dustin/go-coap"
)

func periodicTransmitter(l *net.UDPConn, a *net.UDPAddr, m *coap.Message) {
	subded := time.Now()

	for {
		msg := coap.Message{
			Type:      coap.Acknowledgement,
			Code:      coap.Content,
			MessageID: m.MessageID,
			Payload:   []byte(fmt.Sprintf("Been running for %v", time.Since(subded))),
		}

		msg.SetOption(coap.ContentFormat, coap.TextPlain)
		msg.SetOption(coap.LocationPath, m.Path())

		log.Printf("Transmitting %v", msg)
		err := coap.Transmit(l, a, msg)
		if err != nil {
			log.Printf("Error on transmitter, stopping: %v", err)
			return
		}

		time.Sleep(time.Second)
	}
}

func main() {
	log.Fatal(coap.ListenAndServe("udp", ":5683",
		coap.FuncHandler(func(l *net.UDPConn, a *net.UDPAddr, m *coap.Message) *coap.Message {
			log.Printf("Got message path=%q: %#v from %v", m.Path(), m, a)
			if m.Code == coap.GET && m.Option(coap.Observe) != nil {
				if value, ok := m.Option(coap.Observe).([]uint8); ok &&
					len(value) >= 1 && value[0] == 1 {
					go periodicTransmitter(l, a, m)
				}
			}
			return nil
		})))
}
