// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/commands"
)

func createCommandEndpoint(svc commands.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createCommandReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		cmd := commands.Command{
			Name:        req.Name,
			Command:     req.Command,
			ChannelID:   req.ChannelID,
			ExecuteTime: req.ExecuteTime,
		}
		id, err := svc.CreateCommand(req.token, cmd)
		if err != nil {
			return nil, err
		}
		res := createCommandRes{
			ID:      id,
			created: true,
		}
		return res, nil
	}
}

func viewCommandEndpoint(svc commands.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewCommandReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		cmd, err := svc.ViewCommand(req.token, req.id)
		if err != nil {
			return nil, err
		}

		res := viewCommandRes{
			ID:          cmd.ID,
			Owner:       cmd.Owner,
			Name:        cmd.Name,
			Command:     cmd.Command,
			ChannelID:   cmd.ChannelID,
			ExecuteTime: cmd.ExecuteTime,
			Metadata:    cmd.Metadata,
		}
		return res, nil
	}
}

func listCommandEndpoint(svc commands.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listCommandReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		// page, err := svc.ListCommands(ctx)
		// if err != nil {
		// 	return nil, err
		// }

		// res := commandsPageRes{
		// 	pageRes: pageRes{
		// 		Total:  page.Total,
		// 		Offset: page.Offset,
		// 		Limit:  page.Limit,
		// 		Order:  page.Order,
		// 		Dir:    page.Dir,
		// 	},
		// 	Commands: []viewCommandRes{},
		// }
		// for _, command := range page.Commands {
		// 	view := viewCommandRes{
		// 		ID:       command.ID,
		// 		Metadata: command.Metadata,
		// 	}
		// 	res.Commands = append(res.Commands, view)
		// }
		// return res, nil
		return nil, nil
	}
}

func updateCommandEndpoint(svc commands.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateCommandReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		cmd := commands.Command{
			Command:     req.Command,
			Name:        req.Name,
			ExecuteTime: req.ExecuteTime,
			Metadata:    req.Metadata,
		}
		if err := svc.UpdateCommand(req.token, cmd); err != nil {
			return nil, err
		}

		res := updateCommandRes{}

		return res, nil
	}
}

func removeCommandEndpoint(svc commands.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(removeCommandReq)

		err := req.validate()
		if err == commands.ErrMalformedEntity {
			return removeCommandRes{}, nil
		}
		if err != nil {
			return nil, err
		}
		if err := svc.RemoveCommand(req.token, req.id); err != nil {
			return nil, err
		}
		return removeCommandRes{}, nil
	}
}
