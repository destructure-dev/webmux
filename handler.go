package webmux

import "net/http"

// A Handler responds to an HTTP request.
// Handler is like [http.Handler] but may return an error.
type Handler interface {
	ServeHTTPErr(http.ResponseWriter, *http.Request) error
}

// The HandlerFunc type is an adapter to allow the use of ordinary functions
// as HTTP handlers. If f is a function with the appropriate signature,
// HandlerFunc(f) is a Handler that calls f.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// ServeHTTPErr calls f(w, r).
func (f HandlerFunc) ServeHTTPErr(w http.ResponseWriter, r *http.Request) error {
	return f(w, r)
}

// FallibleFunc adapts an infallible http handler with no return value to return an error.
// The returned error is always nil.
func FallibleFunc(h http.Handler) Handler {
	return HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		h.ServeHTTP(w, r)

		return nil
	})
}
