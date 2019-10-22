// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/users"
)

func registrationEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(userReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		err := svc.Register(ctx, req.user)
		return tokenRes{}, err
	}
}

// Password reset request endpoint.
// When successful password reset link is generated.
// Link is generated using MF_TOKEN_RESET_ENDPOINT env.
// and value from Referer header for host.
// {Referer}+{MF_TOKEN_RESET_ENDPOINT}+{token=TOKEN}
// http://mainflux.com/reset-request?token=xxxxxxxxxxx.
// Email with a link is being sent to the user.
// When user clicks on a link it should get the ui with form to
// enter new password, when form is submitted token and new password
// must be sent as PUT request to 'password/reset' passwordResetEndpoint
func passwordResetRequestEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(passwResetReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		res := passwChangeRes{}
		email := req.Email

		if err := svc.GenerateResetToken(ctx, email, req.Host); err != nil {
			res.Msg = err.Error()
			return res, nil
		}
		res.Msg = MailSent
		return res, nil
	}
}

// This is endpoint that actually sets new password in password reset flow.
// When user clicks on a link in email finally ends on this endpoint as explained in
// the comment above.
func passwordResetEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(resetTokenReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		res := passwChangeRes{}

		if err := svc.ResetPassword(ctx, req.Token, req.Password); err != nil {
			res.Msg = err.Error()
			return res, nil
		}
		res.Msg = ""
		return res, nil
	}
}

func userInfoEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewUserInfoReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		u, err := svc.UserInfo(ctx, req.token)
		if err != nil {
			return nil, err
		}

		return identityRes{u.Email, u.Metadata}, nil
	}
}

func passwordChangeEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(passwChangeReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		res := passwChangeRes{}

		if err := svc.ChangePassword(ctx, req.Token, req.Password, req.OldPassword); err != nil {
			res.Msg = err.Error()
			return res, nil
		}

		return res, nil
	}
}

func loginEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(userReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		token, err := svc.Login(ctx, req.user)
		if err != nil {
			return nil, err
		}

		return tokenRes{token}, nil
	}
}
