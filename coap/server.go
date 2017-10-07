package adapter

import (
	"log"
	"net"

	"github.com/dereulenspiegel/coap-mux"
	"github.com/dustin/go-coap"
)

func notFoundHandler(l *net.UDPConn, a *net.UDPAddr, m *coap.Message) *coap.Message {
	log.Printf("Got message in notFoundHandler: path=%q: %#v from %v", m.Path(), m, a)
	if m.IsConfirmable() {
		return &coap.Message{
			Type: coap.Acknowledgement,
			Code: coap.NotFound,
		}
	}
	return nil
}

func (ca *CoAPAdapter) COAPServer() *mux.Router {
	r := mux.NewRouter()
	r.Handle("/channels/{channel_id}/messages", coap.FuncHandler(ca.sendMessage)).Methods(coap.POST)
	r.Handle("/channels/{channel_id}/messages", coap.FuncHandler(ca.observeMessage)).Methods(coap.GET)

	r.NotFoundHandler = coap.FuncHandler(notFoundHandler)

	return r
}
