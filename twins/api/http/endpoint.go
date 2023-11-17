// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/twins"
	"github.com/go-kit/kit/endpoint"
)

func addTwinEndpoint(svc twins.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addTwinReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		twin := twins.Twin{
			Name:     req.Name,
			Metadata: req.Metadata,
		}
		saved, err := svc.AddTwin(ctx, req.token, twin, req.Definition)
		if err != nil {
			return nil, err
		}

		res := twinRes{
			id:      saved.ID,
			created: true,
		}
		return res, nil
	}
}

func updateTwinEndpoint(svc twins.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateTwinReq)

		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		twin := twins.Twin{
			ID:       req.id,
			Name:     req.Name,
			Metadata: req.Metadata,
		}

		if err := svc.UpdateTwin(ctx, req.token, twin, req.Definition); err != nil {
			return nil, err
		}

		res := twinRes{id: req.id, created: false}
		return res, nil
	}
}

func viewTwinEndpoint(svc twins.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewTwinReq)

		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		twin, err := svc.ViewTwin(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		res := viewTwinRes{
			Owner:       twin.Owner,
			ID:          twin.ID,
			Name:        twin.Name,
			Created:     twin.Created,
			Updated:     twin.Updated,
			Revision:    twin.Revision,
			Definitions: twin.Definitions,
			Metadata:    twin.Metadata,
		}
		return res, nil
	}
}

func listTwinsEndpoint(svc twins.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listReq)

		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		page, err := svc.ListTwins(ctx, req.token, req.offset, req.limit, req.name, req.metadata)
		if err != nil {
			return nil, err
		}

		res := twinsPageRes{
			pageRes: pageRes{
				Total:  page.Total,
				Offset: page.Offset,
				Limit:  page.Limit,
			},
			Twins: []viewTwinRes{},
		}
		for _, twin := range page.Twins {
			view := viewTwinRes{
				Owner:       twin.Owner,
				ID:          twin.ID,
				Name:        twin.Name,
				Created:     twin.Created,
				Updated:     twin.Updated,
				Revision:    twin.Revision,
				Definitions: twin.Definitions,
				Metadata:    twin.Metadata,
			}
			res.Twins = append(res.Twins, view)
		}

		return res, nil
	}
}

func removeTwinEndpoint(svc twins.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewTwinReq)

		err := req.validate()
		if err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		if err := svc.RemoveTwin(ctx, req.token, req.id); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func listStatesEndpoint(svc twins.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listStatesReq)

		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		page, err := svc.ListStates(ctx, req.token, req.offset, req.limit, req.id)
		if err != nil {
			return nil, err
		}

		res := statesPageRes{
			pageRes: pageRes{
				Total:  page.Total,
				Offset: page.Offset,
				Limit:  page.Limit,
			},
			States: []viewStateRes{},
		}
		for _, state := range page.States {
			view := viewStateRes{
				TwinID:     state.TwinID,
				ID:         state.ID,
				Definition: state.Definition,
				Created:    state.Created,
				Payload:    state.Payload,
			}
			res.States = append(res.States, view)
		}

		return res, nil
	}
}
