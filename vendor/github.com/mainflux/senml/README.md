# SenML

[![coverage][cov-badge]][cov-url]
[![go report card][grc-badge]][grc-url]
[![license][license]](LICENSE)

This repository contains a lightweight implementation of [RFC 8428 Sensor Measurement Lists (SenML)](https://tools.ietf.org/html/rfc8428)

## Codec

The following formats are supported:

- JSON
- XML
- CBOR

## Normalization

Normalized (resolved) SenML Pack consists of resolved SenML Records. A SenML Record is referred to as "resolved" if it does not contain any base values, i.e., labels starting with the character "b", except for Base Version fields, and has no relative times.[*](https://tools.ietf.org/html/rfc8428#section-4.6)

## Validation

Valid SenML Record is the record with valid all the required fields and `exactly one` value field. Base values, if present, must be valid, as well. The Pack is valid if all the Records are valid and have the same Base Version.
All SenML Records in a Pack must have the same version number. This is typically done by adding a Base Version field to only the first Record in the Pack or by using the default value.[*](https://tools.ietf.org/html/rfc8428#section-4.4)

[cov-badge]: https://codecov.io/gh/mainflux/senml/branch/master/graph/badge.svg
[cov-url]: https://codecov.io/gh/mainflux/senml
[grc-badge]: https://goreportcard.com/badge/github.com/mainflux/senml
[grc-url]: https://goreportcard.com/report/github.com/mainflux/senml
[license]: https://img.shields.io/badge/license-Apache%20v2.0-blue.svg
