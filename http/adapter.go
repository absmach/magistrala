package http

import "github.com/mainflux/mainflux"

var _ mainflux.MessagePublisher = (*adapterService)(nil)

type adapterService struct {
	pub mainflux.MessagePublisher
}

// New instantiates the domain service implementation.
func New(pub mainflux.MessagePublisher) mainflux.MessagePublisher {
	return &adapterService{pub}
}

func (as *adapterService) Publish(msg mainflux.RawMessage) error {
	return as.pub.Publish(msg)
}
