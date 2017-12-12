// Package client provides a manager service client intended for internal
// service communication.
package client

import (
	"errors"
	"net/http"
	"time"

	"github.com/mainflux/mainflux/manager"
	"github.com/sony/gobreaker"
)

const (
	timeout         = time.Second * 5
	maxFailedReqs   = 3
	maxFailureRatio = 0.6
)

// ErrServiceUnreachable indicates that the service instance is not available.
var ErrServiceUnreachable = errors.New("manager service unavailable")

type managerClient struct {
	url string
	cb  *gobreaker.CircuitBreaker
}

// NewClient instantiates the manager service client given its base URL.
func NewClient(url string) managerClient {
	st := gobreaker.Settings{
		Name: "Manager",
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			fr := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= maxFailedReqs && fr >= maxFailureRatio
		},
	}

	mc := managerClient{
		url: url,
		cb:  gobreaker.NewCircuitBreaker(st),
	}

	return mc
}

func (mc managerClient) Authenticate(req *http.Request) (string, error) {
	response, err := mc.cb.Execute(func() (interface{}, error) {
		hc := &http.Client{
			Timeout: timeout,
		}

		mgReq, err := http.NewRequest("POST", mc.url+"/identity", nil)
		if err != nil {
			return "", ErrServiceUnreachable
		}

		mgReq.Header.Set("Authorization", req.Header.Get("Authorization"))

		res, err := hc.Do(mgReq)
		defer res.Body.Close()

		if err != nil {
			return "", ErrServiceUnreachable
		}

		if res.StatusCode != http.StatusOK {
			return manager.ErrUnauthorizedAccess, nil
		}

		return res.Header.Get("X-Client-Id"), nil
	})

	if err != nil {
		return "", err
	}

	if key, ok := response.(string); !ok {
		return "", manager.ErrUnauthorizedAccess
	} else {
		return key, nil
	}
}
