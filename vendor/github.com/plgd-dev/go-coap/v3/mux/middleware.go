package mux

// MiddlewareFunc is a function which receives an Handler and returns another Handler.
// Typically, the returned handler is a closure which does something with the ResponseWriter and Message passed
// to it, and then calls the handler passed as parameter to the MiddlewareFunc.
type MiddlewareFunc func(Handler) Handler

// Middleware allows MiddlewareFunc to implement the middleware interface.
func (mw MiddlewareFunc) Middleware(handler Handler) Handler {
	return mw(handler)
}

// Use appends a MiddlewareFunc to the chain. Middleware can be used to intercept or otherwise modify requests and/or responses, and are executed in the order that they are applied to the Router.
func (r *Router) Use(mwf ...MiddlewareFunc) {
	r.middlewares = append(r.middlewares, mwf...)
}
