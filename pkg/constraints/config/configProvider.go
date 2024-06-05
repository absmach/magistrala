// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"io"
	"os"
	"strconv"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/pelletier/go-toml"
)

const (
	defaultConstraintPath = "./constraints_config.toml"
	filePermission        = 0o644
)

type stringConstraint struct {
	Users        string `toml:"users"`
	Domains      string `toml:"domains"`
	Things       string `toml:"things"`
	Groups       string `toml:"groups"`
	Channels     string `toml:"channels"`
	MsgRateLimit string `toml:"msg_rate_limit"`
}

// Attempts to read constraints from the default path, if the file does not exist,
// it will be created with the default constraints.
func ParseConstraints() (magistrala.Constraints, error) {
	defConstraints := stringConstraint{
		Users:        "5",
		Domains:      "5",
		Things:       "5",
		Groups:       "5",
		Channels:     "5",
		MsgRateLimit: "5",
	}

	_, err := os.Stat(defaultConstraintPath)
	switch {
	case os.IsNotExist(err):
		buf, err := toml.Marshal(defConstraints)
		if err != nil {
			return magistrala.Constraints{}, err
		}
		if err := os.WriteFile(defaultConstraintPath, buf, 0o644); err != nil {
			return magistrala.Constraints{}, err
		}
	case err != nil:
		return magistrala.Constraints{}, err
	}

	c, err := read(defaultConstraintPath)
	if err != nil {
		return magistrala.Constraints{}, err
	}
	return c, nil
}

func read(file string) (c magistrala.Constraints, err error) {
	data, err := os.Open(file)
	if err != nil {
		return c, errors.Wrap(errors.New("failed to read constraints"), err)
	}
	defer data.Close()

	buf, err := io.ReadAll(data)
	if err != nil {
		return c, errors.Wrap(errors.New("failed to read constraints"), err)
	}

	tmp := stringConstraint{}

	if err := toml.Unmarshal(buf, &tmp); err != nil {
		return magistrala.Constraints{}, err
	}

	c.Users, err = convertStringToUint32(tmp.Users)
	if err != nil {
		return c, errors.Wrap(errors.New("failed to read constraints"), err)
	}

	c.Channels, err = convertStringToUint32(tmp.Channels)
	if err != nil {
		return c, errors.Wrap(errors.New("failed to read constraints"), err)
	}

	c.Domains, err = convertStringToUint32(tmp.Domains)
	if err != nil {
		return c, errors.Wrap(errors.New("failed to read constraints"), err)
	}

	c.Groups, err = convertStringToUint32(tmp.Groups)
	if err != nil {
		return c, errors.Wrap(errors.New("failed to read constraints"), err)
	}

	c.Things, err = convertStringToUint32(tmp.Things)
	if err != nil {
		return c, errors.Wrap(errors.New("failed to read constraints"), err)
	}

	c.MsgRateLimit, err = convertStringToUint32(tmp.MsgRateLimit)
	if err != nil {
		return c, errors.Wrap(errors.New("failed to read constraints"), err)
	}

	return c, nil
}

func convertStringToUint32(str string) (uint32, error) {
	parsedValue, err := strconv.ParseUint(str, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(parsedValue), nil
}
