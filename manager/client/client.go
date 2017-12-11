// Package client provides a manager service client intended for internal
// service communication.
package client

import (
	"errors"
	"net/http"
	"time"

	"github.com/mainflux/mainflux/manager"
)

const timeout = time.Second * 5

// ErrServiceUnreachable indicates that the service instance is not available.
var ErrServiceUnreachable = errors.New("cannot contact manager service")

type managerClient struct {
	url string
}

func (mc managerClient) Authenticate(req *http.Request) (string, error) {
	c := &http.Client{
		Timeout: timeout,
	}

	mgReq, err := http.NewRequest("POST", mc.url+"/identity", nil)
	if err != nil {
		return "", manager.ErrUnauthorizedAccess
	}

	mgReq.Header.Set("Authorization", req.Header.Get("Authorization"))

	res, err := c.Do(mgReq)
	defer res.Body.Close()

	if err != nil {
		return "", ErrServiceUnreachable
	}

	if res.StatusCode != http.StatusOK {
		return "", manager.ErrUnauthorizedAccess
	}

	return res.Header.Get("X-Client-Id"), nil
}

// NewClient instantiates the manager service client given its base URL.
func NewClient(url string) managerClient {
	return managerClient{url}
}
