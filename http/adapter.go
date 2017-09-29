package http

import "github.com/mainflux/mainflux/writer"

var _ Service = (*adapterService)(nil)

type adapterService struct {
	mr writer.MessageRepository
}

// NewService instantiates the domain service implementation.
func NewService(mr writer.MessageRepository) Service {
	return &adapterService{mr}
}

func (as *adapterService) Send(msg writer.RawMessage) {
	as.mr.Save(msg)
}
