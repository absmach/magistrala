// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	atomGraphQLPath       = "/graphql"
	defaultAtomGraphQLURL = "http://localhost:8080" + atomGraphQLPath
	jsonNull              = "null"
)

type graphQLClient struct {
	endpoint   string
	token      string
	httpClient *http.Client
}

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphQLError struct {
	Message string `json:"message"`
}

type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []graphQLError  `json:"errors"`
}

func newGraphQLClient(endpoint, token string, timeout time.Duration) *graphQLClient {
	return &graphQLClient{
		endpoint:   strings.TrimSpace(endpoint),
		token:      strings.TrimSpace(token),
		httpClient: &http.Client{Timeout: timeout},
	}
}

// do runs query and returns the raw JSON value of the requested top-level
// response field. A field that is present but null means Atom holds no such
// object, which is reported as an error so callers never print a bare null.
func (c *graphQLClient) do(ctx context.Context, query string, variables map[string]any, field string) (json.RawMessage, error) {
	if c.endpoint == "" {
		return nil, errors.New("GraphQL endpoint is required")
	}

	body, err := json.Marshal(graphQLRequest{Query: query, Variables: variables})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("GraphQL request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var gqlResp graphQLResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return nil, err
	}
	if len(gqlResp.Errors) > 0 {
		msgs := make([]string, 0, len(gqlResp.Errors))
		for _, gqlErr := range gqlResp.Errors {
			msgs = append(msgs, gqlErr.Message)
		}
		return nil, fmt.Errorf("GraphQL error: %s", strings.Join(msgs, "; "))
	}
	if len(gqlResp.Data) == 0 || string(gqlResp.Data) == jsonNull {
		return nil, errors.New("GraphQL response did not contain data")
	}

	var data map[string]json.RawMessage
	if err := json.Unmarshal(gqlResp.Data, &data); err != nil {
		return nil, err
	}
	value, ok := data[field]
	if !ok {
		return nil, fmt.Errorf("GraphQL response did not contain field %q", field)
	}
	if len(value) == 0 || string(value) == jsonNull {
		return nil, fmt.Errorf("%s not found", field)
	}

	return value, nil
}
