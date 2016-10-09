bone [![GoDoc](https://godoc.org/github.com/squiidz/bone?status.png)](http://godoc.org/github.com/go-zoo/bone) [![Build Status](https://travis-ci.org/go-zoo/bone.svg)](https://travis-ci.org/go-zoo/bone) [![Go Report Card](https://goreportcard.com/badge/go-zoo/bone)](https://goreportcard.com/report/go-zoo/bone)
=======

## What is bone ?

Bone is a lightweight and lightning fast HTTP Multiplexer for Golang. It support :

- URL Parameters
- REGEX Parameters
- Wildcard routes
- Router Prefix
- Sub Router, `mux.SubRoute()`, support most standard router (bone, gorilla/mux, httpRouter etc...)
- Http method declaration
- Support for `http.Handler` and `http.HandlerFunc`
- Custom NotFound handler
- Respect the Go standard `http.Handler` interface

![alt tag](https://c2.staticflickr.com/2/1070/540747396_5542b42cca_z.jpg)

## Speed

```
- BenchmarkBoneMux        10000000               118 ns/op
- BenchmarkZeusMux          100000               144 ns/op
- BenchmarkHttpRouterMux  10000000               134 ns/op
- BenchmarkNetHttpMux      3000000               580 ns/op
- BenchmarkGorillaMux       300000              3333 ns/op
- BenchmarkGorillaPatMux   1000000              1889 ns/op
```

 These test are just for fun, all these router are great and really efficient.
 Bone do not pretend to be the fastest router for every job.

## Example

``` go

package main

import(
  "net/http"

  "github.com/go-zoo/bone"
)

func main () {
  mux := bone.New()

  // mux.Get, Post, etc ... takes http.Handler
  mux.Get("/home/:id", http.HandlerFunc(HomeHandler))
  mux.Get("/profil/:id/:var", http.HandlerFunc(ProfilHandler))
  mux.Post("/data", http.HandlerFunc(DataHandler))

  // Support REGEX Route params
  mux.Get("/index/#id^[0-9]$", http.HandleFunc(IndexHandler))

  // Handle take http.Handler
  mux.Handle("/", http.HandlerFunc(RootHandler))

  // GetFunc, PostFunc etc ... takes http.HandlerFunc
  mux.GetFunc("/test", Handler)

  http.ListenAndServe(":8080", mux)
}

func Handler(rw http.ResponseWriter, req *http.Request) {
  // Get the value of the "id" parameters.
  val := bone.GetValue(req, "id")

  rw.Write([]byte(val))
}

```
## Changelog

#### Update 10 September 2016

- Add support for go1.7 net.Context

#### Update 25 September 2015

- Add support for Sub router

Example :
``` go
func main() {
    mux := bone.New()
    sub := mux.NewRouter()

    sub.GetFunc("/test/example", func(rw http.ResponseWriter, req *http.Request) {
        rw.Write([]byte("From sub router !"))
    })

    mux.SubRoute("/api", sub)

    http.ListenAndServe(":8080", mux)
}

```


#### Update 26 April 2015

- Add Support for REGEX parameters, using ` # ` instead of ` : `.
- Add Mux method ` mux.GetFunc(), mux.PostFunc(), etc ... `, takes ` http.HandlerFunc ` instead of ` http.Handler `.

Example :
``` go
func main() {
    mux.GetFunc("/route/#var^[a-z]$", handler)
}

func handler(rw http.ResponseWriter, req *http.Request) {
    bone.GetValue(req, "var")
}
```

#### Update 29 january 2015

- Speed improvement for url Parameters, from ```~ 1500 ns/op ``` to ```~ 1000 ns/op ```.

#### Update 25 december 2014

After trying to find a way of using the default url.Query() for route parameters, i decide to change the way bone is dealing with this. url.Query() is too slow for good router performance.
So now to get the parameters value in your handler, you need to use
` bone.GetValue(req, key) ` instead of ` req.Url.Query().Get(key) `.
This change give a big speed improvement for every kind of application using route parameters, like ~80x faster ...
Really sorry for breaking things, but i think it's worth it.  

## TODO

- DOC
- More Testing
- Debugging
- Optimisation

## Contributing

1. Fork it
2. Create your feature branch (git checkout -b my-new-feature)
3. Write Tests!
4. Commit your changes (git commit -am 'Add some feature')
5. Push to the branch (git push origin my-new-feature)
6. Create new Pull Request

## License
MIT

## Links
- Blog post talking about bone : http://www.peterbe.com/plog/my-favorite-go-multiplexer

## Libs
- Errors dump for Go : [Trash](https://github.com/go-zoo/trash)
- Middleware Chaining module : [Claw](https://github.com/go-zoo/claw)
