# Go-pg Adapter

[![Build Status](https://travis-ci.org/casbin/casbin-pg-adapter.svg?branch=master)](https://travis-ci.org/casbin/casbin-pg-adapter)
[![Coverage Status](https://coveralls.io/repos/github/casbin/casbin-pg-adapter/badge.svg?branch=master)](https://coveralls.io/github/casbin/casbin-pg-adapter?branch=master)

Go-pg Adapter is the [Go-pg](https://github.com/go-pg/pg) adapter for [Casbin](https://github.com/casbin/casbin). With this library, Casbin can load policy from PostgreSQL or save policy to it.

## Installation

    go get github.com/casbin/casbin-pg-adapter

## Simple Postgres Example

```go
package main

import (
	pgadapter "github.com/casbin/casbin-pg-adapter"
	"github.com/casbin/casbin/v2"
)

func main() {
	// Initialize a Go-pg adapter and use it in a Casbin enforcer:
	// The adapter will use the Postgres database named "casbin".
	// If it doesn't exist, the adapter will create it automatically.
	a, _ := pgadapter.NewAdapter("postgresql://username:password@postgres:5432/database?sslmode=disable") // Your driver and data source.
	// Alternatively, you can construct an adapter instance with *pg.Options:
	// a, _ := pgadapter.NewAdapter(&pg.Options{
	//     Database: "...",
	//     User: "...",
	//     Password: "...",
	// })

	// Or you can use an existing DB "abc" like this:
	// The adapter will use the table named "casbin_rule".
	// If it doesn't exist, the adapter will create it automatically.

	e := casbin.NewEnforcer("examples/rbac_model.conf", a)

	// Load the policy from DB.
	e.LoadPolicy()

	// Check the permission.
	e.Enforce("alice", "data1", "read")

	// Modify the policy.
	// e.AddPolicy(...)
	// e.RemovePolicy(...)

	// Save the policy back to DB.
	e.SavePolicy()
}
```

## Support for FilteredAdapter interface

You can [load a subset of policies](https://casbin.org/docs/en/policy-subset-loading) with this adapter:

```go
package main

import (
	"github.com/casbin/casbin/v2"
	pgadapter "github.com/casbin/casbin-pg-adapter"
)

func main() {
	a, _ := pgadapter.NewAdapter("postgresql://username:password@postgres:5432/database?sslmode=disable")
	e, _ := casbin.NewEnforcer("examples/rbac_model.conf", a)

	e.LoadFilteredPolicy(&pgadapter.Filter{
		P: []string{"", "data1"},
		G: []string{"alice"},
	})
	...
}
```

## Run all tests

    docker-compose run --rm go

## Debug tests

    docker-compose run --rm go dlv test github.com/casbin/casbin-pg-adapter

## Getting Help

-   [Casbin](https://github.com/casbin/casbin)

## License

This project is under Apache 2.0 License. See the [LICENSE](LICENSE) file for the full license text.
