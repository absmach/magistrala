// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"encoding/json"

	"github.com/absmach/supermq/pkg/messaging"
	mgjson "github.com/absmach/supermq/pkg/transformers/json"
	"github.com/absmach/supermq/pkg/transformers/senml"
	"github.com/vadv/gopher-lua-libs/argparse"
	"github.com/vadv/gopher-lua-libs/base64"
	"github.com/vadv/gopher-lua-libs/crypto"
	"github.com/vadv/gopher-lua-libs/db"
	"github.com/vadv/gopher-lua-libs/filepath"
	"github.com/vadv/gopher-lua-libs/ioutil"
	luajson "github.com/vadv/gopher-lua-libs/json"
	"github.com/vadv/gopher-lua-libs/regexp"
	"github.com/vadv/gopher-lua-libs/storage"
	"github.com/vadv/gopher-lua-libs/strings"
	luatime "github.com/vadv/gopher-lua-libs/time"
	"github.com/vadv/gopher-lua-libs/yaml"
	lua "github.com/yuin/gopher-lua"
)

func preload(l *lua.LState) {
	db.Preload(l)
	ioutil.Preload(l)
	luajson.Preload(l)
	yaml.Preload(l)
	crypto.Preload(l)
	regexp.Preload(l)
	luatime.Preload(l)
	storage.Preload(l)
	base64.Preload(l)
	argparse.Preload(l)
	strings.Preload(l)
	filepath.Preload(l)
}

func traverseJson(l *lua.LState, value interface{}) lua.LValue {
	switch val := value.(type) {
	case string:
		return lua.LString(val)
	case float64:
		return lua.LNumber(val)
	case int:
		return lua.LNumber(float64(val))
	case json.Number:
		if num, err := val.Float64(); err != nil {
			return lua.LNumber(num)
		}
		return lua.LNil
	case bool:
		return lua.LBool(val)
	case []interface{}:
		t := l.NewTable()
		for i, j := range val {
			t.RawSetInt(i+1, traverseJson(l, j))
		}
		return t
	case map[string]interface{}:
		t := l.NewTable()
		for k, v := range val {
			t.RawSetString(k, traverseJson(l, v))
		}
		return t
	case []map[string]interface{}:
		t := l.NewTable()
		for i, j := range val {
			t.RawSetInt(i+1, traverseJson(l, j))
		}
		return t
	default:
		return lua.LNil
	}
}

func prepareMsg(l *lua.LState, message *lua.LTable, m *messaging.Message) {
	l.RawSet(message, lua.LString("channel"), lua.LString(m.Channel))
	l.RawSet(message, lua.LString("subtopic"), lua.LString(m.Subtopic))
	l.RawSet(message, lua.LString("publisher"), lua.LString(m.Publisher))
	l.RawSet(message, lua.LString("protocol"), lua.LString(m.Protocol))
	l.RawSet(message, lua.LString("created"), lua.LNumber(m.Created))

	pld := l.NewTable()
	for i, b := range m.Payload {
		// Lua tables are 1-indexed.
		l.RawSet(pld, lua.LNumber(i+1), lua.LNumber(b))
	}
	l.RawSet(message, lua.LString("payload"), pld)
}

func prepareSenml(l *lua.LState, messages *lua.LTable, m []senml.Message) {
	for i, msg := range m {
		insert := l.NewTable()
		insert.RawSetString("channel", lua.LString(msg.Channel))
		insert.RawSetString("subtopic", lua.LString(msg.Subtopic))
		insert.RawSetString("publisher", lua.LString(msg.Publisher))
		insert.RawSetString("protocol", lua.LString(msg.Protocol))
		insert.RawSetString("name", lua.LString(msg.Name))
		insert.RawSetString("unit", lua.LString(msg.Unit))
		insert.RawSetString("time", lua.LNumber(msg.Time))
		insert.RawSetString("update_time", lua.LNumber(msg.UpdateTime))

		if msg.Value != nil {
			insert.RawSetString("value", lua.LNumber(*msg.Value))
		}
		if msg.StringValue != nil {
			insert.RawSetString("string_value", lua.LString(*msg.StringValue))
		}
		if msg.DataValue != nil {
			insert.RawSetString("data_value", lua.LString(*msg.DataValue))
		}
		if msg.BoolValue != nil {
			insert.RawSetString("bool_value", lua.LBool(*msg.BoolValue))
		}
		if msg.Sum != nil {
			insert.RawSetString("sum", lua.LNumber(*msg.Sum))
		}
		// Lua tables are 1-indexed.
		messages.RawSetInt(i+1, insert)
	}
}

func prepareJson(l *lua.LState, messages *lua.LTable, m mgjson.Messages) {
	for i, msg := range m.Data {
		insert := l.NewTable()
		insert.RawSetString("channel", lua.LString(msg.Channel))
		insert.RawSetString("subtopic", lua.LString(msg.Subtopic))
		insert.RawSetString("publisher", lua.LString(msg.Publisher))
		insert.RawSetString("protocol", lua.LString(msg.Protocol))
		insert.RawSetString("format", lua.LString(m.Format))
		insert.RawSetString("payload", traverseJson(l, map[string]interface{}(msg.Payload)))
		// Lua tables are 1-indexed.
		messages.RawSetInt(i+1, insert)
	}
}
