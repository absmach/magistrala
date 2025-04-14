// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"strconv"
	"strings"

	"github.com/absmach/supermq/pkg/messaging"
	lua "github.com/yuin/gopher-lua"
)

type Alarm struct {
	RuleID      string `json:"rule_id"`
	Measurement string `json:"measurement"`
	Value       string `json:"value"`
	Threshold   string `json:"threshold"`
	Unit        string `json:"unit"`
	Cause       string `json:"cause"`
	Severity    uint8  `json:"severity"`
}

func (re *re) sendAlarm(L *lua.LState) int {
	tbl := L.ToTable(1)
	if tbl == nil {
		return 0
	}

	processAlarm := func(alarmTable *lua.LTable) {
		getStr := func(field string) string {
			return alarmTable.RawGetString(field).String()
		}

		severityStr := getStr("severity")
		severity := uint8(1)
		if s, err := strconv.ParseUint(strings.TrimSpace(severityStr), 10, 8); err == nil {
			severity = uint8(s)
		}

		topic := fmt.Sprintf("%s.c.%s.%s", getStr("domain"), getStr("channel"), getStr("subtopic"))

		payload := Alarm{
			RuleID:      getStr("ruleId"),
			Measurement: getStr("measurement"),
			Value:       getStr("value"),
			Threshold:   getStr("threshold"),
			Unit:        getStr("unit"),
			Cause:       getStr("cause"),
			Severity:    severity,
		}

		var buf bytes.Buffer
		if err := gob.NewEncoder(&buf).Encode(payload); err != nil {
			return
		}

		createdNs, err := strconv.ParseInt(getStr("created"), 10, 64)
		if err != nil {
			return
		}

		pubMsg := &messaging.Message{
			Channel:   getStr("channel"),
			Domain:    getStr("domain"),
			Subtopic:  getStr("subtopic"),
			Publisher: getStr("publisher"),
			Protocol:  getStr("protocol"),
			Created:   createdNs,
			Payload:   buf.Bytes(),
		}

		if err := re.alarmsPubSub.Publish(context.Background(), topic, pubMsg); err != nil {
			return
		}
	}

	if tbl.RawGetInt(1) != lua.LNil {
		tbl.ForEach(func(_, value lua.LValue) {
			if alarmTable, ok := value.(*lua.LTable); ok {
				processAlarm(alarmTable)
			}
		})
	} else {
		processAlarm(tbl)
	}

	return 1
}
