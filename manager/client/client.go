// Package client provides a manager service client intended for internal
// service communication.
package client

import (
	"errors"
	"fmt"
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

type ManagerClient struct {
	url string
	cb  *gobreaker.CircuitBreaker
}

// NewClient instantiates the manager service client given its base URL.
func NewClient(url string) ManagerClient {
	st := gobreaker.Settings{
		Name: "Manager",
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			fr := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= maxFailedReqs && fr >= maxFailureRatio
		},
	}

	mc := ManagerClient{
		url: url,
		cb:  gobreaker.NewCircuitBreaker(st),
	}

	return mc
}

func (mc ManagerClient) VerifyToken(token string) (string, error) {
	url := fmt.Sprintf("%s/access-grant", mc.url)
	return mc.makeRequest(url, token)
}

func (mc ManagerClient) CanAccess(channel, token string) (string, error) {
	url := fmt.Sprintf("%s/channels/%s/access-grant", mc.url, channel)
	return mc.makeRequest(url, token)
}

func (mc ManagerClient) makeRequest(url, token string) (string, error) {
	response, err := mc.cb.Execute(func() (interface{}, error) {
		hc := &http.Client{
			Timeout: timeout,
		}

		mgReq, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return "", ErrServiceUnreachable
		}

		mgReq.Header.Set("Authorization", token)

		res, err := hc.Do(mgReq)
		defer res.Body.Close()

		if err != nil {
			return "", ErrServiceUnreachable
		}

		if res.StatusCode != http.StatusOK {
			return manager.ErrUnauthorizedAccess, nil
		}

		return res.Header.Get("X-client-id"), nil
	})

	if err != nil {
		return "", err
	}

	if id, ok := response.(string); !ok {
		return "", manager.ErrUnauthorizedAccess
	} else {
		return id, nil
	}
}
