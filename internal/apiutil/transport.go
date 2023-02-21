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
		switch err {
		case ErrBearerToken,
			ErrMissingID,
			ErrBearerKey,
			ErrInvalidAuthKey,
			ErrInvalidIDFormat,
			ErrNameSize,
			ErrLimitSize,
			ErrOffsetSize,
			ErrInvalidOrder,
			ErrInvalidDirection,
			ErrEmptyList,
			ErrMalformedPolicy,
			ErrMissingPolicySub,
			ErrMissingPolicyObj,
			ErrMissingPolicyAct,
			ErrMissingCertData,
			ErrInvalidTopic,
			ErrInvalidContact,
			ErrMissingEmail,
			ErrMissingHost,
			ErrMissingPass,
			ErrMissingConfPass,
			ErrInvalidResetPass,
			ErrInvalidComparator,
			ErrMissingMemberType,
			ErrInvalidAPIKey,
			ErrMaxLevelExceeded,
			ErrBootstrapState:
			logger.Error(err.Error())
		}

		enc(ctx, err, w)
	}
}

// ReadUintQuery reads the value of uint64 http query parameters for a given key
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

// ReadIntQuery reads the value of uint64 http query parameters for a given key
func ReadIntQuery(r *http.Request, key string, def int64) (int64, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return 0, errors.ErrInvalidQueryParams
	}

	if len(vals) == 0 {
		return def, nil
	}

	strval := vals[0]
	val, err := strconv.ParseInt(strval, 10, 64)
	if err != nil {
		return 0, errors.ErrInvalidQueryParams
	}

	return val, nil
}

// ReadStringQuery reads the value of string http query parameters for a given key
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

// ReadMetadataQuery reads the value of json http query parameters for a given key
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

// ReadBoolQuery reads boolean query parameters in a given http request
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

// ReadFloatQuery reads the value of float64 http query parameters for a given key
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
