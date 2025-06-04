// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"encoding/json"

	"github.com/absmach/supermq/pkg/messaging"
	"github.com/vadv/gopher-lua-libs/argparse"
	"github.com/vadv/gopher-lua-libs/base64"
	bit "github.com/vadv/gopher-lua-libs/bit"
	"github.com/vadv/gopher-lua-libs/crypto"
	"github.com/vadv/gopher-lua-libs/db"
	"github.com/vadv/gopher-lua-libs/filepath"
	client "github.com/vadv/gopher-lua-libs/http/client"
	"github.com/vadv/gopher-lua-libs/ioutil"
	luajson "github.com/vadv/gopher-lua-libs/json"
	"github.com/vadv/gopher-lua-libs/regexp"
	"github.com/vadv/gopher-lua-libs/storage"
	"github.com/vadv/gopher-lua-libs/strings"
	luatime "github.com/vadv/gopher-lua-libs/time"
	"github.com/vadv/gopher-lua-libs/yaml"
	lua "github.com/yuin/gopher-lua"
)

const payloadKey = "payload"

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
	client.Preload(l)
	bit.Preload(l)
}

func prepareMsg(l *lua.LState, msg *messaging.Message) lua.LValue {
	message := l.NewTable()
	message.RawSetString("domain", lua.LString(msg.Domain))
	message.RawSetString("channel", lua.LString(msg.Channel))
	message.RawSetString("subtopic", lua.LString(msg.Subtopic))
	message.RawSetString("publisher", lua.LString(msg.Publisher))
	message.RawSetString("protocol", lua.LString(msg.Protocol))
	message.RawSetString("created", lua.LNumber(msg.Created))

	var payload interface{}
	if err := json.Unmarshal(msg.GetPayload(), &payload); err != nil {
		pld := l.NewTable()
		// If message is not JSON, set binary payload and exit.
		for i, b := range msg.Payload {
			// Lua tables are 1-indexed.
			pld.Insert(i+1, lua.LNumber(b))
		}
		message.RawSetString(payloadKey, pld)
		return message
	}

	// Payload is JSON, set the correct value.
	message.RawSetString(payloadKey, traverseJson(l, payload))
	return message
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
	default:
		return lua.LNil
	}
}

func convertLua(lv lua.LValue) interface{} {
	switch v := lv.(type) {
	case *lua.LTable:
		isArray := true
		v.ForEach(func(key, value lua.LValue) {
			if key.Type() != lua.LTNumber {
				isArray = false
			}
		})

		if isArray {
			arr := []interface{}{}
			v.ForEach(func(key, value lua.LValue) {
				arr = append(arr, convertLua(value))
			})
			return arr
		}

		obj := map[string]interface{}{}
		v.ForEach(func(key, value lua.LValue) {
			obj[key.String()] = convertLua(value)
		})
		return obj
	case lua.LString:
		return string(v)
	case lua.LNumber:
		return float64(v)
	case lua.LBool:
		return bool(v)
	case *lua.LNilType:
		return nil
	default:
		return v.String()
	}
}
