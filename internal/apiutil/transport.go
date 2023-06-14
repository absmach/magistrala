// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package apiutil

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
)

// LoggingErrorEncoder is a go-kit error encoder logging decorator.
func LoggingErrorEncoder(logger logger.Logger, enc kithttp.ErrorEncoder) kithttp.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter) {
		switch {
		case errors.Contains(err, ErrBearerToken),
			errors.Contains(err, ErrMissingID),
			errors.Contains(err, ErrBearerKey),
			errors.Contains(err, ErrInvalidAuthKey),
			errors.Contains(err, ErrInvalidIDFormat),
			errors.Contains(err, ErrNameSize),
			errors.Contains(err, ErrLimitSize),
			errors.Contains(err, ErrOffsetSize),
			errors.Contains(err, ErrInvalidOrder),
			errors.Contains(err, ErrInvalidDirection),
			errors.Contains(err, ErrEmptyList),
			errors.Contains(err, ErrMalformedPolicy),
			errors.Contains(err, ErrMissingPolicySub),
			errors.Contains(err, ErrMissingPolicyObj),
			errors.Contains(err, ErrMalformedPolicyAct),
			errors.Contains(err, ErrMissingCertData),
			errors.Contains(err, ErrInvalidTopic),
			errors.Contains(err, ErrInvalidContact),
			errors.Contains(err, ErrMissingEmail),
			errors.Contains(err, ErrMissingHost),
			errors.Contains(err, ErrMissingPass),
			errors.Contains(err, ErrMissingConfPass),
			errors.Contains(err, ErrInvalidResetPass),
			errors.Contains(err, ErrInvalidComparator),
			errors.Contains(err, ErrMissingMemberType),
			errors.Contains(err, ErrMaxLevelExceeded),
			errors.Contains(err, ErrInvalidAPIKey),
			errors.Contains(err, ErrInvalidLevel),
			errors.Contains(err, ErrBootstrapState),
			errors.Contains(err, ErrInvalidQueryParams),
			errors.Contains(err, ErrMalformedEntity):
			logger.Error(err.Error())
		}

		enc(ctx, err, w)
	}
}

// ReadUintQuery reads the value of uint64 http query parameters for a given key.
func ReadUintQuery(r *http.Request, key string, def uint64) (uint64, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return 0, errors.ErrInvalidQueryParams
	}

	if len(vals) == 0 {
		return def, nil
	}

	strval := vals[0]
	val, err := strconv.ParseUint(strval, 10, 64)
	if err != nil {
		return 0, errors.ErrInvalidQueryParams
	}

	return val, nil
}

// ReadStringQuery reads the value of string http query parameters for a given key.
func ReadStringQuery(r *http.Request, key string, def string) (string, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return "", errors.ErrInvalidQueryParams
	}

	if len(vals) == 0 {
		return def, nil
	}

	return vals[0], nil
}

// ReadMetadataQuery reads the value of json http query parameters for a given key.
func ReadMetadataQuery(r *http.Request, key string, def map[string]interface{}) (map[string]interface{}, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return nil, errors.ErrInvalidQueryParams
	}

	if len(vals) == 0 {
		return def, nil
	}

	m := make(map[string]interface{})
	err := json.Unmarshal([]byte(vals[0]), &m)
	if err != nil {
		return nil, errors.Wrap(errors.ErrInvalidQueryParams, err)
	}

	return m, nil
}

// ReadBoolQuery reads boolean query parameters in a given http request.
func ReadBoolQuery(r *http.Request, key string, def bool) (bool, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return false, errors.ErrInvalidQueryParams
	}

	if len(vals) == 0 {
		return def, nil
	}

	b, err := strconv.ParseBool(vals[0])
	if err != nil {
		return false, errors.ErrInvalidQueryParams
	}

	return b, nil
}

// ReadFloatQuery reads the value of float64 http query parameters for a given key.
func ReadFloatQuery(r *http.Request, key string, def float64) (float64, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return 0, errors.ErrInvalidQueryParams
	}

	if len(vals) == 0 {
		return def, nil
	}

	fval := vals[0]
	val, err := strconv.ParseFloat(fval, 64)
	if err != nil {
		return 0, errors.ErrInvalidQueryParams
	}

	return val, nil
}

type number interface {
	int64 | float64 | uint16 | uint64
}

// ReadNumQuery returns a numeric value.
func ReadNumQuery[N number](r *http.Request, key string, def N) (N, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return 0, errors.ErrInvalidQueryParams
	}
	if len(vals) == 0 {
		return def, nil
	}
	val := vals[0]

	switch any(def).(type) {
	case int64:
		v, err := strconv.ParseInt(val, 10, 64)
		return N(v), err
	case uint64:
		v, err := strconv.ParseUint(val, 10, 64)
		return N(v), err
	case uint16:
		v, err := strconv.ParseUint(val, 10, 16)
		return N(v), err
	case float64:
		v, err := strconv.ParseFloat(val, 64)
		return N(v), err
	default:
		return def, nil
	}
}
