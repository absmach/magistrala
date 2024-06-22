package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/absmach/magistrala/auth"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

var ctx = context.Background()

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	// Create a new PAT
	pat := auth.PAT{
		ID:        uuid.New().String(),
		User:      "user123",
		Name:      "Test Token",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	// Set scope
	pat.Scope.Users = auth.OperationScope{
		Operations: map[auth.OperationType]auth.ScopeValue{
			auth.CreateOp: auth.SelectedIDs{"entity1": {}, "entity2": {}},
		},
	}

	if err := StorePAT(rdb, pat); err != nil {
		panic(err)
	}
	rPAT, err := RetrievePAT(rdb, pat.ID)
	if err != nil {
		panic(err)
	}

	fmt.Println(rPAT.String())
}

func StorePAT(client *redis.Client, pat auth.PAT) error {
	// Create a hash to store PAT fields
	key := "pat:" + pat.ID

	// Convert time.Time fields to Unix timestamps
	issuedAt := strconv.FormatInt(pat.IssuedAt.Unix(), 10)
	expiresAt := strconv.FormatInt(pat.ExpiresAt.Unix(), 10)
	updatedAt := strconv.FormatInt(pat.UpdatedAt.Unix(), 10)
	lastUsedAt := strconv.FormatInt(pat.LastUsedAt.Unix(), 10)
	revokedAt := strconv.FormatInt(pat.RevokedAt.Unix(), 10)

	// Store basic PAT fields
	err := client.HMSet(ctx, key, map[string]interface{}{
		"user":         pat.User,
		"name":         pat.Name,
		"issued_at":    issuedAt,
		"expires_at":   expiresAt,
		"updated_at":   updatedAt,
		"last_used_at": lastUsedAt,
		"revoked":      pat.Revoked,
		"revoked_at":   revokedAt,
	}).Err()
	if err != nil {
		return err
	}

	// Store Scope
	err = StoreScope(client, key+":scope", pat.Scope)
	if err != nil {
		return err
	}

	return nil
}

func StoreScope(client *redis.Client, key string, scope auth.Scope) error {
	// Store Users OperationScope
	err := StoreOperationScope(client, key+":users", scope.Users)
	if err != nil {
		return err
	}

	// Store Domains
	for domainID, domainScope := range scope.Domains {
		domainKey := key + ":domains:" + domainID
		err = StoreDomainScope(client, domainKey, domainScope)
		if err != nil {
			return err
		}
	}

	return nil
}

func StoreOperationScope(client *redis.Client, key string, os auth.OperationScope) error {
	for operation, scopeValue := range os.Operations {
		operationKey := key + ":" + operation.String()
		switch value := scopeValue.(type) {
		case auth.AnyIDs:
			err := client.Set(ctx, operationKey, "*", 0).Err()
			if err != nil {
				return err
			}
		case auth.SelectedIDs:
			for id := range value {
				err := client.HSet(ctx, operationKey, id, "").Err()
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func StoreDomainScope(client *redis.Client, key string, ds auth.DomainScope) error {
	// Store DomainManagement OperationScope
	err := StoreOperationScope(client, key+":domain_management", ds.DomainManagement)
	if err != nil {
		return err
	}

	// Store Entities
	for entityType, operationScope := range ds.Entities {
		entityKey := key + ":entities:" + entityType.String()
		err = StoreOperationScope(client, entityKey, operationScope)
		if err != nil {
			return err
		}
	}

	return nil
}

func RetrievePAT(client *redis.Client, patID string) (*auth.PAT, error) {
	key := "pat:" + patID

	fields, err := client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	issuedAt, _ := strconv.ParseInt(fields["issued_at"], 10, 64)
	expiresAt, _ := strconv.ParseInt(fields["expires_at"], 10, 64)
	updatedAt, _ := strconv.ParseInt(fields["updated_at"], 10, 64)
	lastUsedAt, _ := strconv.ParseInt(fields["last_used_at"], 10, 64)
	revokedAt, _ := strconv.ParseInt(fields["revoked_at"], 10, 64)

	pat := &auth.PAT{
		ID:         patID,
		User:       fields["user"],
		Name:       fields["name"],
		IssuedAt:   time.Unix(issuedAt, 0),
		ExpiresAt:  time.Unix(expiresAt, 0),
		UpdatedAt:  time.Unix(updatedAt, 0),
		LastUsedAt: time.Unix(lastUsedAt, 0),
		Revoked:    fields["revoked"] == "true",
		RevokedAt:  time.Unix(revokedAt, 0),
	}

	// Retrieve Scope
	scope, err := RetrieveScope(client, key+":scope")
	if err != nil {
		return nil, err
	}
	pat.Scope = *scope

	return pat, nil
}

func RetrieveScope(client *redis.Client, key string) (*auth.Scope, error) {
	scope := &auth.Scope{}

	// Retrieve Users OperationScope
	users, err := RetrieveOperationScope(client, key+":users")
	if err != nil {
		return nil, err
	}
	scope.Users = *users

	// Retrieve Domains
	domainKeys, err := client.Keys(ctx, key+":domains:*").Result()
	if err != nil {
		return nil, err
	}

	scope.Domains = make(map[string]auth.DomainScope)
	for _, domainKey := range domainKeys {
		domainID := domainKey[len(key+":domains:"):]
		domainScope, err := RetrieveDomainScope(client, domainKey)
		if err != nil {
			return nil, err
		}
		scope.Domains[domainID] = *domainScope
	}

	return scope, nil
}

func RetrieveOperationScope(client *redis.Client, key string) (*auth.OperationScope, error) {
	os := &auth.OperationScope{
		Operations: make(map[auth.OperationType]auth.ScopeValue),
	}

	operationKeys, err := client.Keys(ctx, key+":*").Result()
	if err != nil {
		return nil, err
	}

	for _, operationKey := range operationKeys {
		operationStr := operationKey[len(key+":"):]
		operation, err := auth.ParseOperationType(operationStr) // You'll need to implement this function to convert string to OperationType
		if err != nil {
			return nil, err
		}

		if wildcard, err := client.Get(ctx, operationKey).Result(); err == nil && wildcard == "*" {
			os.Operations[operation] = auth.AnyIDs{}
		} else {
			ids, err := client.HKeys(ctx, operationKey).Result()
			if err != nil {
				return nil, err
			}

			selectedIDs := auth.SelectedIDs{}
			for _, id := range ids {
				selectedIDs[id] = struct{}{}
			}
			os.Operations[operation] = selectedIDs
		}
	}

	return os, nil
}

func RetrieveDomainScope(client *redis.Client, key string) (*auth.DomainScope, error) {
	ds := &auth.DomainScope{
		Entities: make(map[auth.DomainEntityType]auth.OperationScope),
	}

	// Retrieve DomainManagement OperationScope
	domainManagement, err := RetrieveOperationScope(client, key+":domain_management")
	if err != nil {
		return nil, err
	}
	ds.DomainManagement = *domainManagement

	// Retrieve Entities
	entityKeys, err := client.Keys(ctx, key+":entities:*").Result()
	if err != nil {
		return nil, err
	}

	for _, entityKey := range entityKeys {
		entityTypeStr := entityKey[len(key+":entities:")]
		entityType, err := auth.ParseDomainEntityType(string(entityTypeStr)) // You'll need to implement this function to convert string to DomainEntityType
		if err != nil {
			return nil, err
		}

		operationScope, err := RetrieveOperationScope(client, entityKey)
		if err != nil {
			return nil, err
		}
		ds.Entities[entityType] = *operationScope
	}

	return ds, nil
}
