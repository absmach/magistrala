// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type rootOptions struct {
	graphQLURL string
	token      string
}

// NewRootCmd returns the Atom-backed Magistrala CLI root command.
func NewRootCmd() *cobra.Command {
	opts := &rootOptions{
		graphQLURL: envOrDefault("MG_ATOM_GRAPHQL_URL", defaultAtomGraphQLURL),
		token:      firstEnv("MG_ATOM_TOKEN", "MAGISTRALA_TOKEN"),
	}

	cmd := &cobra.Command{
		Use:   "magistrala-cli",
		Short: "Magistrala command line client",
		Long:  "Magistrala command line client backed by Atom GraphQL.",
	}
	cmd.PersistentFlags().StringVar(&opts.graphQLURL, "graphql-url", opts.graphQLURL, "Atom GraphQL endpoint")
	cmd.PersistentFlags().StringVar(&opts.token, "token", opts.token, "Atom bearer token")

	cmd.AddCommand(
		newLoginCmd(opts),
		newDomainsCmd(opts),
		newGraphQLDevicesCmd(opts),
		newGraphQLChannelsCmd(opts),
		newGraphQLGroupsCmd(opts),
		newAuthzCmd(opts),
	)
	return cmd
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return ""
}

func (o *rootOptions) client() *graphQLClient {
	return newGraphQLClient(o.graphQLURL, o.token)
}

func (o *rootOptions) authedClient() (*graphQLClient, error) {
	if o.token == "" {
		return nil, errors.New("token is required; pass --token or set MG_ATOM_TOKEN")
	}
	return o.client(), nil
}

func writeOutput(cmd *cobra.Command, value any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}

func commonListFlags(cmd *cobra.Command, limit, offset *int) {
	cmd.Flags().IntVar(limit, "limit", 20, "maximum number of records")
	cmd.Flags().IntVar(offset, "offset", 0, "number of records to skip")
}

func parseJSONFlag(raw string) (map[string]any, error) {
	if raw == "" {
		return nil, nil
	}
	var value map[string]any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, fmt.Errorf("invalid JSON value: %w", err)
	}
	return value, nil
}

func optionalString(input string) any {
	if input == "" {
		return nil
	}
	return input
}

func optionalObject(input map[string]any) any {
	if input == nil {
		return nil
	}
	return input
}
