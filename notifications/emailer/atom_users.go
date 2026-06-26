// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package emailer

import (
	"context"
	"fmt"

	"github.com/absmach/magistrala/internal/atom"
)

// AtomUserResolver resolves notification users from Atom entities.
type AtomUserResolver struct {
	client *atom.Client
}

// NewAtomUserResolver creates an Atom-backed notification user resolver.
func NewAtomUserResolver(client *atom.Client) AtomUserResolver {
	return AtomUserResolver{client: client}
}

// FetchUsers loads users by ID from Atom.
func (r AtomUserResolver) FetchUsers(ctx context.Context, userIDs []string) (map[string]User, error) {
	users := make(map[string]User, len(userIDs))
	for _, userID := range userIDs {
		if userID == "" {
			continue
		}
		entity, err := r.client.GetEntity(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("fetch atom entity %s: %w", userID, err)
		}
		users[userID] = atomEntityUser(entity)
	}
	return users, nil
}

func atomEntityUser(entity atom.Entity) User {
	email := attrString(entity.Attributes, "email")
	if email == "" {
		email = attrString(entity.Attributes, "primary_email")
	}
	username := attrString(entity.Attributes, "username")
	if username == "" {
		username = entity.Name
	}
	return User{
		ID:        entity.ID,
		Email:     email,
		Username:  username,
		FirstName: attrString(entity.Attributes, "first_name"),
		LastName:  attrString(entity.Attributes, "last_name"),
	}
}

func attrString(attrs atom.Attributes, key string) string {
	value, ok := attrs[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprint(typed)
	}
}
