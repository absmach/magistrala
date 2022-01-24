// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mainflux

import (
	"encoding/json"
	"net/http"
)

const (
	contentType     = "Content-Type"
	contentTypeJSON = "application/health+json"
	svcStatus       = "pass"
	description     = " service"
)

var (
	// Version represents the last service git tag in git history.
	// It's meant to be set using go build ldflags:
	// -ldflags "-X 'github.com/mainflux/mainflux.Version=0.0.0'"
	Version = "0.0.0"
	// Commit represents the service git commit hash.
	// It's meant to be set using go build ldflags:
	// -ldflags "-X 'github.com/mainflux/mainflux.Commit=ffffffff'"
	Commit = "ffffffff"
	// BuildTime represetns the service build time.
	// It's meant to be set using go build ldflags:
	// -ldflags "-X 'github.com/mainflux/mainflux.BuildTime=1970-01-01_00:00:00'"
	BuildTime = "1970-01-01_00:00:00"
)

// HealthInfo contains version endpoint response.
type HealthInfo struct {
	// Status contains service status.
	Status string `json:"status"`

	// Version contains current service version.
	Version string `json:"version"`

	// Commit represents the git hash commit.
	Commit string `json:"commit"`

	// Description contains service description.
	Description string `json:"description"`

	// BuildTime contains service build time.
	BuildTime string `json:"build_time"`
}

// Health exposes an HTTP handler for retrieving service health.
func Health(service string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add(contentType, contentTypeJSON)
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		res := HealthInfo{
			Status:      svcStatus,
			Version:     Version,
			Commit:      Commit,
			Description: service + description,
			BuildTime:   BuildTime,
		}

		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(res); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
}
