// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package apiutil

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/absmach/magistrala/pkg/errors"
	kithttp "github.com/go-kit/kit/transport/http"
)

// LoggingErrorEncoder is a go-kit error encoder logging decorator.
func LoggingErrorEncoder(logger *slog.Logger, enc kithttp.ErrorEncoder) kithttp.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter) {
		if errors.Contains(err, ErrValidation) {
			logger.Error(err.Error())
		}
		enc(ctx, err, w)
	}
}

// ReadStringQuery reads the value of string http query parameters for a given key.
func ReadStringQuery(r *http.Request, key, def string) (string, error) {
	vals := r.URL.Query()[key]
	if len(vals) > 1 {
		return "", ErrInvalidQueryParams
	}

	if len(vals) == 0 {
		return def, nil
	}

	return vals[0], nil
}

// ReadMetadataQuery reads the value of json http query parameters for a given key.
func ReadMetadataQuery(r *http.Request, key string, def map[string]interface{}) (map[string]interface{}, error) {
	vals := r.URL.Query()[key]
	if len(vals) > 1 {
		return nil, ErrInvalidQueryParams
	}

	if len(vals) == 0 {
		return def, nil
	}

	m := make(map[string]interface{})
	err := json.Unmarshal([]byte(vals[0]), &m)
	if err != nil {
		return nil, errors.Wrap(ErrInvalidQueryParams, err)
	}

	return m, nil
}

// ReadBoolQuery reads boolean query parameters in a given http request.
func ReadBoolQuery(r *http.Request, key string, def bool) (bool, error) {
	vals := r.URL.Query()[key]
	if len(vals) > 1 {
		return false, ErrInvalidQueryParams
	}

	if len(vals) == 0 {
		return def, nil
	}

	b, err := strconv.ParseBool(vals[0])
	if err != nil {
		return false, errors.Wrap(ErrInvalidQueryParams, err)
	}

	return b, nil
}

type number interface {
	int64 | float64 | uint16 | uint64
}

// ReadNumQuery returns a numeric value.
func ReadNumQuery[N number](r *http.Request, key string, def N) (N, error) {
	vals := r.URL.Query()[key]
	if len(vals) > 1 {
		return 0, ErrInvalidQueryParams
	}
	if len(vals) == 0 {
		return def, nil
	}
	val := vals[0]

	switch any(def).(type) {
	case int64:
		v, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return 0, errors.Wrap(ErrInvalidQueryParams, err)
		}
		return N(v), nil
	case uint64:
		v, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return 0, errors.Wrap(ErrInvalidQueryParams, err)
		}
		return N(v), nil
	case uint16:
		v, err := strconv.ParseUint(val, 10, 16)
		if err != nil {
			return 0, errors.Wrap(ErrInvalidQueryParams, err)
		}
		return N(v), nil
	case float64:
		v, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, errors.Wrap(ErrInvalidQueryParams, err)
		}
		return N(v), nil
	default:
		return def, nil
	}
}
