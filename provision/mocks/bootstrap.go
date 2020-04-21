package mocks

import "github.com/mainflux/mainflux/bootstrap"

type bootstrapSvc struct{}

func (svc bootstrapSvc) Add(string, bootstrap.Config) (bootstrap.Config, error) {
	panic("not implemented")
}

func (svc bootstrapSvc) View(string, string) (bootstrap.Config, error) {
	panic("not implemented")
}

func (svc bootstrapSvc) Update(string, bootstrap.Config) error {
	panic("not implemented")
}

func (svc bootstrapSvc) UpdateCert(string, string, string, string, string) error {
	panic("not implemented")
}

func (svc bootstrapSvc) UpdateConnections(string, string, []string) error {
	panic("not implemented")
}

func (svc bootstrapSvc) List(string, bootstrap.Filter, uint64, uint64) (bootstrap.ConfigsPage, error) {
	panic("not implemented")
}

func (svc bootstrapSvc) Remove(string, string) error {
	panic("not implemented")
}

func (svc bootstrapSvc) Bootstrap(string, string, bool) (bootstrap.Config, error) {
	panic("not implemented")
}

func (svc bootstrapSvc) ChangeState(string, string, bootstrap.State) error {
	panic("not implemented")
}

func (svc bootstrapSvc) RemoveConfigHandler(string) error {
	panic("not implemented")
}

func (svc bootstrapSvc) UpdateChannelHandler(bootstrap.Channel) error {
	panic("not implemented")
}

func (svc bootstrapSvc) RemoveChannelHandler(string) error {
	panic("not implemented")
}

func (svc bootstrapSvc) DisconnectThingHandler(string, string) error {
	panic("not implemented")
}
