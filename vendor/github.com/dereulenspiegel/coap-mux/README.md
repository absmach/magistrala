# coap-mux [![Build Status](https://travis-ci.org/dereulenspiegel/coap-mux.svg)](https://travis-ci.org/dereulenspiegel/coap-mux) [![Coverage Status](https://coveralls.io/repos/dereulenspiegel/coap-mux/badge.svg?branch=master&service=github)](https://coveralls.io/github/dereulenspiegel/coap-mux?branch=master) [![GoDoc](https://godoc.org/github.com/olebedev/config?status.png)](https://godoc.org/github.com/dereulenspiegel/coap-mux)

This package provides basic support for routing based on path and method for
CoAP server. This library is heavily inspired by [mux](https://github.com/gorilla/mux).

## Installation

`go get github.com/dereulenspiegel/coap-mux`

## Usage

Create a new router with `NewRouter()` and add routes with `NewRoute()`,
`Handle(path, handler)` or `Path(string)`. The returned route object can be
configured further.
