package badgergob_test

import (
	"log"
	"testing"
	"time"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/badgergob"
	badger "github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"
)

func generateTestPAT() auth.PAT {
	return auth.PAT{
		ID:        uuid.New().String(),
		User:      "user123",
		Name:      "Test Token",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Scope: auth.Scope{
			Users: auth.OperationScope{
				Operations: map[auth.OperationType]auth.ScopeValue{
					auth.CreateOp: auth.SelectedIDs{"entity1": {}, "entity2": {}},
				},
			},
		},
	}
}

func BenchmarkGetPAT(b *testing.B) {
	// Open Badger database
	opts := badger.DefaultOptions("./badger")
	opts.Logger = nil
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Store a PAT to retrieve
	pat := generateTestPAT()
	err = badgergob.StorePAT(db, pat)
	if err != nil {
		log.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := badgergob.GetPAT(db, pat.ID)
		if err != nil {
			log.Fatal(err)
		}
	}
}
