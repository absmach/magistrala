// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package outputs

import (
	"encoding/json"
	"strings"

	"github.com/absmach/supermq/pkg/errors"
)

const (
	MsgKey       = "message"
	LogicRespKey = "result"
)

const Protocol = "nats"

// OutputType is the indicator for type of the output
// so we can move it to the Go instead calling Go from Lua.
type OutputType uint

const (
	ChannelsType OutputType = iota
	AlarmsType
	SaveSenMLType
	EmailType
	SaveRemotePgType
)

var (
	scriptKindToString = [...]string{"channels", "alarms", "save_senml", "email", "save_remote_pg"}
	stringToScriptKind = map[string]OutputType{
		"channels":       ChannelsType,
		"alarms":         AlarmsType,
		"save_senml":     SaveSenMLType,
		"email":          EmailType,
		"save_remote_pg": SaveRemotePgType,
	}
)

func (s OutputType) String() string {
	if int(s) < 0 || int(s) >= len(scriptKindToString) {
		return "unknown"
	}
	return scriptKindToString[s]
}

// MarshalJSON converts OutputType to JSON.
func (s OutputType) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON parses JSON string into OutputType.
func (s *OutputType) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	lower := strings.ToLower(str)
	if val, ok := stringToScriptKind[lower]; ok {
		*s = val
		return nil
	}
	return errors.New("invalid OutputType: " + str)
}
