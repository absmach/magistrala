package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/mainflux/mproxy/pkg/logger"
	"github.com/mainflux/mproxy/pkg/session"
)

const contentType = "application/json"

// ErrMissingAuthentication returned when no basic or Authorization header is set.
var ErrMissingAuthentication = errors.New("missing authorization")

// Handler default handler reads authorization header and
// performs authorization before proxying the request.
func (p *Proxy) Handler(w http.ResponseWriter, r *http.Request) {
	username, password, ok := r.BasicAuth()
	switch {
	case ok:
		break
	case r.Header.Get("Authorization") != "":
		password = r.Header.Get("Authorization")
	default:
		encodeError(w, http.StatusBadGateway, ErrMissingAuthentication)
		return
	}

	s := &session.Session{
		Password: []byte(password),
		Username: username,
	}
	ctx := session.NewContext(r.Context(), s)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		encodeError(w, http.StatusBadRequest, err)
		p.logger.Error(err.Error())
		return
	}
	if err := r.Body.Close(); err != nil {
		encodeError(w, http.StatusInternalServerError, err)
		p.logger.Error(err.Error())
		return
	}

	// r.Body is reset to ensure it can be safely copied by httputil.ReverseProxy.
	// no close method is required since NopClose Close() always returns nill.
	r.Body = io.NopCloser(bytes.NewBuffer(payload))
	if err := p.session.AuthConnect(ctx); err != nil {
		encodeError(w, http.StatusUnauthorized, err)
		p.logger.Error(err.Error())
		return
	}
	if err := p.session.Publish(ctx, &r.RequestURI, &payload); err != nil {
		encodeError(w, http.StatusBadRequest, err)
		p.logger.Error(err.Error())
		return
	}
	p.target.ServeHTTP(w, r)
}

func encodeError(w http.ResponseWriter, statusCode int, err error) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", contentType)
	if err := json.NewEncoder(w).Encode(err); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// Proxy represents HTTP Proxy.
type Proxy struct {
	address string
	target  *httputil.ReverseProxy
	session session.Handler
	logger  logger.Logger
}

func NewProxy(address, targetUrl string, handler session.Handler, logger logger.Logger) (Proxy, error) {
	target, err := url.Parse(targetUrl)
	if err != nil {
		return Proxy{}, err
	}

	return Proxy{
		address: address,
		target:  httputil.NewSingleHostReverseProxy(target),
		session: handler,
		logger:  logger,
	}, nil
}

func (p *Proxy) Listen() error {
	if err := http.ListenAndServe(p.address, nil); err != nil {
		return err
	}

	p.logger.Info("Server Exiting...")
	return nil
}

func (p *Proxy) ListenTLS(cert, key string) error {
	if err := http.ListenAndServeTLS(p.address, cert, key, nil); err != nil {
		return err
	}

	p.logger.Info("Server Exiting...")
	return nil
}
