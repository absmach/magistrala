package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"strconv"
	"time"

	"github.com/absmach/magistrala/auth"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

var ctx = context.Background()

func init() {
	gob.Register(auth.SelectedIDs{})
	gob.Register(auth.AnyIDs{})
}
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

	if err := StorePATGob(rdb, pat); err != nil {
		panic(err)
	}
	rPAT, err := RetrievePATGob(rdb, pat.ID)
	if err != nil {
		panic(err)
	}

	fmt.Println(rPAT.String())
}

func StorePATGob(client *redis.Client, pat auth.PAT) error {
	scopeByte, err := EncodeScopeToGob(pat.Scope)
	if err != nil {
		return err
	}
	// Create a hash to store PAT fields
	key := "pat:" + pat.ID

	// Convert time.Time fields to Unix timestamps
	issuedAt := strconv.FormatInt(pat.IssuedAt.Unix(), 10)
	expiresAt := strconv.FormatInt(pat.ExpiresAt.Unix(), 10)
	updatedAt := strconv.FormatInt(pat.UpdatedAt.Unix(), 10)
	lastUsedAt := strconv.FormatInt(pat.LastUsedAt.Unix(), 10)
	revokedAt := strconv.FormatInt(pat.RevokedAt.Unix(), 10)

	// Store basic PAT fields
	err = client.HMSet(ctx, key, map[string]interface{}{
		"user":         pat.User,
		"name":         pat.Name,
		"issued_at":    issuedAt,
		"expires_at":   expiresAt,
		"updated_at":   updatedAt,
		"last_used_at": lastUsedAt,
		"revoked":      pat.Revoked,
		"revoked_at":   revokedAt,
		"scope":        scopeByte,
	}).Err()
	if err != nil {
		return err
	}
	return nil
}

func RetrievePATGob(client *redis.Client, patID string) (*auth.PAT, error) {
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

	// Decode scope from bytes
	scopeBytes := []byte(fields["scope"])

	scope, err := DecodeGobToScope(scopeBytes)
	if err != nil {
		return nil, err
	}

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
		Scope:      scope, // Assign decoded scope to PAT's Scope field
	}

	return pat, nil
}

func EncodeScopeToGob(scope auth.Scope) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(scope); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodeGobToScope(scopeBytes []byte) (auth.Scope, error) {
	buf := bytes.NewBuffer(scopeBytes)
	var scope auth.Scope
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&scope); err != nil {
		return auth.Scope{}, err
	}
	return scope, nil
}
