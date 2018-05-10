package postgres_test

import (
	"fmt"
	"testing"

	"github.com/mainflux/mainflux/clients"
	"github.com/mainflux/mainflux/clients/postgres"
	"github.com/stretchr/testify/assert"
)

func TestClientSave(t *testing.T) {
	email := "client-save@example.com"
	clientRepo := postgres.NewClientRepository(db, testLog)
	client := clients.Client{
		ID:    clientRepo.ID(),
		Owner: email,
	}

	hasErr := clientRepo.Save(client) != nil
	assert.False(t, hasErr, fmt.Sprintf("create new client: expected false got %t\n", hasErr))
}

func TestClientUpdate(t *testing.T) {
	email := "client-update@example.com"

	clientRepo := postgres.NewClientRepository(db, testLog)

	c := clients.Client{
		ID:    clientRepo.ID(),
		Owner: email,
	}

	clientRepo.Save(c)

	cases := map[string]struct {
		client clients.Client
		err    error
	}{
		"existing client":                            {c, nil},
		"non-existing client with existing user":     {clients.Client{ID: wrong, Owner: email}, clients.ErrNotFound},
		"non-existing client with non-existing user": {clients.Client{ID: wrong, Owner: wrong}, clients.ErrNotFound},
	}

	for desc, tc := range cases {
		err := clientRepo.Update(tc.client)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestSingleClientRetrieval(t *testing.T) {
	email := "client-single-retrieval@example.com"

	clientRepo := postgres.NewClientRepository(db, testLog)

	c := clients.Client{
		ID:    clientRepo.ID(),
		Owner: email,
	}

	clientRepo.Save(c)

	cases := map[string]struct {
		owner string
		ID    string
		err   error
	}{
		"existing user":                      {c.Owner, c.ID, nil},
		"existing user, non-existing client": {c.Owner, wrong, clients.ErrNotFound},
		"non-existing owner":                 {wrong, c.ID, clients.ErrNotFound},
	}

	for desc, tc := range cases {
		_, err := clientRepo.One(tc.owner, tc.ID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestMultiClientRetrieval(t *testing.T) {
	email := "client-multi-retrieval@example.com"

	clientRepo := postgres.NewClientRepository(db, testLog)

	n := 10

	for i := 0; i < n; i++ {
		c := clients.Client{
			ID:    clientRepo.ID(),
			Owner: email,
		}

		clientRepo.Save(c)
	}

	cases := map[string]struct {
		owner  string
		offset int
		limit  int
		size   int
	}{
		"existing owner, retrieve all":    {email, 0, n, n},
		"existing owner, retrieve subset": {email, 1, 6, 6},
		"non-existing owner":              {wrong, 1, 6, 0},
	}

	for desc, tc := range cases {
		n := len(clientRepo.All(tc.owner, tc.offset, tc.limit))
		assert.Equal(t, tc.size, n, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, n))
	}
}

func TestClientRemoval(t *testing.T) {
	email := "client-removal@example.com"

	clientRepo := postgres.NewClientRepository(db, testLog)
	client := clients.Client{
		ID:    clientRepo.ID(),
		Owner: email,
	}
	clientRepo.Save(client)

	// show that the removal works the same for both existing and non-existing
	// (removed) client
	for i := 0; i < 2; i++ {
		if err := clientRepo.Remove(email, client.ID); err != nil {
			t.Fatalf("#%d: failed to remove client due to: %s", i, err)
		}

		if _, err := clientRepo.One(email, client.ID); err != clients.ErrNotFound {
			t.Fatalf("#%d: expected %s got %s", i, clients.ErrNotFound, err)
		}
	}
}
