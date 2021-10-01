package commands

import (
	"context"
)

type Metadata map[string]interface{}

type Command struct {
	ID          string
	Owner       string
	Name        string
	Command     string
	ChannelID   string
	ExecuteTime string
	Metadata    Metadata
}

type CommandRepository interface {
	Save(ctx context.Context, c Command) (string, error)

	Update(ctx context.Context, u Command) error

	RetrieveByID(ctx context.Context, id string) (Command, error)

	// RetrieveAll(ctx context.Context, offset, limit uint64, commandIDs []string, email string, m Metadata) (CommandPage, error)

	Remove(ctx context.Context, owner, id string) error
}
