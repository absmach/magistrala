// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"time"

	"github.com/absmach/magistrala"
	mglog "github.com/absmach/magistrala/logger"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/users"
)

var _ users.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger mglog.Logger
	svc    users.Service
}

// LoggingMiddleware adds logging facilities to the clients service.
func LoggingMiddleware(svc users.Service, logger mglog.Logger) users.Service {
	return &loggingMiddleware{logger, svc}
}

// RegisterClient logs the register_client request. It logs the client id and token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) RegisterClient(ctx context.Context, token string, client mgclients.Client) (c mgclients.Client, err error) {
	defer func(begin time.Time) {
		fields := map[string]interface{}{
			"method":   "register_client",
			"id":       c.ID,
			"token":    token,
			"duration": time.Since(begin),
		}
		message := fmt.Sprintf("Method %s with id %s using token %s took %s to complete", fields["method"], fields["id"], fields["token"], fields["duration"])
		if err != nil {
			fields["error"] = err.Error()
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, fields["error"]))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.RegisterClient(ctx, token, client)
}

// IssueToken logs the issue_token request. It logs the client identity and token type and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) IssueToken(ctx context.Context, identity, secret, domainID string) (t *magistrala.Token, err error) {
	defer func(begin time.Time) {
		fields:= map[string]interface{}{
			"method": "issue_token",
			"access": t.AccessType,
			"identity": identity,
			"duration": time.Since(begin),
		}
		message := fmt.Sprintf("Method %s", fields["method"])
		if t != nil {
			message = fmt.Sprintf("%s of type %s", message, fields["access"])
		}
		message = fmt.Sprintf("%s for client %s took %s to complete", message, fields["identity"], fields["duration"])
		if err != nil {
			fields["error"] = err.Error()
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, fields["error"]))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.IssueToken(ctx, identity, secret, domainID)
}

// RefreshToken logs the refresh_token request. It logs the refreshtoken, token type and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) RefreshToken(ctx context.Context, refreshToken, domainID string) (t *magistrala.Token, err error) {
	defer func(begin time.Time) {
		fields:= map[string]interface{}{
			"method": "refresh_token",
			"access": t.AccessType,
			"refreshToken": refreshToken,
			"duration": time.Since(begin),
		}
		message := fmt.Sprintf("Method %s", fields["method"])
		if t != nil {
			message = fmt.Sprintf("%s of type %s", message, fields["access"])
		}
		message = fmt.Sprintf("%s for refresh token %s took %s to complete", message, fields["refreshToken"], fields["duration"])
		if err != nil {
			fields["error"] = err.Error()
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, fields["error"]))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.RefreshToken(ctx, refreshToken, domainID)
}

// ViewClient logs the view_client request. It logs the client id and token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ViewClient(ctx context.Context, token, id string) (c mgclients.Client, err error) {
	defer func(begin time.Time) {
		fields:= map[string]interface{}{
			"method": "view_client",
			"token": token,
			"duration": time.Since(begin),
		}
		message := fmt.Sprintf("Method %s using token %s took %s to complete",fields["method"], fields["token"], fields["duration"])
		if err != nil {
			fields["error"] = err.Error()
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, fields["error"]))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.ViewClient(ctx, token, id)
}

// ViewProfile logs the view_profile request. It logs the client id and token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ViewProfile(ctx context.Context, token string) (c mgclients.Client, err error) {
	defer func(begin time.Time) {
		fields:= map[string]interface{}{
			"method": "view_client",
			"id": c.ID,
			"token": token,
			"duration": time.Since(begin),
		}
		message := fmt.Sprintf("Method %s with id %s using token %s took %s to complete", fields["method"], fields["id"], fields["token"], fields["duration"])
		if err != nil {
			fields["error"] = err.Error()
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, fields["error"]))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.ViewProfile(ctx, token)
}

