// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/absmach/supermq/pkg/errors"
)

var ErrInvalidRecurringType = errors.New("invalid recurring type")

const (
	hoursInDay   = 24
	daysInWeek   = 7
	monthsInYear = 12

	protocol = "nats"
)

// ScriptOutput is the indicator for type of the logic
// so we can move it to the Go instead calling Go from Lua.
type ScriptOutput uint

const (
	Channels ScriptOutput = iota
	Alarms
	SaveSenML
	Email
	SaveRemotePg
)

var (
	scriptKindToString = [...]string{"channels", "alarms", "save_senml", "email", "save_remote_pg"}
	stringToScriptKind = map[string]ScriptOutput{
		"channels":       Channels,
		"alarms":         Alarms,
		"save_senml":     SaveSenML,
		"email":          Email,
		"save_remote_pg": SaveRemotePg,
	}
)

func (s ScriptOutput) String() string {
	if int(s) < 0 || int(s) >= len(scriptKindToString) {
		return "unknown"
	}
	return scriptKindToString[s]
}

// MarshalJSON converts ScriptOutput to JSON.
func (s ScriptOutput) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON parses JSON string into ScriptOutput.
func (s *ScriptOutput) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	lower := strings.ToLower(str)
	if val, ok := stringToScriptKind[lower]; ok {
		*s = val
		return nil
	}
	return errors.New("invalid ScriptOutput: " + str)
}

type (
	// ScriptType indicates Runtime type for the future versions
	// that will support JS or Go runtimes alongside Lua.
	ScriptType uint

	Metadata map[string]interface{}

	Script struct {
		Type    ScriptType     `json:"type"`
		Outputs []ScriptOutput `json:"outputs"`
		Value   string         `json:"value"`
	}
)

type Rule struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	DomainID      string    `json:"domain"`
	Metadata      Metadata  `json:"metadata,omitempty"`
	InputChannel  string    `json:"input_channel"`
	InputTopic    string    `json:"input_topic"`
	Logic         Script    `json:"logic"`
	OutputChannel string    `json:"output_channel,omitempty"`
	OutputTopic   string    `json:"output_topic,omitempty"`
	Schedule      Schedule  `json:"schedule,omitempty"`
	Status        Status    `json:"status"`
	CreatedAt     time.Time `json:"created_at,omitempty"`
	CreatedBy     string    `json:"created_by,omitempty"`
	UpdatedAt     time.Time `json:"updated_at,omitempty"`
	UpdatedBy     string    `json:"updated_by,omitempty"`
}
