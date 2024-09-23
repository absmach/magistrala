// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"time"

	"github.com/absmach/magistrala/pkg/clients"
)

type Users struct {
	ID          string         `json:"id"`
	Name        string         `json:"name,omitempty"`
	UserName    string         `json:"user_name,omitempty"`
	FirstName   string         `json:"first_name,omitempty"`
	LastName    string         `json:"last_name,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Credentials Credentials    `json:"credentials,omitempty"`
	Metadata    Metadata       `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at,omitempty"`
	UpdatedAt   time.Time      `json:"updated_at,omitempty"`
	UpdatedBy   string         `json:"updated_by,omitempty"`
	Status      clients.Status `json:"status,omitempty"` // 1 for enabled, 0 for disabled
	Role        clients.Role   `json:"role,omitempty"`   // 1 for admin, 0 for normal user
	Permissions []string       `json:"permissions,omitempty"`
}

type Credentials struct {
	Identity string `json:"identity,omitempty"` // username or generated login ID
	Secret   string `json:"secret,omitempty"`   // password or token
}

// Metadata represents arbitrary JSON.
type Metadata map[string]interface{}