// ListClients logs the list_clients request. It logs the token and page metadata and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ListClients(ctx context.Context, token string, pm mgclients.Page) (cp mgclients.ClientsPage, err error) {
	defer func(begin time.Time) {
		fields := map[string]interface{}{
			"method":   "list_client",
			"token":    token,
			"total":    cp.Total,
			"duration": time.Since(begin),
		}
		message := fmt.Sprintf("Method %s listed %d clients using token %s took %s to complete", fields["method"], fields["total"], fields["token"], fields["duration"])
        if err != nil {
            fields["error"] = err.Error()
            lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, fields["error"]))
            return
        }
        lm.logger.Info(fmt.Sprintf("%s without errors.", message))
    }(time.Now())
    return lm.svc.ListClients(ctx, token, pm)
}

// UpdateClient logs the update_client request. It logs the client id and token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) UpdateClient(ctx context.Context, token string, client mgclients.Client) (c mgclients.Client, err error) {
	defer func(begin time.Time) {
		fields := map[string]interface{}{
			"method":   "update_client_name_and_metadata",
			"token":    token,
			"id":       c.ID,
			"duration": time.Since(begin),
		}
		message := fmt.Sprintf("Method %s for client with id %s using token %s took %s to complete", fields["method"], fields["id"], fields["token"], fields["duration"])
		if err != nil {
			fields["error"] = err.Error()
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, fields["error"]))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.UpdateClient(ctx, token, client)
}

// UpdateClientTags logs the update_client_tags request. It logs the client id and token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) UpdateClientTags(ctx context.Context, token string, client mgclients.Client) (c mgclients.Client, err error) {
	defer func(begin time.Time) {
		fields := map[string]interface{}{
			"method":   "update_client_tags",
			"token":    token,
			"id":       c.ID,
			"duration": time.Since(begin),
		}
		message := fmt.Sprintf("Method %s for client with id %s using token %s took %s to complete", fields["method"], fields["id"], fields["token"], fields["duration"])
		if err != nil {
			fields["error"] = err.Error()
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, fields["error"]))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.UpdateClientTags(ctx, token, client)
}

// UpdateClientIdentity logs the update_client_identity request. It logs the client id and token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) UpdateClientIdentity(ctx context.Context, token, id, identity string) (c mgclients.Client, err error) {
	defer func(begin time.Time) {
		fields := map[string]interface{}{
			"method":   "update_client_identity",
			"token":    token,
			"id":       c.ID,
			"duration": time.Since(begin),
		}
		message := fmt.Sprintf("Method %s for client with id %s using token %s took %s to complete", fields["method"], fields["id"], fields["token"], fields["duration"])
		if err != nil {
			fields["error"] = err.Error()
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, fields["error"]))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.UpdateClientIdentity(ctx, token, id, identity)
}

// UpdateClientSecret logs the update_client_secret request. It logs the client id and token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) UpdateClientSecret(ctx context.Context, token, oldSecret, newSecret string) (c mgclients.Client, err error) {
	defer func(begin time.Time) {
		fields := map[string]interface{}{
			"method":   "update_client_secret",
			"token":    token,
			"id":       c.ID,
			"duration": time.Since(begin),
		}
		message := fmt.Sprintf("Method %s for client with id %s using token %s took %s to complete", fields["method"], fields["id"], fields["token"], fields["duration"])
		if err != nil {
			fields["error"] = err.Error()
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, fields["error"]))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.UpdateClientSecret(ctx, token, oldSecret, newSecret)
}

// GenerateResetToken logs the generate_reset_token request. It logs the email and host and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) GenerateResetToken(ctx context.Context, email, host string) (err error) {
	defer func(begin time.Time) {
		fields := map[string]interface{}{
			"method":   "generate_reset_token",
			"email":       email,
			"host": host,
			"duration": time.Since(begin),
		}
		message := fmt.Sprintf("Method %s for email %s and host %s took %s to complete", fields["method"], fields["email"], fields["host"], fields["duration"])
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.GenerateResetToken(ctx, email, host)
}

// ResetSecret logs the reset_secret request. It logs the token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ResetSecret(ctx context.Context, token, secret string) (err error) {
	defer func(begin time.Time) {
		fields := map[string]interface{}{
			"method":   "reset_secret",
			"token":    token,
			"duration": time.Since(begin),
		}
		message := fmt.Sprintf("Method %s using token %s took %s to complete",  fields["method"], fields["token"], fields["duration"])
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.ResetSecret(ctx, token, secret)
}

