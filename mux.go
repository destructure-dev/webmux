package webmux

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
)

// ErrMuxNotFound is returned by ServeMux when a matching handler was not found.
var ErrMuxNotFound = errors.New("mux match not found")

// ctxKey is an unexported type to prevent collisions.
type ctxKey int

// muxKey is the key for MuxMatch values in contexts.
var muxKey ctxKey

// ServeMux is an HTTP request multiplexer.
// It matches the method and URL of each incoming request against a list of
// registered routes and calls the handler for the method and pattern that
// most closely matches the request.
//
// Patterns name paths like "/users". A pattern may contain dynamic path segments.
// The syntax for patterns is a subset of the browser's [URL Pattern API]:
//
//   - Literal strings which will be matched exactly.
//   - Wildcards of the form "/users/*" match any string.
//   - Named groups of the form "/users/:id" match any string like wildcards,
//     but assign a name that can be used to lookup the matched segment.
//
// Placeholders may only appear between slashes, as in "/users/:id/profile",
// or as the last path segment, as in "/images/*".
//
// Requests are matched by first looking for an exact match, then falling back
// to pattern matches. Thus the pattern "/users/new" would win over "/users/:id".
// The weight of named and un-named parameters is the same.
//
// More specific matches are prioritized over less specific matches. For example,
// if both "/users" and "/users/:id" are registered, a request for "/users/1"
// would match "/users/:id".
//
// If multiple routes are registered for the same method and pattern, even if
// the parameter names are different, ServeMux will panic.
//
// [URL Pattern API]: https://developer.mozilla.org/en-US/docs/Web/API/URL_Pattern_API
type ServeMux struct {
	errHandler ErrorHandler
	pool       *sync.Pool
	root       *node
}

// NewMux allocates and returns a new ServeMux ready for use.
func NewMux() *ServeMux {
	return &ServeMux{
		errHandler: new(StatusErrorHandler),
		pool: &sync.Pool{
			New: func() any {
				return new(MuxMatch)
			},
		},
		root: &node{},
	}
}

// Handle registers the handler for the given method and pattern.
// If a handler already exists for method and pattern, Handle panics.
func (mux *ServeMux) Handle(method, pattern string, handler Handler) {
	mux.HandleMethods(Methods(method), pattern, handler)
}

// HandleFunc registers the handler function for the given method and pattern.
func (mux *ServeMux) HandleFunc(method, pattern string, handler func(http.ResponseWriter, *http.Request) error) {
	if handler == nil {
		panic("webmux: nil handler")
	}

	mux.HandleMethods(Methods(method), pattern, HandlerFunc(handler))
}

// Handle registers the handler for the given methods and pattern.
func (mux *ServeMux) HandleMethods(methods MethodSet, pattern string, handler Handler) {
	if len(methods) == 0 {
		panic("webmux: empty method set")
	}

	if pattern == "" {
		panic("webmux: invalid pattern")
	}
	if handler == nil {
		panic("webmux: nil handler")
	}

	path := cleanPath(pattern)
	params := make([]string, 0)
	current := mux.root

	for path != "" {
		head, tail := shiftPath(path)

		if head != "" && (head[0] == ':' || head[0] == '*') {
			params = append(params, head[1:])
			head = "*"
		}

		next, ok := current.children[head]

		if !ok {
			next = &node{}
		}

		current.addChild(head, next)
		current = next
		path = tail
	}

	entry := current.entry

	if entry == nil {
		entry = &muxEntry{
			pattern: pattern,
			params:  params,
			methods: Methods(http.MethodOptions),
		}

		current.entry = entry
	}

	for _, method := range methods {
		entry.setHandler(method, handler)
	}
}

// HandleMethodsFunc registers the handler function for the given methods and pattern.
func (mux *ServeMux) HandleMethodsFunc(methods MethodSet, pattern string, handler func(http.ResponseWriter, *http.Request) error) {
	if handler == nil {
		panic("webmux: nil handler")
	}

	mux.HandleMethods(methods, pattern, HandlerFunc(handler))
}

// HandleError registers the error handler for mux.
func (mux *ServeMux) HandleError(errHandler ErrorHandler) {
	mux.errHandler = errHandler
}

// HandleErrorFunc registers the error handler function for mux.
func (mux *ServeMux) HandleErrorFunc(errHandler ErrorHandlerFunc) {
	mux.errHandler = ErrorHandlerFunc(errHandler)
}

// Lookup finds the handlers matching the URL of r.
func (mux *ServeMux) Lookup(r *http.Request) *MuxMatch {
	match := &MuxMatch{}

	return mux.lookup(r, match)
}

