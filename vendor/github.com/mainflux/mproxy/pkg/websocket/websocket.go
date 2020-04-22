package websocket

import (
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mproxy/pkg/session"
)

// Proxy represents WS Proxy.
type Proxy struct {
	target string
	path   string
	scheme string
	event  session.Handler
	logger logger.Logger
}

// New - creates new HTTP proxy
func New(target, path, scheme string, event session.Handler, logger logger.Logger) *Proxy {
	return &Proxy{
		target: target,
		path:   path,
		scheme: scheme,
		event:  event,
		logger: logger,
	}
}

var upgrader = websocket.Upgrader{
	// Timeout for WS upgrade request handshake
	HandshakeTimeout: 10 * time.Second,
	// Paho JS client expecting header Sec-WebSocket-Protocol:mqtt in Upgrade response during handshake.
	Subprotocols: []string{"mqtt"},
	// Allow CORS
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Handler - proxies HTTP traffic
func (p Proxy) Handler() http.Handler {
	return p.handle()
}

func (p Proxy) handle() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cconn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			p.logger.Error("Error upgrading connection " + err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		go p.pass(cconn)
	})
}

func (p Proxy) pass(in *websocket.Conn) {
	defer in.Close()

	url := url.URL{
		Scheme: p.scheme,
		Host:   p.target,
		Path:   p.path,
	}

	dialer := &websocket.Dialer{
		Subprotocols: []string{"mqtt"},
	}
	srv, _, err := dialer.Dial(url.String(), nil)

	if err != nil {
		p.logger.Error("Unable to connect to broker, reason: " + err.Error())
		return
	}

	errc := make(chan error, 1)
	c := newConn(in)
	s := newConn(srv)

	defer s.Close()
	defer c.Close()

	session := session.New(c, s, p.event, p.logger)
	err = session.Stream()
	errc <- err
	p.logger.Warn("Broken connection for client: " + session.Client.ID + " with error: " + err.Error())
}
