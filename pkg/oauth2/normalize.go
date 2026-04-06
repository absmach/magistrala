// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package oauth2

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/absmach/magistrala/users"
)

type normalizedUser struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Picture   string `json:"picture"`
}

func NormalizeUser(data []byte, provider string) (users.User, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return users.User{}, err
	}

	normalized := normalizeProfile(raw)

	userBytes, err := json.Marshal(normalized)
	if err != nil {
		return users.User{}, err
	}

	var user normalizedUser
	if err := json.Unmarshal(userBytes, &user); err != nil {
		return users.User{}, err
	}

	if err := validateUser(user); err != nil {
		return users.User{}, err
	}

	return users.User{
		ID:             user.ID,
		FirstName:      user.FirstName,
		LastName:       user.LastName,
		Email:          user.Email,
		ProfilePicture: user.Picture,
		Metadata:       users.Metadata{"oauth_provider": provider},
	}, nil
}

func normalizeProfile(raw map[string]any) map[string]any {
	normalized := make(map[string]any)

	keyMap := map[string][]string{
		"id":         {"id"},
		"first_name": {"given_name", "first_name", "givenName", "firstname"},
		"last_name":  {"family_name", "last_name", "familyName", "lastname"},
		"username":   {"username", "user_name", "userName"},
		"email":      {"email", "email_address", "emailAddress"},
		"picture":    {"picture", "profile_picture", "profilePicture", "avatar"},
	}

	for stdKey, variants := range keyMap {
		for _, variant := range variants {
			if val, ok := raw[variant]; ok {
				normalized[stdKey] = val
				break
			}
		}
	}

	return normalized
}

func validateUser(user normalizedUser) error {
	var missing []string
	if user.ID == "" {
		missing = append(missing, "id")
	}
	if user.FirstName == "" {
		missing = append(missing, "first_name")
	}
	if user.LastName == "" {
		missing = append(missing, "last_name")
	}
	if user.Email == "" {
		missing = append(missing, "email")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required fields: %s", strings.Join(missing, ", "))
	}
	return nil
}
