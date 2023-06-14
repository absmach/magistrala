package jwk

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/lestrrat-go/httprc"
)

type Fetcher interface {
	Fetch(context.Context, string, ...FetchOption) (Set, error)
}

type FetchFunc func(context.Context, string, ...FetchOption) (Set, error)

func (f FetchFunc) Fetch(ctx context.Context, u string, options ...FetchOption) (Set, error) {
	return f(ctx, u, options...)
}

var globalFetcher httprc.Fetcher

func init() {
	var nworkers int
	v := os.Getenv(`JWK_FETCHER_WORKER_COUNT`)
	if c, err := strconv.ParseInt(v, 10, 64); err == nil {
		nworkers = int(c)
	}
	if nworkers < 1 {
		nworkers = 3
	}

	globalFetcher = httprc.NewFetcher(context.Background(), httprc.WithFetcherWorkerCount(nworkers))
}

// Fetch fetches a JWK resource specified by a URL. The url must be
// pointing to a resource that is supported by `net/http`.
//
// If you are using the same `jwk.Set` for long periods of time during
// the lifecycle of your program, and would like to periodically refresh the
// contents of the object with the data at the remote resource,
// consider using `jwk.Cache`, which automatically refreshes
// jwk.Set objects asynchronously.
func Fetch(ctx context.Context, u string, options ...FetchOption) (Set, error) {
	var hrfopts []httprc.FetchOption
	var parseOptions []ParseOption
	for _, option := range options {
		if parseOpt, ok := option.(ParseOption); ok {
			parseOptions = append(parseOptions, parseOpt)
			continue
		}

		//nolint:forcetypeassert
		switch option.Ident() {
		case identHTTPClient{}:
			hrfopts = append(hrfopts, httprc.WithHTTPClient(option.Value().(HTTPClient)))
		case identFetchWhitelist{}:
			hrfopts = append(hrfopts, httprc.WithWhitelist(option.Value().(httprc.Whitelist)))
		}
	}

	res, err := globalFetcher.Fetch(ctx, u, hrfopts...)
	if err != nil {
		return nil, fmt.Errorf(`failed to fetch %q: %w`, u, err)
	}

	buf, err := io.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return nil, fmt.Errorf(`failed to read response body for %q: %w`, u, err)
	}

	return Parse(buf, parseOptions...)
}
