// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/absmach/magistrala/pkg/atom"
	smqsdk "github.com/absmach/magistrala/pkg/sdk"
	"github.com/spf13/cobra"
)

const (
	defCertsSvcURL    = "http://localhost:9010"
	defHTTPAdapterURL = "http://localhost:9026"
)

type rootOptions struct {
	graphQLURL string
	token      string
	timeout    time.Duration
}

// NewRootCmd returns the Atom-backed Magistrala CLI root command.
func NewRootCmd() *cobra.Command {
	atomCfg := atom.LoadConfig()
	opts := &rootOptions{
		graphQLURL: graphQLURLFromEnv(atomCfg.URL),
		token:      strings.TrimSpace(atomCfg.Token),
		timeout:    atomCfg.Timeout,
	}

	// health is the one command still served by the legacy per-service HTTP
	// APIs rather than by Atom, so it keeps using the Magistrala SDK.
	SetSDK(smqsdk.NewSDK(smqsdk.Config{
		CertsURL:       envOrDefault("MG_CERTS_URL", defCertsSvcURL),
		HTTPAdapterURL: envOrDefault("MG_HTTP_ADAPTER_URL", defHTTPAdapterURL),
	}))

	cmd := &cobra.Command{
		Use:   "magistrala-cli",
		Short: "Magistrala command line client",
		Long:  "Magistrala command line client backed by Atom GraphQL.",
		// devices, gateways and device-types go through pkg/atom.Client, which
		// can only be built once --graphql-url and --token have been parsed.
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			cfg := atomCfg
			cfg.URL = atomBaseURL(opts.graphQLURL, atomCfg.URL)
			cfg.Token = opts.token
			SetAtomClient(atom.NewClient(cfg))
		},
	}
	cmd.PersistentFlags().StringVar(&opts.graphQLURL, "graphql-url", opts.graphQLURL, "Atom GraphQL endpoint")
	cmd.PersistentFlags().StringVar(&opts.token, "token", opts.token, "Atom bearer token")
	cmd.PersistentFlags().BoolVarP(&RawOutput, "raw", "r", RawOutput, "Enables raw output mode for easier parsing of output")
	cmd.PersistentFlags().Uint64VarP(&Limit, varLimit, "l", Limit, "Limit query parameter")
	cmd.PersistentFlags().Uint64VarP(&Offset, varOffset, "o", Offset, "Offset query parameter")
	cmd.PersistentFlags().StringVarP(&Name, varName, "n", Name, "Name query parameter")

	cmd.AddCommand(
		NewHealthCmd(),
		newLoginCmd(opts),
		newWorkspacesCmd(opts),
		newGraphQLChannelsCmd(opts),
		newGraphQLGroupsCmd(opts),
		newAuthzCmd(opts),
		NewDevicesCmd(),
		NewGatewaysCmd(),
		NewDeviceTypesCmd(),
	)

	return cmd
}

// atomBaseURL turns a GraphQL endpoint back into the Atom base URL
// pkg/atom.Client expects, so --graphql-url configures both clients. An
// endpoint that does not end in the GraphQL path leaves the base untouched.
func atomBaseURL(endpoint, fallback string) string {
	endpoint = strings.TrimSpace(endpoint)
	base := strings.TrimSuffix(endpoint, atomGraphQLPath)
	if base != endpoint && base != "" {
		return base
	}

	return fallback
}

// graphQLURLFromEnv resolves the Atom GraphQL endpoint, preferring an explicit
// ATOM_GRAPHQL_URL over the ATOM_URL base the rest of the repo configures.
func graphQLURLFromEnv(atomURL string) string {
	if endpoint := strings.TrimSpace(os.Getenv("ATOM_GRAPHQL_URL")); endpoint != "" {
		return endpoint
	}
	if base := strings.TrimRight(strings.TrimSpace(atomURL), "/"); base != "" {
		return base + atomGraphQLPath
	}

	return defaultAtomGraphQLURL
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}

	return fallback
}

func (o *rootOptions) client() *graphQLClient {
	return newGraphQLClient(o.graphQLURL, o.token, o.timeout)
}

func (o *rootOptions) authedClient() (*graphQLClient, error) {
	if strings.TrimSpace(o.token) == "" {
		return nil, errors.New("token is required; pass --token or set ATOM_ADMIN_TOKEN")
	}

	return o.client(), nil
}

// writeOutput re-indents the raw Atom response, which keeps field order and
// leaves characters such as & and < untouched.
func writeOutput(cmd *cobra.Command, value json.RawMessage) error {
	var buf bytes.Buffer
	if err := json.Indent(&buf, value, "", "  "); err != nil {
		return err
	}
	buf.WriteByte('\n')
	_, err := cmd.OutOrStdout().Write(buf.Bytes())

	return err
}

// listVariables seeds the pagination every list query takes from the shared
// --limit/--offset flags the rest of the CLI already uses.
func listVariables() map[string]any {
	return map[string]any{varLimit: Limit, varOffset: Offset}
}

func parseJSONFlag(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var value map[string]any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, fmt.Errorf("invalid JSON value: %w", err)
	}

	return value, nil
}

// gqlInput builds a GraphQL input object. Empty values are omitted rather than
// sent as an explicit null, which GraphQL reads as "set this field to null".
type gqlInput map[string]any

func (in gqlInput) setString(key, value string) {
	if value != "" {
		in[key] = value
	}
}

func (in gqlInput) setObject(key string, value map[string]any) {
	if value != nil {
		in[key] = value
	}
}
