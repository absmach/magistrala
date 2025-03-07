// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package json

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/absmach/supermq/pkg/errors"
)

var errUnsupportedFormat = errors.New("unsupported time format")

func parseTimestamp(format string, timestamp interface{}, location string) (time.Time, error) {
	switch format {
	case "unix", "unix_ms", "unix_us", "unix_ns":
		return parseUnix(format, timestamp)
	default:
		if location == "" {
			location = "UTC"
		}
		return parseTime(format, timestamp, location)
	}
}

func parseUnix(format string, timestamp interface{}) (time.Time, error) {
	integer, fractional, err := parseComponents(timestamp)
	if err != nil {
		return time.Unix(0, 0), err
	}

	switch strings.ToLower(format) {
	case "unix":
		return time.Unix(integer, fractional).UTC(), nil
	case "unix_ms":
		return time.Unix(0, integer*1e6).UTC(), nil
	case "unix_us":
		return time.Unix(0, integer*1e3).UTC(), nil
	case "unix_ns":
		return time.Unix(0, integer).UTC(), nil
	default:
		return time.Unix(0, 0), errUnsupportedFormat
	}
}

func parseComponents(timestamp interface{}) (int64, int64, error) {
	switch ts := timestamp.(type) {
	case string:
		parts := strings.SplitN(ts, ".", 2)
		if len(parts) == 2 {
			return parseUnixTimeComponents(parts[0], parts[1])
		}

		parts = strings.SplitN(ts, ",", 2)
		if len(parts) == 2 {
			return parseUnixTimeComponents(parts[0], parts[1])
		}

		integer, err := strconv.ParseInt(ts, 10, 64)
		if err != nil {
			return 0, 0, err
		}
		return integer, 0, nil
	case int8:
		return int64(ts), 0, nil
	case int16:
		return int64(ts), 0, nil
	case int32:
		return int64(ts), 0, nil
	case int64:
		return ts, 0, nil
	case uint8:
		return int64(ts), 0, nil
	case uint16:
		return int64(ts), 0, nil
	case uint32:
		return int64(ts), 0, nil
	case uint64:
		return int64(ts), 0, nil
	case float32:
		integer, fractional := math.Modf(float64(ts))
		return int64(integer), int64(fractional * 1e9), nil
	case float64:
		integer, fractional := math.Modf(ts)
		return int64(integer), int64(fractional * 1e9), nil
	default:
		return 0, 0, errUnsupportedFormat
	}
}

func parseUnixTimeComponents(first, second string) (int64, int64, error) {
	integer, err := strconv.ParseInt(first, 10, 64)
	if err != nil {
		return 0, 0, err
	}

	// Convert to nanoseconds, dropping any greater precision.
	buf := []byte("000000000")
	copy(buf, second)

	fractional, err := strconv.ParseInt(string(buf), 10, 64)
	if err != nil {
		return 0, 0, err
	}
	return integer, fractional, nil
}

func parseTime(format string, timestamp interface{}, location string) (time.Time, error) {
	switch ts := timestamp.(type) {
	case string:
		loc, err := time.LoadLocation(location)
		if err != nil {
			return time.Unix(0, 0), err
		}
		switch strings.ToLower(format) {
		case "ansic":
			format = time.ANSIC
		case "unixdate":
			format = time.UnixDate
		case "rubydate":
			format = time.RubyDate
		case "rfc822":
			format = time.RFC822
		case "rfc822z":
			format = time.RFC822Z
		case "rfc850":
			format = time.RFC850
		case "rfc1123":
			format = time.RFC1123
		case "rfc1123z":
			format = time.RFC1123Z
		case "rfc3339":
			format = time.RFC3339
		case "rfc3339nano":
			format = time.RFC3339Nano
		case "stamp":
			format = time.Stamp
		case "stampmilli":
			format = time.StampMilli
		case "stampmicro":
			format = time.StampMicro
		case "stampnano":
			format = time.StampNano
		}
		return time.ParseInLocation(format, ts, loc)
	default:
		return time.Unix(0, 0), errUnsupportedFormat
	}
}
