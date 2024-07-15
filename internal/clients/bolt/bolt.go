// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bolt

import (
	"io/fs"
	"strconv"
	"time"

	"github.com/absmach/magistrala/pkg/errors"
	"github.com/caarlos0/env/v10"
	bolt "go.etcd.io/bbolt"
)

var (
	errConfig  = errors.New("failed to load BoltDB configuration")
	errConnect = errors.New("failed to connect to BoltDB database")
	errInit    = errors.New("failed to initialize to BoltDB database")
)

type FileMode fs.FileMode

func (fm *FileMode) UnmarshalText(text []byte) error {
	temp, err := strconv.ParseUint(string(text), 8, 32)
	if err != nil {
		return err
	}
	*fm = FileMode(temp)
	return nil
}

// Config contains BoltDB specific parameters.
type Config struct {
	FileDirPath string        `env:"FILE_DIR_PATH"  envDefault:"./magistrala-data"`
	FileName    string        `env:"FILE_NAME"      envDefault:"magistrala-pat.db"`
	FileMode    FileMode      `env:"FILE_MODE"      envDefault:"0600"`
	Bucket      string        `env:"BUCKET"         envDefault:"magistrala"`
	Timeout     time.Duration `env:"TIMEOUT"        envDefault:"0"`
}

// Setup load configuration from environment and creates new BoltDB.
func Setup(envPrefix string, initFn func(*bolt.Tx, string) error) (*bolt.DB, error) {
	return SetupDB(envPrefix, initFn)
}

// SetupDB load configuration from environment,.
func SetupDB(envPrefix string, initFn func(*bolt.Tx, string) error) (*bolt.DB, error) {
	cfg := Config{}
	if err := env.ParseWithOptions(&cfg, env.Options{Prefix: envPrefix}); err != nil {
		return nil, errors.Wrap(errConfig, err)
	}
	bdb, err := Connect(cfg, initFn)
	if err != nil {
		return nil, err
	}

	return bdb, nil
}

// Connect establishes connection to the BoltDB.
func Connect(cfg Config, initFn func(*bolt.Tx, string) error) (*bolt.DB, error) {
	filePath := cfg.FileDirPath + "/" + cfg.FileName
	db, err := bolt.Open(filePath, fs.FileMode(cfg.FileMode), nil)
	if err != nil {
		return nil, errors.Wrap(errConnect, err)
	}
	if initFn != nil {
		if err := Init(db, cfg, initFn); err != nil {
			return nil, err
		}
	}
	return db, nil
}

func Init(db *bolt.DB, cfg Config, initFn func(*bolt.Tx, string) error) error {
	if err := db.Update(func(tx *bolt.Tx) error {
		return initFn(tx, cfg.Bucket)
	}); err != nil {
		return errors.Wrap(errInit, err)
	}
	return nil
}
