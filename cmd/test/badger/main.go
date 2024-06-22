package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/absmach/magistrala/auth"
	badger "github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"
)

const timeLayout = "2006-01-02 15:04:05.999999999 -0700 MST"

func main() {
	// Open Badger database
	db, err := badger.Open(badger.DefaultOptions("./badger"))
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

func storePAT(db *badger.DB, pat auth.PAT) error {
	return db.Update(func(txn *badger.Txn) error {
		if err := txn.Set([]byte(fmt.Sprintf("pat:%s:id", pat.ID)), []byte(pat.ID)); err != nil {
			return err
		}
		if err := txn.Set([]byte(fmt.Sprintf("pat:%s:user", pat.ID)), []byte(pat.User)); err != nil {
			return err
		}
		if err := txn.Set([]byte(fmt.Sprintf("pat:%s:name", pat.ID)), []byte(pat.Name)); err != nil {
			return err
		}
		if err := txn.Set([]byte(fmt.Sprintf("pat:%s:issued_at", pat.ID)), []byte(pat.IssuedAt.String())); err != nil {
			return err
		}
		if err := txn.Set([]byte(fmt.Sprintf("pat:%s:expires_at", pat.ID)), []byte(pat.ExpiresAt.String())); err != nil {
			return err
		}
		// Store scope
		scopeKeyPrefix := fmt.Sprintf("pat:%s:scope", pat.ID)
		if err := storeOperationScope(txn, scopeKeyPrefix, pat.Scope.Users); err != nil {
			return err
		}
		return nil
	})
}

func storeOperationScope(txn *badger.Txn, keyPrefix string, os auth.OperationScope) error {
	for opType, scopeValue := range os.Operations {
		scopeKey := fmt.Sprintf("%s:%d", keyPrefix, opType)
		switch v := scopeValue.(type) {
		case auth.AnyIDs:
			if err := txn.Set([]byte(scopeKey), []byte("*")); err != nil {
				return err
			}
		case auth.SelectedIDs:
			for id := range v {
				if err := txn.Set([]byte(fmt.Sprintf("%s:%s", scopeKey, id)), []byte(id)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func getPAT(db *badger.DB, id string) (auth.PAT, error) {
	var pat auth.PAT
	err := db.View(func(txn *badger.Txn) error {
		patID, err := txn.Get([]byte(fmt.Sprintf("pat:%s:id", id)))
		if err != nil {
			return err
		}
		err = patID.Value(func(val []byte) error {
			pat.ID = string(val)
			return nil
		})
		if err != nil {
			return err
		}

		user, err := txn.Get([]byte(fmt.Sprintf("pat:%s:user", id)))
		if err != nil {
			return err
		}
		err = user.Value(func(val []byte) error {
			pat.User = string(val)
			return nil
		})
		if err != nil {
			return err
		}

		name, err := txn.Get([]byte(fmt.Sprintf("pat:%s:name", id)))
		if err != nil {
			return err
		}
		err = name.Value(func(val []byte) error {
			pat.Name = string(val)
			return nil
		})
		if err != nil {
			return err
		}

		issuedAt, err := txn.Get([]byte(fmt.Sprintf("pat:%s:issued_at", id)))
		if err != nil {
			return err
		}
		err = issuedAt.Value(func(val []byte) error {
			pat.IssuedAt, err = time.Parse(timeLayout, strings.Split(string(val), " m=")[0])
			return err
		})
		if err != nil {
			return err
		}

		expiresAt, err := txn.Get([]byte(fmt.Sprintf("pat:%s:expires_at", id)))
		if err != nil {
			return err
		}
		err = expiresAt.Value(func(val []byte) error {
			pat.ExpiresAt, err = time.Parse(timeLayout, strings.Split(string(val), " m=")[0])
			return err
		})
		if err != nil {
			return err
		}

		// Retrieve scope
		scopeKeyPrefix := fmt.Sprintf("pat:%s:scope", id)
		scope, err := getOperationScope(txn, scopeKeyPrefix)
		if err != nil {
			return err
		}
		pat.Scope.Users = scope

		return nil
	})
	return pat, err
}

func getOperationScope(txn *badger.Txn, keyPrefix string) (auth.OperationScope, error) {
	os := auth.OperationScope{Operations: make(map[auth.OperationType]auth.ScopeValue)}
	opt := badger.DefaultIteratorOptions
	opt.Prefix = []byte(keyPrefix)
	it := txn.NewIterator(opt)
	defer it.Close()

	for it.Rewind(); it.Valid(); it.Next() {
		item := it.Item()
		k := item.Key()
		opTypeStr := string(k[len(keyPrefix)+1:])
		var opType auth.OperationType
		_, err := fmt.Sscanf(opTypeStr, "%d", &opType)
		if err != nil {
			return os, err
		}

		item.Value(func(val []byte) error {
			if string(val) == "*" {
				os.Operations[opType] = auth.AnyIDs{}
			} else {
				if os.Operations[opType] == nil {
					os.Operations[opType] = auth.SelectedIDs{}
				}
				os.Operations[opType].(auth.SelectedIDs)[string(val)] = struct{}{}
			}
			return nil
		})
	}

	return os, nil
}
