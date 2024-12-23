package re

import "time"

type Kind uint

type Script struct {
	Kind  Kind   `json:"kind"`
	Value []byte `json:"value"`
}

type Schedule struct {
	Dates     []time.Time `json:"date,omitempty"`
	Recurring bool
}

type Rule struct {
	ID           string    `json:"id"`
	InputTopics  []string  `json:"input_topics"`
	Logic        Script    `json:"script"`
	OutputTopics []string  `json:"output_topics,omitempty"`
	Schedule     Schedule  `json:"schedule,omitempty"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
	UpdatedAt    time.Time `json:"updated_at,omitempty"`
	UpdatedBy    string    `json:"updated_by,omitempty"`
}

type Repository interface {
	AddRule(r Rule) (Rule, error)
	ViewRule(id string) (Rule, error)
	UpdateRule(r Rule) (Rule, error)
	RemoveRule(id string) error
	ListRules() ([]Rule, error)
}

type Service interface {
	AddRule(r Rule) (Rule, error)
	ViewRule(id string) (Rule, error)
	UpdateRule(r Rule) (Rule, error)
	ListRules() ([]Rule, error)
	RemoveRule(id string) error
}
