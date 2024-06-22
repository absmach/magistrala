package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/absmach/magistrala/auth"
	"github.com/google/uuid"
	"go.etcd.io/bbolt"
)

const timeLayout = "2006-01-02 15:04:05.999999999 -0700 MST"

func init() {
	gob.Register(auth.SelectedIDs{})
	gob.Register(auth.AnyIDs{})
}
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
	if err := storePATGob(db, pat); err != nil {
		log.Fatal(err)
	}

	// Retrieve PAT
	retrievedPAT, err := getPATGob(db, pat.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Retrieved PAT: %+v\n", retrievedPAT)
}

func storePATGob(db *bbolt.DB, pat auth.PAT) error {
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
		scopeBytes, err := EncodeScopeToGob(pat.Scope)
		if err != nil {
			return err
		}
		if err := bucket.Put([]byte(fmt.Sprintf("pat:%s:scope", pat.ID)), scopeBytes); err != nil {
			return err
		}

		return nil
	})
}

func getPATGob(db *bbolt.DB, id string) (auth.PAT, error) {
	var pat auth.PAT
	err := db.View(func(tx *bbolt.Tx) (err error) {
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
			pat.IssuedAt, err = time.Parse(timeLayout, strings.Split(string(issuedAt), " m=")[0])
			if err != nil {
				return err
			}
		}

		expiresAt := bucket.Get([]byte(fmt.Sprintf("pat:%s:expires_at", id)))
		if expiresAt != nil {
			pat.ExpiresAt, err = time.Parse(timeLayout, strings.Split(string(expiresAt), " m=")[0])
			if err != nil {
				return err
			}
		}

		// Retrieve scope
		scopeBytes := bucket.Get([]byte(fmt.Sprintf("pat:%s:scope", id)))
		if scopeBytes != nil {
			scope, err := DecodeGobToScope(scopeBytes)
			if err != nil {
				return err
			}
			pat.Scope = scope
		}

		return nil
	})
	return pat, err
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
