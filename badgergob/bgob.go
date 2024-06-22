package badgergob

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/absmach/magistrala/auth"
	badger "github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"
)

const timeLayout = "2006-01-02 15:04:05.999999999 -0700 MST"

func init() {
	gob.Register(auth.SelectedIDs{})
	gob.Register(auth.AnyIDs{})
}
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
	if err := StorePAT(db, pat); err != nil {
		log.Fatal(err)
	}

	// Retrieve PAT
	retrievedPAT, err := GetPAT(db, pat.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Retrieved PAT: %+v\n", retrievedPAT)
}

func StorePAT(db *badger.DB, pat auth.PAT) error {
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
		scopeBytes, err := EncodeScopeToGob(pat.Scope)
		if err != nil {
			return err
		}
		if err := txn.Set([]byte(fmt.Sprintf("pat:%s:scope", pat.ID)), scopeBytes); err != nil {
			return err
		}
		return nil
	})
}

func GetPAT(db *badger.DB, id string) (auth.PAT, error) {
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
		scope, err := txn.Get([]byte(fmt.Sprintf("pat:%s:scope", id)))
		if err != nil {
			return err
		}

		err = scope.Value(func(val []byte) error {
			pat.Scope, err = DecodeGobToScope(val)
			return err
		})
		if err != nil {
			return err
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
