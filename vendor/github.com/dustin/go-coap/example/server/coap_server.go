package main

import (
	"log"
	"net"

	"github.com/dustin/go-coap"
)

func handleA(l *net.UDPConn, a *net.UDPAddr, m *coap.Message) *coap.Message {
	log.Printf("Got message in handleA: path=%q: %#v from %v", m.Path(), m, a)
	if m.IsConfirmable() {
		res := &coap.Message{
			Type:      coap.Acknowledgement,
			Code:      coap.Content,
			MessageID: m.MessageID,
			Token:     m.Token,
			Payload:   []byte("hello to you!"),
		}
		res.SetOption(coap.ContentFormat, coap.TextPlain)

		log.Printf("Transmitting from A %#v", res)
		return res
	}
	return nil
}

func handleB(l *net.UDPConn, a *net.UDPAddr, m *coap.Message) *coap.Message {
	log.Printf("Got message in handleB: path=%q: %#v from %v", m.Path(), m, a)
	if m.IsConfirmable() {
		res := &coap.Message{
			Type:      coap.Acknowledgement,
			Code:      coap.Content,
			MessageID: m.MessageID,
			Token:     m.Token,
			Payload:   []byte("good bye!"),
		}
		res.SetOption(coap.ContentFormat, coap.TextPlain)

		log.Printf("Transmitting from B %#v", res)
		return res
	}
	return nil
}

func main() {
	mux := coap.NewServeMux()
	mux.Handle("/a", coap.FuncHandler(handleA))
	mux.Handle("/b", coap.FuncHandler(handleB))

	log.Fatal(coap.ListenAndServe("udp", ":5683", mux))
}
