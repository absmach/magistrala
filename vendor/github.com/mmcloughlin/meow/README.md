# meow

Golang implementation of the [Meow hash](https://mollyrocket.com/meowhash), an
extremely fast non-cryptographic hash.

[![GoDoc Reference](http://img.shields.io/badge/godoc-reference-5272B4.svg?style=flat-square)](http://godoc.org/github.com/mmcloughlin/meow)

## Warning

The [official
implemention](https://github.com/cmuratori/meow_hash) is _in flux_, therefore this one is too. The [Travis CI build](https://travis-ci.org/mmcloughlin/meow) ([config](.travis.yml)) tests against master branch of the reference implementation, therefore build status should be a good indicator of compatibility. This package is unlikely to be updated until the reference implementation [stabilizes](https://github.com/cmuratori/meow_hash/issues/29).

## License

[Zlib license](https://spdx.org/licenses/Zlib.html) following the [official
implemention](https://github.com/cmuratori/meow_hash/blob/master/LICENSE).