// SendPasswordReset logs the send_password_reset request. It logs the token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) SendPasswordReset(ctx context.Context, host, email, user, token string) (err error) {
	defer func(begin time.Time) {
		fields := map[string]interface{}{
			"method":   "send_password_reset",
			"token":    token,
			"duration": time.Since(begin),
		}
		message := fmt.Sprintf("Method %s using token %s took %s to complete", fields["method"], fields["token"], fields["duration"])
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.SendPasswordReset(ctx, host, email, user, token)
}

// UpdateClientRole logs the update_client_role request. It logs the client id and token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) UpdateClientRole(ctx context.Context, token string, client mgclients.Client) (c mgclients.Client, err error) {
	defer func(begin time.Time) {
		fields := map[string]interface{}{
			"method":   "update_client_role",
			"id":       c.ID,
			"token":    token,
			"duration": time.Since(begin),
		}
		message := fmt.Sprintf("Method %s for client with id %s using token %s took %s to complete",  fields["method"], fields["id"], fields["token"], fields["duration"])
		if err != nil {
			fields["error"] = err.Error()
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, fields["error"]))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.UpdateClientRole(ctx, token, client)
}

// EnableClient logs the enable_client request. It logs the client id and token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) EnableClient(ctx context.Context, token, id string) (c mgclients.Client, err error) {
	defer func(begin time.Time) {
		fields := map[string]interface{}{
			"method":   "enable_client",
			"id":       c.ID,
			"token":    token,
			"duration": time.Since(begin),
		}
		message := fmt.Sprintf("Method %s for client with id %s using token %s took %s to complete",  fields["method"], fields["id"], fields["token"], fields["duration"])
		if err != nil {
			fields["error"] = err.Error()
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, fields["error"]))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.EnableClient(ctx, token, id)
}

// DisableClient logs the disable_client request. It logs the client id and token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) DisableClient(ctx context.Context, token, id string) (c mgclients.Client, err error) {
	defer func(begin time.Time) {
		fields := map[string]interface{}{
			"method":   "disable_client",
			"id":       c.ID,
			"token":    token,
			"duration": time.Since(begin),
		}
		message := fmt.Sprintf("Method %s for client with id %s using token %s took %s to complete", fields["method"], fields["id"], fields["token"], fields["duration"])
		if err != nil {
			fields["error"] = err.Error()
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, fields["error"]))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.DisableClient(ctx, token, id)
}

// ListMembers logs the list_members request. It logs the group id, token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ListMembers(ctx context.Context, token, objectKind, objectID string, cp mgclients.Page) (mp mgclients.MembersPage, err error) {
	defer func(begin time.Time) {
		fields := map[string]interface{}{
			"method":   "list_members",
			"page": mp.Total,
			"objectKind":    objectKind,
			"objectID":       objectID,
			"token":    token,
			"duration": time.Since(begin),
		}
		message := fmt.Sprintf("Method list_members %d members for object kind %s and object id %s and token %s took %s to complete", fields["page"], fields["objectKind"], fields["objectID"], fields["token"], fields["duration"])
		if err != nil {
			fields["error"] = err.Error()
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, fields["error"]))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.ListMembers(ctx, token, objectKind, objectID, cp)
}

// Identify logs the identify request. It logs the token and the time it took to complete the request.
func (lm *loggingMiddleware) Identify(ctx context.Context, token string) (id string, err error) {
	defer func(begin time.Time) {
		fields := map[string]interface{}{
			"method":   "identify",
			"id":       id,
			"token":    token,
			"duration": time.Since(begin),
		}
		message := fmt.Sprintf("Method %s for token %s with id %s took %s to complete", fields["method"], fields["id"], fields["token"], fields["duration"])
		if err != nil {
			fields["error"] = err.Error()
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, fields["error"]))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.Identify(ctx, token)
}
