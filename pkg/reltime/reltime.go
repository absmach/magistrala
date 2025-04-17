// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package reltime

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/absmach/magistrala/pkg/errors"
)

var (
	re = regexp.MustCompile(`(?i)^now\(\)([\+\-])(.+)$`)

	ErrInvalidDuration   = errors.New("invalid duration format")
	ErrInvalidExpression = errors.New("invalid time expression")
	ErrUnsupportedUnit   = errors.New("unsupported unit")
)

func Parse(expr string) (time.Time, error) {
	now := time.Now()
	expr = strings.ReplaceAll(expr, " ", "")

	if strings.EqualFold(expr, "now()") {
		return now, nil
	}

	matches := re.FindStringSubmatch(expr)
	if len(matches) != 3 {
		return time.Time{}, errors.Wrap(ErrInvalidExpression, fmt.Errorf("%s", expr))
	}

	sign := matches[1]
	durStr := matches[2]
	if strings.ContainsAny(durStr, "+-") {
		return time.Time{}, errors.Wrap(ErrInvalidExpression, fmt.Errorf("%s", expr))
	}

	dur, err := parseComplexDuration(durStr)
	if err != nil {
		return time.Time{}, err
	}

	if sign == "-" {
		return now.Add(-dur), nil
	}
	return now.Add(dur), nil
}

func parseComplexDuration(s string) (time.Duration, error) {
	var total time.Duration
	re := regexp.MustCompile(`(\d+)([smhdwMY])`)
	matches := re.FindAllStringSubmatch(s, -1)

	if matches == nil {
		return 0, errors.Wrap(ErrInvalidDuration, fmt.Errorf("%s", s))
	}

	for _, match := range matches {
		val, _ := strconv.Atoi(match[1])
		unit := match[2]

		var d time.Duration
		switch unit {
		case "s":
			d = time.Duration(val) * time.Second
		case "m":
			d = time.Duration(val) * time.Minute
		case "h":
			d = time.Duration(val) * time.Hour
		case "d":
			d = time.Duration(val) * 24 * time.Hour
		case "w":
			d = time.Duration(val) * 7 * 24 * time.Hour
		default:
			return 0, errors.Wrap(ErrUnsupportedUnit, fmt.Errorf("%s", unit))
		}

		total += d
	}
	return total, nil
}
