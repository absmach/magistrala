// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package bolt contains PAT repository implementations using
// bolt as the underlying database.
package bolt

import (
	"github.com/absmach/magistrala/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

var errInit = errors.New("failed to initialize BoltDB")

func Init(tx *bolt.Tx, bucket string) error {
	_, err := tx.CreateBucketIfNotExists([]byte(bucket))
	if err != nil {
		return errors.Wrap(errInit, err)
	}
	return nil
}
