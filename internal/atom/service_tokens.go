// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const DefaultServiceEntityID = "00000000-0000-0000-0000-000000000003"

type ServiceTokenSpec struct {
	Name        string
	Env         string
	Description string
}

type TokenProvisionOptions struct {
	OutputPath      string
	ServiceEntityID string
	Rotate          string
	Specs           []ServiceTokenSpec
}

type TokenProvisionResult struct {
	OutputPath string
	Preserved  []string
	Created    []string
	Rotated    []string
}

func DefaultServiceTokenSpecs() []ServiceTokenSpec {
	return []ServiceTokenSpec{
		{Name: "fluxmq-auth", Env: "MG_ATOM_TOKEN_FLUXMQ_AUTH", Description: "Magistrala Docker Compose token for fluxmq-auth"},
		{Name: "fluxmq-node1", Env: "MG_ATOM_TOKEN_FLUXMQ_NODE1", Description: "Magistrala Docker Compose token for fluxmq-node1"},
		{Name: "fluxmq-node2", Env: "MG_ATOM_TOKEN_FLUXMQ_NODE2", Description: "Magistrala Docker Compose token for fluxmq-node2"},
		{Name: "fluxmq-node3", Env: "MG_ATOM_TOKEN_FLUXMQ_NODE3", Description: "Magistrala Docker Compose token for fluxmq-node3"},
		{Name: atomServiceTokenJournal, Env: "MG_ATOM_TOKEN_JOURNAL", Description: "Magistrala Docker Compose token for journal"},
		{Name: "notifications", Env: "MG_ATOM_TOKEN_NOTIFICATIONS", Description: "Magistrala Docker Compose token for notifications"},
		{Name: "timescale-reader", Env: "MG_ATOM_TOKEN_TIMESCALE_READER", Description: "Magistrala Docker Compose token for timescale-reader"},
		{Name: "re", Env: "MG_ATOM_TOKEN_RE", Description: "Magistrala Docker Compose token for rule engine"},
		{Name: "alarms", Env: "MG_ATOM_TOKEN_ALARMS", Description: "Magistrala Docker Compose token for alarms"},
		{Name: "reports", Env: "MG_ATOM_TOKEN_REPORTS", Description: "Magistrala Docker Compose token for reports"},
		{Name: "postgres-reader", Env: "MG_ATOM_TOKEN_POSTGRES_READER", Description: "Magistrala Docker Compose token for postgres-reader"},
	}
}

func ProvisionServiceTokens(ctx context.Context, client *Client, opts TokenProvisionOptions) (TokenProvisionResult, error) {
	if client == nil {
		return TokenProvisionResult{}, fmt.Errorf("atom client is nil")
	}
	if strings.TrimSpace(opts.OutputPath) == "" {
		return TokenProvisionResult{}, fmt.Errorf("token output path is required")
	}
	entityID := strings.TrimSpace(opts.ServiceEntityID)
	if entityID == "" {
		entityID = DefaultServiceEntityID
	}
	specs := opts.Specs
	if len(specs) == 0 {
		specs = DefaultServiceTokenSpecs()
	}
	rotate, err := normalizeRotation(opts.Rotate, specs)
	if err != nil {
		return TokenProvisionResult{}, err
	}

	existing, err := readTokenEnvFile(opts.OutputPath)
	if err != nil {
		return TokenProvisionResult{}, err
	}

	values := make(map[string]string, len(specs))
	result := TokenProvisionResult{OutputPath: opts.OutputPath}
	for _, spec := range specs {
		token := strings.TrimSpace(existing[spec.Env])
		shouldRotate := rotate["all"] || rotate[spec.Env]
		if token != "" && !shouldRotate {
			active, err := client.TokenActive(ctx, token)
			if err == nil && active {
				values[spec.Env] = token
				result.Preserved = append(result.Preserved, spec.Env)
				continue
			}
		}
		if token != "" && shouldRotate {
			credentialID, ok := CredentialIDFromAccessToken(token)
			if ok {
				if err := client.RevokeCredential(ctx, entityID, credentialID); err != nil && !IsNotFound(err) {
					return TokenProvisionResult{}, fmt.Errorf("revoke %s credential %s: %w", spec.Env, credentialID, err)
				}
			}
		}
		created, err := client.CreateUnscopedAccessToken(ctx, entityID, spec.Name, spec.Description)
		if err != nil {
			return TokenProvisionResult{}, fmt.Errorf("create %s token: %w", spec.Env, err)
		}
		if strings.TrimSpace(created.Token) == "" {
			return TokenProvisionResult{}, fmt.Errorf("create %s token: atom returned an empty token", spec.Env)
		}
		values[spec.Env] = created.Token
		if shouldRotate {
			result.Rotated = append(result.Rotated, spec.Env)
		} else {
			result.Created = append(result.Created, spec.Env)
		}
	}

	if err := writeTokenEnvFile(opts.OutputPath, specs, values); err != nil {
		return TokenProvisionResult{}, err
	}
	return result, nil
}

