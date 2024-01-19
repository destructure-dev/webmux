package webmux

import (
	"errors"
	"log"
	"net/http"
)

// ErrorHandler handles errors that arise while handling http requests.
type ErrorHandler interface {
	ErrorHTTP(w http.ResponseWriter, r *http.Request, err error)
}

// The ErrorHandlerFunc type is an adapter to allow functions to be used as
// HTTP error handlers.
type ErrorHandlerFunc func(w http.ResponseWriter, r *http.Request, err error)

// ErrorHTTP calls f(w, err, code).
func (f ErrorHandlerFunc) ErrorHTTP(w http.ResponseWriter, r *http.Request, err error) {
	f(w, r, err)
}

// StatusErrorHandler returns a basic error handler that just returns a HTTP status error response.
// Any errors are logged before writing the response.
func StatusErrorHandler() ErrorHandler {
	return ErrorHandlerFunc(StatusError)
}

// StatusError replies to a request with an appropriate status code and HTTP status text.
func StatusError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, ErrMuxNotFound) {
		match, ok := FromContext(r.Context())

		if !ok {
			writeError(w, http.StatusNotFound)
			return
		}

		w.Header().Add("Allow", match.Methods().String())
		writeError(w, http.StatusMethodNotAllowed)

		return
	}

	log.Printf("mux error: %s", err.Error())

	writeError(w, http.StatusInternalServerError)
}

// writeError calls [http.Error] with the [http.StatusText] for code and code.
func writeError(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}
