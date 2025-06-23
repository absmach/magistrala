// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"encoding/hex"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	lua "github.com/yuin/gopher-lua"
)

const (
	MsgKey       = "message"
	LogicRespKey = "result"
)

func luaEncrypt(l *lua.LState) int {
	key, iv, data, err := decodeParams(l)
	if err != nil {
		l.Push(lua.LNil)
		l.Push(lua.LString(fmt.Sprintf("failed to decode params: %v", err)))
		return 2
	}

	enc, err := encrypt(key, iv, data)
	if err != nil {
		l.Push(lua.LNil)
		l.Push(lua.LString(fmt.Sprintf("failed to encrypt: %v", err)))
		return 2
	}
	l.Push(lua.LString(hex.EncodeToString(enc)))

	return 1
}

func luaDecrypt(l *lua.LState) int {
	key, iv, data, err := decodeParams(l)
	if err != nil {
		l.Push(lua.LNil)
		l.Push(lua.LString(fmt.Sprintf("failed to decode params: %v", err)))
		return 2
	}

	dec, err := decrypt(key, iv, data)
	if err != nil {
		l.Push(lua.LNil)
		l.Push(lua.LString(fmt.Sprintf("failed to decrypt: %v", err)))
		return 2
	}

	l.Push(lua.LString(hex.EncodeToString(dec)))

	return 1
}

func decodeParams(l *lua.LState) (key, iv, data []byte, err error) {
	keyStr := l.ToString(1)
	ivStr := l.ToString(2)
	dataStr := l.ToString(3)

	key, err = hex.DecodeString(keyStr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode key: %v", err)
	}

	iv, err = hex.DecodeString(ivStr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode IV: %v", err)
	}

	data, err = hex.DecodeString(dataStr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode data: %v", err)
	}

	return key, iv, data, nil
}
