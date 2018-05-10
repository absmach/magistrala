// Package http contains the domain concept definitions needed to support
// Mainflux http adapter service functionality.
package http

import "github.com/mainflux/mainflux"

var _ mainflux.MessagePublisher = (*adapterService)(nil)

type adapterService struct {
	pub mainflux.MessagePublisher
}

// New instantiates the HTTP adapter implementation.
func New(pub mainflux.MessagePublisher) mainflux.MessagePublisher {
	return &adapterService{pub}
}

func (as *adapterService) Publish(msg mainflux.RawMessage) error {
	return as.pub.Publish(msg)
}
