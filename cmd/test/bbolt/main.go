package main

import (
	"fmt"
	"log"
	"time"

	"github.com/absmach/magistrala/auth"
	"github.com/google/uuid"
	"go.etcd.io/bbolt"
)

func main() {
	// Open bbolt database
	db, err := bbolt.Open("./bbolt.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

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

	// Store PAT
	if err := storePAT(db, pat); err != nil {
		log.Fatal(err)
	}

	// Retrieve PAT
	retrievedPAT, err := getPAT(db, pat.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Retrieved PAT: %+v\n", retrievedPAT)
}

func storePAT(db *bbolt.DB, pat auth.PAT) error {
	return db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("pats"))
		if err != nil {
			return err
		}

		if err := bucket.Put([]byte(fmt.Sprintf("pat:%s:id", pat.ID)), []byte(pat.ID)); err != nil {
			return err
		}
		if err := bucket.Put([]byte(fmt.Sprintf("pat:%s:user", pat.ID)), []byte(pat.User)); err != nil {
			return err
		}
		if err := bucket.Put([]byte(fmt.Sprintf("pat:%s:name", pat.ID)), []byte(pat.Name)); err != nil {
			return err
		}
		if err := bucket.Put([]byte(fmt.Sprintf("pat:%s:issued_at", pat.ID)), []byte(pat.IssuedAt.String())); err != nil {
			return err
		}
		if err := bucket.Put([]byte(fmt.Sprintf("pat:%s:expires_at", pat.ID)), []byte(pat.ExpiresAt.String())); err != nil {
			return err
		}
		// Store scope
		scopeKeyPrefix := fmt.Sprintf("pat:%s:scope", pat.ID)
		if err := storeOperationScope(bucket, scopeKeyPrefix, pat.Scope.Users); err != nil {
			return err
		}
		return nil
	})
}

func storeOperationScope(bucket *bbolt.Bucket, keyPrefix string, os auth.OperationScope) error {
	for opType, scopeValue := range os.Operations {
		scopeKey := []byte(fmt.Sprintf("%s:%d", keyPrefix, opType))
		switch v := scopeValue.(type) {
		case auth.AnyIDs:
			if err := bucket.Put(scopeKey, []byte("*")); err != nil {
				return err
			}
		case auth.SelectedIDs:
			for id := range v {
				if err := bucket.Put([]byte(fmt.Sprintf("%s:%s", scopeKey, id)), []byte(id)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func getPAT(db *bbolt.DB, id string) (auth.PAT, error) {
	var pat auth.PAT
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("pats"))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		patID := bucket.Get([]byte(fmt.Sprintf("pat:%s:id", id)))
		if patID == nil {
			return fmt.Errorf("pat with ID %s not found", id)
		}
		pat.ID = string(patID)

		user := bucket.Get([]byte(fmt.Sprintf("pat:%s:user", id)))
		if user != nil {
			pat.User = string(user)
		}

		name := bucket.Get([]byte(fmt.Sprintf("pat:%s:name", id)))
		if name != nil {
			pat.Name = string(name)
		}

		issuedAt := bucket.Get([]byte(fmt.Sprintf("pat:%s:issued_at", id)))
		if issuedAt != nil {
			pat.IssuedAt, _ = time.Parse(time.RFC3339, string(issuedAt))
		}

		expiresAt := bucket.Get([]byte(fmt.Sprintf("pat:%s:expires_at", id)))
		if expiresAt != nil {
			pat.ExpiresAt, _ = time.Parse(time.RFC3339, string(expiresAt))
		}

		// Retrieve scope
		scopeKeyPrefix := fmt.Sprintf("pat:%s:scope", id)
		scope, err := getOperationScope(bucket, scopeKeyPrefix)
		if err != nil {
			return err
		}
		pat.Scope.Users = scope

		return nil
	})
	return pat, err
}

func getOperationScope(bucket *bbolt.Bucket, keyPrefix string) (auth.OperationScope, error) {
	os := auth.OperationScope{Operations: make(map[auth.OperationType]auth.ScopeValue)}
	c := bucket.Cursor()
	prefix := []byte(keyPrefix)
	for k, v := c.Seek(prefix); k != nil && len(k) > len(prefix) && string(k[:len(prefix)]) == keyPrefix; k, v = c.Next() {
		opTypeStr := string(k[len(prefix)+1:])
		var opType auth.OperationType
		_, err := fmt.Sscanf(opTypeStr, "%d", &opType)
		if err != nil {
			return os, err
		}

		if string(v) == "*" {
			os.Operations[opType] = auth.AnyIDs{}
		} else {
			if os.Operations[opType] == nil {
				os.Operations[opType] = auth.SelectedIDs{}
			}
			os.Operations[opType].(auth.SelectedIDs)[string(v)] = struct{}{}
		}
	}

	return os, nil
}
