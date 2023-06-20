// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"time"

	mflog "github.com/mainflux/mainflux/logger"
	mfclients "github.com/mainflux/mainflux/pkg/clients"
	"github.com/mainflux/mainflux/things/clients"
)

var _ clients.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger mflog.Logger
	svc    clients.Service
}

func LoggingMiddleware(svc clients.Service, logger mflog.Logger) clients.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) CreateThings(ctx context.Context, token string, clients ...mfclients.Client) (cs []mfclients.Client, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_things %d things using token %s took %s to complete", len(cs), token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.CreateThings(ctx, token, clients...)
}

func (lm *loggingMiddleware) ViewClient(ctx context.Context, token, id string) (c mfclients.Client, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_thing for thing with id %s using token %s took %s to complete", id, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.ViewClient(ctx, token, id)
}

func (lm *loggingMiddleware) ListClients(ctx context.Context, token string, pm mfclients.Page) (cp mfclients.ClientsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_things using token %s took %s to complete", token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.ListClients(ctx, token, pm)
}

func (lm *loggingMiddleware) UpdateClient(ctx context.Context, token string, client mfclients.Client) (c mfclients.Client, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_thing_name_and_metadata for thing with id %s using token %s took %s to complete", c.ID, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.UpdateClient(ctx, token, client)
}

func (lm *loggingMiddleware) UpdateClientTags(ctx context.Context, token string, client mfclients.Client) (c mfclients.Client, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_thing_tags for thing with id %s using token %s took %s to complete", c.ID, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.UpdateClientTags(ctx, token, client)
}

func (lm *loggingMiddleware) UpdateClientSecret(ctx context.Context, token, oldSecret, newSecret string) (c mfclients.Client, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_thing_secret for thing with id %s using token %s took %s to complete", c.ID, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.UpdateClientSecret(ctx, token, oldSecret, newSecret)
}

func (lm *loggingMiddleware) UpdateClientOwner(ctx context.Context, token string, client mfclients.Client) (c mfclients.Client, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_thing_owner for thing with id %s using token %s took %s to complete", c.ID, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.UpdateClientOwner(ctx, token, client)
}

func (lm *loggingMiddleware) EnableClient(ctx context.Context, token string, id string) (c mfclients.Client, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method enable_thing for thing with id %s using token %s took %s to complete", id, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.EnableClient(ctx, token, id)
}

func (lm *loggingMiddleware) DisableClient(ctx context.Context, token string, id string) (c mfclients.Client, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method disable_thing for thing with id %s using token %s took %s to complete", id, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.DisableClient(ctx, token, id)
}

func (lm *loggingMiddleware) ListClientsByGroup(ctx context.Context, token, channelID string, cp mfclients.Page) (mp mfclients.MembersPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_things_by_channel for channel with id %s using token %s took %s to complete", channelID, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.ListClientsByGroup(ctx, token, channelID, cp)
}

func (lm *loggingMiddleware) Identify(ctx context.Context, key string) (id string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method identify for thing with id %s and key %s took %s to complete", id, key, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.Identify(ctx, key)
}
