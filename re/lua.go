// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"encoding/json"

	"github.com/absmach/senml"
	"github.com/absmach/supermq/pkg/messaging"

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

func prepareMsg(l *lua.LState, m *messaging.Message) lua.LValue {
	message := l.NewTable()
	message.RawSetString("domain", lua.LString(m.Domain))
	message.RawSetString("channel", lua.LString(m.Channel))
	message.RawSetString("subtopic", lua.LString(m.Subtopic))
	message.RawSetString("publisher", lua.LString(m.Publisher))
	message.RawSetString("protocol", lua.LString(m.Protocol))
	message.RawSetString("created", lua.LNumber(m.Created))

	var payload interface{}
	pld := l.NewTable()
	if err := json.Unmarshal(m.GetPayload(), &payload); err != nil {
		for i, b := range m.Payload {
			// Lua tables are 1-indexed.
			pld.Insert(i+1, lua.LNumber(b))
		}
		message.RawSetString("payload", pld)
	}

	// Paylad is JSON, set the value.
	payload = traverseJson(l, payload)
	message.RawSetString("payload", pld)
	return message
}

func prepareSenml(l *lua.LState, msg *messaging.Message) (lua.LValue, error) {
	message := l.NewTable()
	pack, err := senml.Decode(msg.Payload, senml.JSON)
	if err != nil {
		return &lua.LNilType{}, err
	}
	message.RawSetString("domain", lua.LString(msg.Domain))
	message.RawSetString("channel", lua.LString(msg.Channel))
	message.RawSetString("subtopic", lua.LString(msg.Subtopic))
	message.RawSetString("publisher", lua.LString(msg.Publisher))
	message.RawSetString("protocol", lua.LString(msg.Protocol))
	message.RawSetString("created", lua.LNumber(msg.Created))
	payload := l.NewTable()

	for i, r := range pack.Records {
		insert := l.NewTable()
		insert.RawSetString("bn", lua.LString(r.BaseName))
		insert.RawSetString("bt", lua.LNumber(r.BaseTime))
		insert.RawSetString("bu", lua.LString(r.BaseUnit))
		insert.RawSetString("bver", lua.LNumber(r.BaseVersion))
		insert.RawSetString("bv", lua.LNumber(r.BaseValue))
		insert.RawSetString("bs", lua.LNumber(r.BaseSum))
		insert.RawSetString("n", lua.LString(r.Name))
		insert.RawSetString("u", lua.LString(r.Unit))
		insert.RawSetString("t", lua.LNumber(r.Time))
		insert.RawSetString("ut", lua.LNumber(r.UpdateTime))
		if r.Value != nil {
			insert.RawSetString("v", lua.LNumber(*r.Value))
		}
		if r.StringValue != nil {
			insert.RawSetString("vs", lua.LString(*r.StringValue))
		}
		if r.DataValue != nil {
			insert.RawSetString("vd", lua.LString(*r.DataValue))
		}
		if r.BoolValue != nil {
			insert.RawSetString("vb", lua.LBool(*r.BoolValue))
		}
		if r.Sum != nil {
			insert.RawSetString("s", lua.LNumber(*r.Sum))
		}
		payload.RawSetInt(i+1, insert)
	}
	message.RawSetString("payload", payload)
	return message, nil
}

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