func (mux *ServeMux) lookup(r *http.Request, match *MuxMatch) *MuxMatch {
	path := cleanPath(r.URL.Path)
	current := mux.root

	var entry *muxEntry

	for path != "" {
		head, tail := shiftPath(path)

		next, ok := current.children[head]

		if !ok {
			next, ok = current.children["*"]

			if ok {
				match.values = append(match.values, head)
			}
		}

		if !ok {
			return nil
		}

		current = next
		path = tail

		if current.entry != nil {
			entry = current.entry
		}
	}

	if entry == nil {
		return nil
	}

	match.muxEntry = entry

	return match
}

// ServeHTTPErr dispatches the request to the handler whose method and pattern
// most closely matches the request URL, forwarding any errors.
func (mux *ServeMux) ServeHTTPErr(w http.ResponseWriter, r *http.Request) error {
	match := mux.pool.Get().(*MuxMatch)
	match.Reset()
	defer func() {
		mux.pool.Put(match)
	}()

	match = mux.lookup(r, match)

	if match == nil {
		return ErrMuxNotFound
	}

	h := match.Handler(r.Method)

	if h == nil && r.Method == http.MethodHead {
		h = match.Handler(http.MethodGet)
	}

	if h == nil && r.Method == http.MethodOptions {
		w.Header().Add("Allow", match.Methods().String())
		w.WriteHeader(http.StatusNoContent)
		return nil
	}

	if h == nil {
		return ErrMuxNotFound
	}

	r = r.WithContext(NewContext(r.Context(), match))

	return h.ServeHTTPErr(w, r)
}

// ServeHttp implements [http.Handler] by dispatching the request to the handler
// whose method and pattern most closely matches the request URL.
func (mux *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := mux.ServeHTTPErr(w, r)

	if err != nil {
		mux.errHandler.ErrorHTTP(w, r, err)
	}
}

// Type node is a single node in the routing tree.
type node struct {
	children map[string]*node // path segment to child node
	entry    *muxEntry
}

// addChild adds child at path to n.
func (n *node) addChild(path string, child *node) {
	if n.children == nil {
		n.children = make(map[string]*node)
	}

	n.children[path] = child
}

// muxEntry is a leaf node in the routing tree.
// A muxEntry maps HTTP methods to handlers.
type muxEntry struct {
	pattern  string             // raw URL pattern
	params   []string           // param names in the order they appear in pattern
	handlers map[string]Handler // http Method to handler
	methods  MethodSet          // cache of allowed HTTP methods
}

// setHandler sets the handler for method to handler.
// If a handler is already registered, setHandler panics.
// If the method is "GET" and a handler is not registered for method "HEAD",
// the handler is registered for "HEAD" as well.
func (e *muxEntry) setHandler(method string, handler Handler) {
	if e.handlers == nil {
		e.handlers = make(map[string]Handler)
	}

	_, ok := e.handlers[method]

	if ok {
		panic(fmt.Sprintf("web: multiple registrations for %s %s", method, e.pattern))
	}

	e.handlers[method] = handler

	e.methods = e.methods.Add(method)

	if method == http.MethodGet && !e.methods.Has(http.MethodHead) {
		e.methods = e.methods.Add(http.MethodHead)
	}
}

// MuxMatch represents a matched handler for a given request.
// The MuxMatch provides access to the pattern that matched and the values
// extracted from the path for any dynamic parameters that appear in the pattern.
type MuxMatch struct {
	*muxEntry
	values []string
}

// Reset clears the MuxMatch for re-use.
func (m *MuxMatch) Reset() {
	m.muxEntry = nil
	if m.values != nil {
		m.values = m.values[0:0]
	}
}

// Pattern returns the URL pattern for the match.
func (m *MuxMatch) Pattern() string {
	if m.muxEntry == nil {
		return ""
	}

	return m.pattern
}

// Params returns the matched parameters from the URL in the order that they
// appear in the pattern.
func (m *MuxMatch) Params() []string {
	if m.muxEntry == nil {
		return nil
	}

	return m.params
}

// Param returns the parameter value for the given placeholder name.
func (m *MuxMatch) Param(name string) string {
	if m.muxEntry == nil {
		return ""
	}

	// With only a few params so this is faster than allocating a map
	for i, k := range m.params {
		if k == name {
			return m.values[i]
		}
	}

	return ""
}

// Methods returns all of the methods this MuxMatch responds to.
func (m *MuxMatch) Methods() MethodSet {
	if m.muxEntry == nil {
		return MethodSet{}
	}

	return m.methods
}

// Handler returns the handler registered for method.
// Handler returns nil if a handler is not registered for method.
func (m *MuxMatch) Handler(method string) Handler {
	if m.muxEntry == nil {
		return nil
	}

	return m.handlers[method]
}

// NewContext returns a new Context that carries value u.
func NewContext(ctx context.Context, m *MuxMatch) context.Context {
	return context.WithValue(ctx, muxKey, m)
}

// FromContext returns the MuxMatch value stored in ctx, if any.
func FromContext(ctx context.Context) (*MuxMatch, bool) {
	m, ok := ctx.Value(muxKey).(*MuxMatch)
	return m, ok
}