func (c *Client) TokenActive(ctx context.Context, token string) (bool, error) {
	res, err := c.Introspect(ctx, token)
	if err != nil {
		return false, err
	}
	return res.Active, nil
}

func CredentialIDFromAccessToken(token string) (string, bool) {
	rest, ok := strings.CutPrefix(strings.TrimSpace(token), "atom_")
	if !ok {
		return "", false
	}
	idHex, secretHex, ok := strings.Cut(rest, "_")
	if !ok || len(idHex) != 32 || len(secretHex) != 64 || !isLowerHex(idHex) || !isLowerHex(secretHex) {
		return "", false
	}
	return fmt.Sprintf("%s-%s-%s-%s-%s", idHex[0:8], idHex[8:12], idHex[12:16], idHex[16:20], idHex[20:32]), true
}

func normalizeRotation(raw string, specs []ServiceTokenSpec) (map[string]bool, error) {
	rotation := map[string]bool{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return rotation, nil
	}
	if strings.EqualFold(raw, "all") {
		rotation["all"] = true
		return rotation, nil
	}
	lookup := map[string]string{}
	for _, spec := range specs {
		lookup[strings.ToLower(spec.Env)] = spec.Env
		lookup[strings.ToLower(strings.TrimPrefix(spec.Env, "MG_ATOM_TOKEN_"))] = spec.Env
		lookup[strings.ToLower(strings.ReplaceAll(spec.Name, "-", "_"))] = spec.Env
	}
	key := strings.ToLower(strings.ReplaceAll(raw, "-", "_"))
	env, ok := lookup[key]
	if !ok {
		return nil, fmt.Errorf("unknown token rotation target %q", raw)
	}
	rotation[env] = true
	return rotation, nil
}

func readTokenEnvFile(path string) (map[string]string, error) {
	values := map[string]string{}
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return values, nil
		}
		return nil, fmt.Errorf("read token env file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key != "" {
			values[key] = value
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan token env file: %w", err)
	}
	return values, nil
}

func writeTokenEnvFile(path string, specs []ServiceTokenSpec, values map[string]string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create token env directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".env.tokens-*")
	if err != nil {
		return fmt.Errorf("create token env temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("secure token env temp file: %w", err)
	}
	if _, err := fmt.Fprintln(tmp, "# Generated by atom-bootstrap provision-tokens. Do not commit."); err != nil {
		_ = tmp.Close()
		return err
	}
	for _, spec := range specs {
		value := strings.TrimSpace(values[spec.Env])
		if value == "" {
			_ = tmp.Close()
			return fmt.Errorf("missing generated token for %s", spec.Env)
		}
		if _, err := fmt.Fprintf(tmp, "%s=%s\n", spec.Env, value); err != nil {
			_ = tmp.Close()
			return fmt.Errorf("write token env file: %w", err)
		}
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close token env temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace token env file: %w", err)
	}
	return nil
}

func isLowerHex(value string) bool {
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}
