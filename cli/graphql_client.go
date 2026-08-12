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

const defaultAtomGraphQLURL = "http://localhost:8080/graphql"

type graphQLClient struct {
	endpoint   string
	token      string
	httpClient *http.Client
}

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func newGraphQLClient(endpoint, token string) *graphQLClient {
	return &graphQLClient{
		endpoint: endpoint,
		token:    token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *graphQLClient) do(ctx context.Context, query string, variables map[string]any, out any) error {
	if strings.TrimSpace(c.endpoint) == "" {
		return errors.New("GraphQL endpoint is required")
	}

	body, err := json.Marshal(graphQLRequest{Query: query, Variables: variables})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("GraphQL request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var gqlResp graphQLResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return err
	}
	if len(gqlResp.Errors) > 0 {
		msgs := make([]string, 0, len(gqlResp.Errors))
		for _, gqlErr := range gqlResp.Errors {
			msgs = append(msgs, gqlErr.Message)
		}
		return fmt.Errorf("GraphQL error: %s", strings.Join(msgs, "; "))
	}
	if out == nil {
		return nil
	}
	if len(gqlResp.Data) == 0 || string(gqlResp.Data) == "null" {
		return errors.New("GraphQL response did not contain data")
	}
	return json.Unmarshal(gqlResp.Data, out)
}
