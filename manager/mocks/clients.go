package mocks

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/mainflux/mainflux/manager"
)

var _ manager.ClientRepository = (*clientRepositoryMock)(nil)

type clientRepositoryMock struct {
	mu      sync.Mutex
	counter int
	clients map[string]manager.Client
}

// NewClientRepository creates in-memory client repository.
func NewClientRepository() manager.ClientRepository {
	return &clientRepositoryMock{
		clients: make(map[string]manager.Client),
	}
}

func (repo *clientRepositoryMock) Id() string {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	repo.counter += 1
	return strconv.Itoa(repo.counter)
}

func (repo *clientRepositoryMock) Save(client manager.Client) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	repo.clients[key(client.Owner, client.ID)] = client

	return nil
}

func (repo *clientRepositoryMock) Update(client manager.Client) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	dbKey := key(client.Owner, client.ID)

	if _, ok := repo.clients[dbKey]; !ok {
		return manager.ErrNotFound
	}

	log.Print("c")
	repo.clients[dbKey] = client

	return nil
}

func (repo *clientRepositoryMock) One(owner, id string) (manager.Client, error) {
	if c, ok := repo.clients[key(owner, id)]; ok {
		return c, nil
	}

	return manager.Client{}, manager.ErrNotFound
}

func (repo *clientRepositoryMock) All(owner string) []manager.Client {
	prefix := fmt.Sprintf("%s-", owner)

	clients := make([]manager.Client, 0)

	for k, v := range repo.clients {
		if strings.HasPrefix(k, prefix) {
			clients = append(clients, v)
		}
	}

	return clients
}

func (repo *clientRepositoryMock) Remove(owner, id string) error {
	delete(repo.clients, key(owner, id))
	return nil
}
