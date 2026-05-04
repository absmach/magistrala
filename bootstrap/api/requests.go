// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	apiutil "github.com/absmach/magistrala/api/http/util"
	"github.com/absmach/magistrala/bootstrap"
)

const maxLimitSize = 100

type addReq struct {
	token         string
	ExternalID    string         `json:"external_id"`
	ExternalKey   string         `json:"external_key"`
	Name          string         `json:"name"`
	Content       string         `json:"content"`
	ClientCert    string         `json:"client_cert"`
	ClientKey     string         `json:"client_key"`
	CACert        string         `json:"ca_cert"`
	ProfileID     string         `json:"profile_id"`
	RenderContext map[string]any `json:"render_context"`
}

func (req addReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.ExternalID == "" {
		return apiutil.ErrMissingID
	}

	if req.ExternalKey == "" {
		return apiutil.ErrBearerKey
	}

	return nil
}

type entityReq struct {
	id string
}

func (req entityReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type updateReq struct {
	id      string
	Name    string `json:"name"`
	Content string `json:"content"`
}

func (req updateReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type updateCertReq struct {
	clientID   string
	ClientCert string `json:"client_cert"`
	ClientKey  string `json:"client_key"`
	CACert     string `json:"ca_cert"`
}

func (req updateCertReq) validate() error {
	if req.clientID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type listReq struct {
	filter bootstrap.Filter
	offset uint64
	limit  uint64
}

func (req listReq) validate() error {
	if req.limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	return nil
}

type bootstrapReq struct {
	key string
	id  string
}

func (req bootstrapReq) validate() error {
	if req.key == "" {
		return apiutil.ErrBearerKey
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type changeConfigStatusReq struct {
	token string
	id    string
}

func (req changeConfigStatusReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

// --- Profile requests ---

type createProfileReq struct {
	bootstrap.Profile
}

func (req createProfileReq) validate() error {
	if req.Name == "" {
		return apiutil.ErrMissingID
	}
	return nil
}

type uploadProfileReq struct {
	bootstrap.Profile
}

func (req uploadProfileReq) validate() error {
	if req.Name == "" {
		return apiutil.ErrMissingID
	}
	return nil
}

type viewProfileReq struct {
	profileID string
}

func (req viewProfileReq) validate() error {
	if req.profileID == "" {
		return apiutil.ErrMissingID
	}
	return nil
}

type updateProfileReq struct {
	profileID string
	bootstrap.Profile
}

func (req updateProfileReq) validate() error {
	if req.profileID == "" || req.Name == "" {
		return apiutil.ErrMissingID
	}
	return nil
}

type renderPreviewReq struct {
	profileID     string
	Config        bootstrap.Config            `json:"config"`
	RenderContext map[string]any              `json:"render_context,omitempty"`
	Bindings      []bootstrap.BindingSnapshot `json:"bindings,omitempty"`
}

func (req renderPreviewReq) validate() error {
	if req.profileID == "" {
		return apiutil.ErrMissingID
	}
	return nil
}

type deleteProfileReq struct {
	profileID string
}

func (req deleteProfileReq) validate() error {
	if req.profileID == "" {
		return apiutil.ErrMissingID
	}
	return nil
}

type listProfilesReq struct {
	offset uint64
	limit  uint64
}

func (req listProfilesReq) validate() error {
	if req.limit == 0 || req.limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}
	return nil
}

// --- Enrollment binding requests ---

type assignProfileReq struct {
	configID  string
	ProfileID string `json:"profile_id"`
}

func (req assignProfileReq) validate() error {
	if req.configID == "" || req.ProfileID == "" {
		return apiutil.ErrMissingID
	}
	return nil
}

type bindResourcesReq struct {
	token    string
	configID string
	Bindings []bootstrap.BindingRequest `json:"bindings"`
}

func (req bindResourcesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.configID == "" {
		return apiutil.ErrMissingID
	}
	if len(req.Bindings) == 0 {
		return apiutil.ErrEmptyList
	}
	for _, b := range req.Bindings {
		if b.Slot == "" || b.Type == "" || b.ResourceID == "" {
			return apiutil.ErrMissingID
		}
	}
	return nil
}

type listBindingsReq struct {
	configID string
}

func (req listBindingsReq) validate() error {
	if req.configID == "" {
		return apiutil.ErrMissingID
	}
	return nil
}

type refreshBindingsReq struct {
	token    string
	configID string
}

func (req refreshBindingsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.configID == "" {
		return apiutil.ErrMissingID
	}
	return nil
}
