# Errors

`errors` package serve to build an arbitrary long error chain in order to capture errors returned from nested service calls.

`errors` package contains the custom Go `error` interface implementation, `Error`. You use the `Error` interface to **wrap** two errors in a containing error as well as to test recursively if a given error **contains** some other error.
