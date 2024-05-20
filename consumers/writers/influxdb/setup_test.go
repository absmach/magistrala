// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package influxdb_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	influxdata "github.com/influxdata/influxdb-client-go/v2"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const (
	dbToken       = "test-token"
	dbOrg         = "test-org"
	dbAdmin       = "test-admin"
	dbPass        = "test-password"
	dbBucket      = "test-bucket"
	dbInitMode    = "setup"
	dbFluxEnabled = "true"
	dbBindAddress = ":8088"
	port          = "8086/tcp"
	db            = "influxdb"
	dbVersion     = "2.7-alpine"
	poolMaxWait   = 120 * time.Second
)

var address string

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		testLog.Error(fmt.Sprintf("Could not connect to docker: %s", err))
	}

	container, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: db,
		Tag:        dbVersion,
		Env: []string{
			fmt.Sprintf("DOCKER_INFLUXDB_INIT_MODE=%s", dbInitMode),
			fmt.Sprintf("DOCKER_INFLUXDB_INIT_USERNAME=%s", dbAdmin),
			fmt.Sprintf("DOCKER_INFLUXDB_INIT_PASSWORD=%s", dbPass),
			fmt.Sprintf("DOCKER_INFLUXDB_INIT_ORG=%s", dbOrg),
			fmt.Sprintf("DOCKER_INFLUXDB_INIT_BUCKET=%s", dbBucket),
			fmt.Sprintf("DOCKER_INFLUXDB_INIT_ADMIN_TOKEN=%s", dbToken),
			fmt.Sprintf("INFLUXDB_HTTP_FLUX_ENABLED=%s", dbFluxEnabled),
			fmt.Sprintf("INFLUXDB_BIND_ADDRESS=%s", dbBindAddress),
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start container: %s", err)
	}
	handleInterrupt(pool, container)

	address = fmt.Sprintf("%s:%s", "http://localhost", container.GetPort(port))
	pool.MaxWait = poolMaxWait

	if err := pool.Retry(func() error {
		client = influxdata.NewClientWithOptions(address, dbToken, influxdata.DefaultOptions())
		_, err = client.Ready(context.Background())
		return err
	}); err != nil {
		testLog.Error(fmt.Sprintf("Could not connect to docker: %s", err))
	}

	code := m.Run()
	if err := pool.Purge(container); err != nil {
		testLog.Error(fmt.Sprintf("Could not purge container: %s", err))
	}

	os.Exit(code)
}

func handleInterrupt(pool *dockertest.Pool, container *dockertest.Resource) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		if err := pool.Purge(container); err != nil {
			log.Fatalf("Could not purge container: %s", err)
		}
		os.Exit(0)
	}()
}
