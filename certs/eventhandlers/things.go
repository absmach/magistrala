package eventhandlers

import (
	"context"

	"github.com/mainflux/mainflux/certs"
	"github.com/mainflux/mainflux/certs/pki"
	thingsEvent "github.com/mainflux/mainflux/internal/clients/events/things"
	"github.com/mainflux/mainflux/pkg/errors"
)

type things struct {
	pki  pki.Agent
	repo certs.Repository
}

var _ thingsEvent.EventHandler = (*things)(nil)

func NewThingsEventHandlers(repo certs.Repository, pki pki.Agent) thingsEvent.EventHandler {
	return &things{repo: repo, pki: pki}
}

func (teh *things) ThingCreated(ctx context.Context, cte thingsEvent.CreateThingEvent) error {
	return nil
}
func (teh *things) ThingUpdated(ctx context.Context, ute thingsEvent.UpdateThingEvent) error {
	return nil
}
func (teh *things) ThingRemoved(ctx context.Context, rte thingsEvent.RemoveThingEvent) error {
	cp, err := teh.repo.RetrieveThingCerts(ctx, rte.ID)
	if err != nil {
		return err
	}

	// create async thing event handler with go routine and return error via channels
	var retErr error
	for _, cert := range cp.Certs {
		_, err := teh.pki.Revoke(cert.Serial)
		if err != nil {
			retErr = errors.Wrap(retErr, err)
		}
	}
	err = teh.repo.RemoveThingCerts(ctx, rte.ID)
	if err != nil {
		retErr = errors.Wrap(retErr, err)
	}
	return retErr
}
func (teh *things) ChannelCreated(ctx context.Context, cce thingsEvent.CreateChannelEvent) error {
	return nil
}
func (teh *things) ChannelUpdated(ctx context.Context, uce thingsEvent.UpdateChannelEvent) error {
	return nil
}
func (teh *things) ChannelRemoved(ctx context.Context, rce thingsEvent.RemoveChannelEvent) error {
	return nil
}
func (teh *things) ThingConnected(ctx context.Context, cte thingsEvent.ConnectThingEvent) error {
	return nil
}
func (teh *things) ThingDisconnected(ctx context.Context, dte thingsEvent.DisconnectThingEvent) error {
	return nil
}
