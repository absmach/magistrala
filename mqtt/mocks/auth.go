package mocks

import (
	"context"
	"github.com/mainflux/mainflux/pkg/auth"
	"github.com/mainflux/mainflux/pkg/errors"
)

type MockClient struct {
	key   map[string]string
	conns map[string]interface{}
}

func NewClient(key map[string]string, conns map[string]interface{}) auth.Client {
	return MockClient{key: key, conns: conns}
}

func (cli MockClient) Authorize(ctx context.Context, chanID, thingID string) error {
	for k, v := range cli.conns {
		if k == chanID && v == thingID {
			return nil
		}
	}
	return errors.ErrAuthentication
}

func (cli MockClient) Identify(ctx context.Context, thingKey string) (string, error) {
	if id, ok := cli.key[thingKey]; ok {
		return id, nil
	}
	return "", errors.ErrAuthentication
}
