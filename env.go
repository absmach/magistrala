// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mainflux

import (
	"os"

	"github.com/subosito/gotenv"
)

// Env reads specified environment variable. If no value has been found,
// fallback is returned.
func Env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}

// LoadEnvFile loads environment variables defined in an .env formatted file.
func LoadEnvFile(envfilepath string) error {
	err := gotenv.Load(envfilepath)
	return err
}
