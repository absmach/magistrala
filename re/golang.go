// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"

	pkglog "github.com/absmach/magistrala/pkg/logger"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/messaging"
	golang "github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

const logicFunction = "main.logicFunction"

// Type message is an SMQ message with payload replaces by JSON deserialized payload.
type message struct {
	Channel   string `json:"channel,omitempty"`
	Domain    string `json:"domain,omitempty"`
	Subtopic  string `json:"subtopic,omitempty"`
	Publisher string `json:"publisher,omitempty"`
	Protocol  string `json:"protocol,omitempty"`
	Created   int64  `json:"created,omitempty"`
	Payload   any    `json:"payload,omitempty"`
}

func (re *re) processGo(ctx context.Context, details []slog.Attr, r Rule, msg *messaging.Message) pkglog.RunInfo {
	i := golang.New(golang.Options{})
	if err := i.Use(stdlib.Symbols); err != nil {
		return pkglog.RunInfo{Level: slog.LevelError, Details: details, Message: err.Error(), Error: err}
	}
	m := message{
		Created:   msg.Created,
		Domain:    msg.Domain,
		Publisher: msg.Publisher,
		Channel:   msg.Channel,
		Subtopic:  msg.Subtopic,
		Protocol:  msg.Protocol,
	}
	var pld any
	if err := json.Unmarshal(msg.Payload, &pld); err != nil {
		pld = msg.Payload
	}
	m.Payload = pld

	err := i.Use(golang.Exports{
		"messaging/m": {
			"message": reflect.ValueOf(m),
		},
	})
	if err != nil {
		return pkglog.RunInfo{Level: slog.LevelError, Details: details, Message: err.Error(), Error: err}
	}
	if _, err = i.Eval(r.Logic.Value); err != nil {
		return pkglog.RunInfo{Level: slog.LevelError, Details: details, Message: err.Error(), Error: err}
	}
	ifc, err := i.Eval(logicFunction)
	if err != nil {
		return pkglog.RunInfo{Level: slog.LevelError, Details: details, Message: err.Error(), Error: err}
	}
	f, ok := ifc.Interface().(func() any)
	if !ok {
		err := fmt.Errorf("invalid logic function signature")
		return pkglog.RunInfo{Level: slog.LevelError, Message: err.Error(), Details: details, Error: err}
	}
	res := f()
	if b, ok := res.(bool); ok && !b {
		return pkglog.RunInfo{Level: slog.LevelInfo, Message: "logic returned false", Details: details}
	}
	for _, o := range r.Outputs {
		if e := re.handleOutput(ctx, o, r, msg, res); e != nil {
			err = errors.Wrap(e, err)
		}
	}
	ret := pkglog.RunInfo{Level: slog.LevelInfo, Details: details, Message: "rule processed successfully"}
	if err != nil {
		ret.Level = slog.LevelError
		ret.Message = fmt.Sprintf("failed to handle rule output: %s", err)
		ret.Error = err
	}
	return ret
}
