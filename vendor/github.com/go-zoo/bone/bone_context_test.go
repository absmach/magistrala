// +build go1.7

package bone

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRoutingVariableWithContext(t *testing.T) {
	var (
		expected = "variable"
		got      string
		mux      = New()
		w        = httptest.NewRecorder()
	)

	appFn := func(w http.ResponseWriter, r *http.Request) {
		got = GetValue(r, "vartest")
	}

	middlewareFn := func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), "key", "customValue")
		newReq := r.WithContext(ctx)
		appFn(w, newReq)
	}

	mux.Get("/:vartest", http.HandlerFunc(middlewareFn))
	r, err := http.NewRequest("GET", fmt.Sprintf("/%s", expected), nil)
	if err != nil {
		t.Fatal(err)
	}
	mux.ServeHTTP(w, r)

	if got != expected {
		t.Fatalf("expected %s, got %s", expected, got)
	}
}
