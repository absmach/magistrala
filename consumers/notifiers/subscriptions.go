// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package notifiers

import "context"

// Subscription represents a user Subscription.
type Subscription struct {
	ID      string
	OwnerID string
	Contact string
	Topic   string
}

// Page represents page metadata with content.
type Page struct {
	PageMetadata
	Total         uint
	Subscriptions []Subscription
}

// PageMetadata contains page metadata that helps navigation.
type PageMetadata struct {
	Offset uint
	// Limit values less than 0 indicate no limit.
	Limit   int
	Topic   string
	Contact string
}

// SubscriptionsRepository specifies a Subscription persistence API.
//
//go:generate mockery --name SubscriptionsRepository --output=./mocks --filename repository.go --quiet --note "Copyright (c) Abstract Machines"
type SubscriptionsRepository interface {
	// Save persists a subscription. Successful operation is indicated by non-nil
	// error response.
	Save(ctx context.Context, sub Subscription) (string, error)

	// Retrieve retrieves the subscription for the given id.
	Retrieve(ctx context.Context, id string) (Subscription, error)

	// RetrieveAll retrieves all the subscriptions for the given page metadata.
	RetrieveAll(ctx context.Context, pm PageMetadata) (Page, error)

	// Remove removes the subscription for the given ID.
	Remove(ctx context.Context, id string) error
}
