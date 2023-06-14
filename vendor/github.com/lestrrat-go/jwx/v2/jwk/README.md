# JWK [![Go Reference](https://pkg.go.dev/badge/github.com/lestrrat-go/jwx/v2/jwk.svg)](https://pkg.go.dev/github.com/lestrrat-go/jwx/v2/jwk)

Package jwk implements JWK as described in [RFC7517](https://tools.ietf.org/html/rfc7517).
If you are looking to use JWT wit JWKs, look no further than [github.com/lestrrat-go/jwx](../jwt).

* Parse and work with RSA/EC/Symmetric/OKP JWK types
  * Convert to and from JSON
  * Convert to and from raw key types (e.g. *rsa.PrivateKey)
* Ability to keep a JWKS fresh using *jwk.AutoRefersh

## Supported key types:

| kty | Curve                   | Go Key Type                                   |
|:----|:------------------------|:----------------------------------------------|
| RSA | N/A                     | rsa.PrivateKey / rsa.PublicKey (2)            |
| EC  | P-256<br>P-384<br>P-521<br>secp256k1 (1) | ecdsa.PrivateKey / ecdsa.PublicKey (2)        |
| oct | N/A                     | []byte                                        |
| OKP | Ed25519 (1)             | ed25519.PrivateKey / ed25519.PublicKey (2)    |
|     | X25519 (1)              | (jwx/)x25519.PrivateKey / x25519.PublicKey (2)|

* Note 1: Experimental
* Note 2: Either value or pointers accepted (e.g. rsa.PrivateKey or *rsa.PrivateKey)

# Documentation

Please read the [API reference](https://pkg.go.dev/github.com/lestrrat-go/jwx/v2/jwk), or
the how-to style documentation on how to use JWK can be found in the [docs directory](../docs/04-jwk.md).

# Auto-Refresh a key during a long running process

<!-- INCLUDE(examples/jwk_cache_example_test.go) -->
```go
package examples_test

import (
  "context"
  "fmt"
  "time"

  "github.com/lestrrat-go/jwx/v2/jwk"
)

func ExampleJWK_Cache() {
  ctx, cancel := context.WithCancel(context.Background())

  const googleCerts = `https://www.googleapis.com/oauth2/v3/certs`

  // First, set up the `jwk.Cache` object. You need to pass it a
  // `context.Context` object to control the lifecycle of the background fetching goroutine.
  //
  // Note that by default refreshes only happen very 15 minutes at the
  // earliest. If you need to control this, use `jwk.WithRefreshWindow()`
  c := jwk.NewCache(ctx)

  // Tell *jwk.Cache that we only want to refresh this JWKS
  // when it needs to (based on Cache-Control or Expires header from
  // the HTTP response). If the calculated minimum refresh interval is less
  // than 15 minutes, don't go refreshing any earlier than 15 minutes.
  c.Register(googleCerts, jwk.WithMinRefreshInterval(15*time.Minute))

  // Refresh the JWKS once before getting into the main loop.
  // This allows you to check if the JWKS is available before we start
  // a long-running program
  _, err := c.Refresh(ctx, googleCerts)
  if err != nil {
    fmt.Printf("failed to refresh google JWKS: %s\n", err)
    return
  }

  // Pretend that this is your program's main loop
MAIN:
  for {
    select {
    case <-ctx.Done():
      break MAIN
    default:
    }
    keyset, err := c.Get(ctx, googleCerts)
    if err != nil {
      fmt.Printf("failed to fetch google JWKS: %s\n", err)
      return
    }
    _ = keyset
    // The returned `keyset` will always be "reasonably" new. It is important that
    // you always call `ar.Fetch()` before using the `keyset` as this is where the refreshing occurs.
    //
    // By "reasonably" we mean that we cannot guarantee that the keys will be refreshed
    // immediately after it has been rotated in the remote source. But it should be close\
    // enough, and should you need to forcefully refresh the token using the `(jwk.Cache).Refresh()` method.
    //
    // If re-fetching the keyset fails, a cached version will be returned from the previous successful
    // fetch upon calling `(jwk.Cache).Fetch()`.

    // Do interesting stuff with the keyset... but here, we just
    // sleep for a bit
    time.Sleep(time.Second)

    // Because we're a dummy program, we just cancel the loop now.
    // If this were a real program, you prosumably loop forever
    cancel()
  }
  // OUTPUT:
}
```
source: [examples/jwk_cache_example_test.go](https://github.com/lestrrat-go/jwx/blob/v2/examples/jwk_cache_example_test.go)
<!-- END INCLUDE -->

Parse and use a JWK key:

<!-- INCLUDE(examples/jwk_example_test.go) -->
```go
package examples_test

import (
  "bytes"
  "context"
  "crypto/ecdsa"
  "crypto/elliptic"
  "fmt"
  "log"

  "github.com/lestrrat-go/jwx/v2/internal/json"
  "github.com/lestrrat-go/jwx/v2/jwk"
)

func ExampleJWK_Usage() {
  // Use jwk.Cache if you intend to keep reuse the JWKS over and over
  set, err := jwk.Fetch(context.Background(), "https://www.googleapis.com/oauth2/v3/certs")
  if err != nil {
    log.Printf("failed to parse JWK: %s", err)
    return
  }

  // Key sets can be serialized back to JSON
  {
    jsonbuf, err := json.Marshal(set)
    if err != nil {
      log.Printf("failed to marshal key set into JSON: %s", err)
      return
    }
    log.Printf("%s", jsonbuf)
  }

  for it := set.Iterate(context.Background()); it.Next(context.Background()); {
    pair := it.Pair()
    key := pair.Value.(jwk.Key)

    var rawkey interface{} // This is the raw key, like *rsa.PrivateKey or *ecdsa.PrivateKey
    if err := key.Raw(&rawkey); err != nil {
      log.Printf("failed to create public key: %s", err)
      return
    }
    // Use rawkey for jws.Verify() or whatever.
    _ = rawkey

    // You can create jwk.Key from a raw key, too
    fromRawKey, err := jwk.FromRaw(rawkey)
    if err != nil {
      log.Printf("failed to acquire raw key from jwk.Key: %s", err)
      return
    }

    // Keys can be serialized back to JSON
    jsonbuf, err := json.Marshal(key)
    if err != nil {
      log.Printf("failed to marshal key into JSON: %s", err)
      return
    }

    fromJSONKey, err := jwk.Parse(jsonbuf)
    if err != nil {
      log.Printf("failed to parse json: %s", err)
      return
    }
    _ = fromJSONKey
    _ = fromRawKey
  }
  // OUTPUT:
}

//nolint:govet
func ExampleJWK_MarshalJSON() {
  // to get the same values every time, we need to create a static source
  // of "randomness"
  rdr := bytes.NewReader([]byte("01234567890123456789012345678901234567890123456789ABCDEF"))
  raw, err := ecdsa.GenerateKey(elliptic.P384(), rdr)
  if err != nil {
    fmt.Printf("failed to generate new ECDSA private key: %s\n", err)
    return
  }

  key, err := jwk.FromRaw(raw)
  if err != nil {
    fmt.Printf("failed to create ECDSA key: %s\n", err)
    return
  }
  if _, ok := key.(jwk.ECDSAPrivateKey); !ok {
    fmt.Printf("expected jwk.ECDSAPrivateKey, got %T\n", key)
    return
  }

  key.Set(jwk.KeyIDKey, "mykey")

  buf, err := json.MarshalIndent(key, "", "  ")
  if err != nil {
    fmt.Printf("failed to marshal key into JSON: %s\n", err)
    return
  }
  fmt.Printf("%s\n", buf)

  // OUTPUT:
  // {
  //   "crv": "P-384",
  //   "d": "ODkwMTIzNDU2Nzg5MDEyMz7deMbyLt8g4cjcxozuIoygLLlAeoQ1AfM9TSvxkFHJ",
  //   "kid": "mykey",
  //   "kty": "EC",
  //   "x": "gvvRMqm1w5aHn7sVNA2QUJeOVcedUnmiug6VhU834gzS9k87crVwu9dz7uLOdoQl",
  //   "y": "7fVF7b6J_6_g6Wu9RuJw8geWxEi5ja9Gp2TSdELm5u2E-M7IF-bsxqcdOj3n1n7N"
  // }
}
```
source: [examples/jwk_example_test.go](https://github.com/lestrrat-go/jwx/blob/v2/examples/jwk_example_test.go)
<!-- END INCLUDE -->
