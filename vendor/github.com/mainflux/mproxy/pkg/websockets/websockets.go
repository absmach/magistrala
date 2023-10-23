package websockets

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/mainflux/mproxy/pkg/logger"
	"github.com/mainflux/mproxy/pkg/session"
	"golang.org/x/sync/errgroup"
)

var (
	upgrader               = websocket.Upgrader{}
	ErrAuthorizationNotSet = errors.New("authorization not set")
)

type Proxy struct {
	target  string
	address string
	event   session.Handler
	logger  logger.Logger
}

func (p *Proxy) Handler(w http.ResponseWriter, r *http.Request) {
	var token string
	headers := http.Header{}
	switch {
	case len(r.URL.Query()["authorization"]) != 0:
		token = r.URL.Query()["authorization"][0]
	case r.Header.Get("Authorization") != "":
		token = r.Header.Get("Authorization")
		headers.Add("Authorization", token)
	default:
		http.Error(w, ErrAuthorizationNotSet.Error(), http.StatusUnauthorized)
		return
	}

	target := fmt.Sprintf("%s%s", p.target, r.RequestURI)

	targetConn, _, err := websocket.DefaultDialer.Dial(target, headers)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer targetConn.Close()

	topic := r.URL.Path
	s := session.Session{Password: []byte(token)}
	ctx := session.NewContext(context.Background(), &s)
	if err := p.event.AuthConnect(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if err := p.event.AuthSubscribe(ctx, &[]string{topic}); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if err := p.event.Subscribe(ctx, &[]string{topic}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	inConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		p.logger.Warn(err.Error())
		return
	}
	defer inConn.Close()

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return p.stream(ctx, topic, inConn, targetConn, true)
	})
	g.Go(func() error {
		return p.stream(ctx, topic, targetConn, inConn, false)
	})

	if err := g.Wait(); err != nil {
		if err := p.event.Unsubscribe(ctx, &[]string{topic}); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		p.logger.Error(fmt.Sprintf("ws Proxy terminated: %s", err))
		return
	}
}

func (p *Proxy) stream(ctx context.Context, topic string, src, dest *websocket.Conn, upstream bool) error {
	for {
		messageType, payload, err := src.ReadMessage()
		if err != nil {
			return err
		}
		if upstream {
			if err := p.event.AuthPublish(ctx, &topic, &payload); err != nil {
				return err
			}
			if err := p.event.Publish(ctx, &topic, &payload); err != nil {
				return err
			}
		}
		if err := dest.WriteMessage(messageType, payload); err != nil {
			return err
		}
	}
}

func NewProxy(address, target string, logger logger.Logger, handler session.Handler) (*Proxy, error) {
	return &Proxy{target: target, address: address, logger: logger, event: handler}, nil
}

// Listen - listen withrout tls.
func (p *Proxy) Listen() error {
	return http.ListenAndServe(p.address, http.HandlerFunc(p.Handler))
}

// ListenTLS - version of Listen with TLS encryption.
func (p Proxy) ListenTLS(crt, key string) error {
	return http.ListenAndServeTLS(p.address, crt, key, http.HandlerFunc(p.Handler))
}
