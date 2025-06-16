// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"

	pkglog "github.com/absmach/magistrala/pkg/logger"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/messaging"
	golang "github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

func (re *re) processGo(ctx context.Context, details []slog.Attr, r Rule, msg *messaging.Message) pkglog.RunInfo {
	i := golang.New(golang.Options{})
	if err := i.Use(stdlib.Symbols); err != nil {
		return pkglog.RunInfo{Level: slog.LevelError, Details: details, Message: err.Error()}
	}
	err := i.Use(map[string]map[string]reflect.Value{
		"main": {
			"message": reflect.ValueOf(&msg).Elem(),
		},
	})
	if err != nil {
		return pkglog.RunInfo{Level: slog.LevelError, Details: details, Message: err.Error()}
	}
	res, err := i.Eval(r.Logic.Value)
	if err != nil {
		return pkglog.RunInfo{Level: slog.LevelError, Details: details, Message: err.Error()}
	}
	for _, o := range r.Logic.Outputs {
		if res.Kind() == reflect.Bool && !res.Bool() {
			return pkglog.RunInfo{Level: slog.LevelInfo, Message: "logic returned false", Details: details}
		}
		if e := re.handleOutput(ctx, o, r, msg, res); e != nil {
			err = errors.Wrap(e, err)
		}
	}
	ret := pkglog.RunInfo{Level: slog.LevelInfo, Details: details, Message: "rule processed successfully"}
	if err != nil {
		ret.Level = slog.LevelError
		ret.Message = fmt.Sprintf("failed to handle rule output: %s", err)
	}
	return ret
}
