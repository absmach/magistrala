
package grpc_test

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	svc = newService(map[string]string{token: email})
	startGRPCServer(svc, port)

	code := m.Run()

	os.Exit(code)
}
