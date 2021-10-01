// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"errors"
	"fmt"
)

var (
	// ErrMalformedEntity indicates malformed entity specification (e.g.
	// invalid username or password).
	ErrMalformedEntity = errors.New("malformed entity specification")

	// ErrUnauthorizedAccess indicates missing or invalid credentials provided
	// when accessing a protected resource.
	ErrUnauthorizedAccess = errors.New("missing or invalid credentials provided")
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	CreateCommand(token string, commands Command) (string, error)
	ViewCommand(token string, id string) (Command, error)
	ListCommands(token string, filter interface{}) ([]Command, error)
	UpdateCommand(token string, commands Command) error
	RemoveCommand(token string, id string) error
}

type commandsService struct {
	repo CommandRepository
}

var _ Service = (*commandsService)(nil)

func New(repo CommandRepository) Service {
	return commandsService{
		repo: repo,
	}
}
func (ks commandsService) CreateCommand(token string, commands Command) (string, error) {
	fmt.Println("Command Created")
	return "", nil
}

func (ks commandsService) ViewCommand(token, id string) (Command, error) {
	fmt.Println("View Command")
	return Command{}, nil
}

func (ks commandsService) ListCommands(token string, filter interface{}) ([]Command, error) {
	fmt.Println("List Command")
	return nil, nil
}

func (ks commandsService) UpdateCommand(token string, command Command) error {
	fmt.Println("Command Updated")
	return nil
}

func (ks commandsService) RemoveCommand(token, id string) error {
	fmt.Println("Command removed")
	return nil
}
