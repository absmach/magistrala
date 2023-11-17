// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cassandra_test

import (
	"fmt"
	"os"
	"testing"

	casclient "github.com/absmach/magistrala/internal/clients/cassandra"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/gocql/gocql"
	"github.com/ory/dockertest/v3"
)

var logger, _ = mglog.New(os.Stdout, mglog.Info.String())

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		logger.Error(fmt.Sprintf("Could not connect to docker: %s", err))
	}

	container, err := pool.Run("cassandra", "3.11.10", []string{})
	if err != nil {
		logger.Error(fmt.Sprintf("Could not start container: %s", err))
	}

	port := container.GetPort("9042/tcp")
	addr = fmt.Sprintf("%s:%s", addr, port)

	if err = pool.Retry(func() error {
		if err := createKeyspace([]string{addr}); err != nil {
			return err
		}

		session, err := casclient.Connect(casclient.Config{
			Hosts:    []string{addr},
			Keyspace: keyspace,
		})
		if err != nil {
			return err
		}
		defer session.Close()

		return nil
	}); err != nil {
		logger.Fatal(fmt.Sprintf("Could not connect to docker: %s", err))
	}

	code := m.Run()

	if err := pool.Purge(container); err != nil {
		logger.Error(fmt.Sprintf("Could not purge container: %s", err))
	}

	os.Exit(code)
}

func createKeyspace(hosts []string) error {
	cluster := gocql.NewCluster(hosts...)
	cluster.Consistency = gocql.Quorum

	session, err := cluster.CreateSession()
	if err != nil {
		return err
	}
	defer session.Close()

	keyspaceCQL := fmt.Sprintf(`CREATE KEYSPACE IF NOT EXISTS %s WITH replication =
                   {'class':'SimpleStrategy','replication_factor':'1'}`, keyspace)

	return session.Query(keyspaceCQL).Exec()
}
