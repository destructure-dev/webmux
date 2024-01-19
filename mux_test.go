package webmux_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/alecthomas/assert/v2"
	"go.destructure.dev/webmux"
)

func newTestHandler(v string) webmux.Handler {
	return webmux.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Add("Content-Length", strconv.Itoa(len(v)))
		w.Write([]byte(v))
		w.WriteHeader(200)

		return nil
	})
}

func TestServeMuxLookupParams(t *testing.T) {
	mux := webmux.NewMux()

	h := newTestHandler("hello")
	mux.Handle(http.MethodGet, "/users/:user/posts/:post", h)

	r := httptest.NewRequest(http.MethodGet, "/users/1/posts/2", nil)

	match := mux.Lookup(r)

	assert.NotZero(t, match)
	assert.Equal(t, h, match.Handler(http.MethodGet))
	assert.Equal(t, "1", match.Param("user"))
	assert.Equal(t, "2", match.Param("post"))
}

func TestServeMuxLookupPatternMatching(t *testing.T) {
	var tests = []struct {
		name     string
		patterns []string
		reqURL   string
		want     string
	}{
		{
			"trailing slash ignored",
			[]string{"/users", "/users/*any"},
			"/users/",
			"/users",
		},
		{
			"exact over param",
			[]string{"/users/new", "/users/:id"},
			"/users/new",
			"/users/new",
		},
		{
			"exact over prefix",
			[]string{"/users/new", "/users/*any"},
			"/users/new",
			"/users/new",
		},
		{
			"exact over prefix when trailing slash",
			[]string{"/home", "home/:page"},
			"/home/",
			"/home",
		},
		{
			"root path",
			[]string{"/", "/users"},
			"/",
			"/",
		},
		{
			"params over prefix",
			[]string{"/assets/*", "/assets/:kind/:name"},
			"/assets/js/app.js",
			"/assets/:kind/:name",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mux := webmux.NewMux()

			for _, p := range tc.patterns {
				h := newTestHandler(p)
				mux.Handle(http.MethodGet, p, h)
			}

			r := httptest.NewRequest(http.MethodGet, tc.reqURL, nil)

			match := mux.Lookup(r)

			assert.NotZero(t, match)

			assert.Equal(t, tc.want, match.Pattern())
		})
	}
}

func TestServeMuxLookupMethodMatching(t *testing.T) {
	mux := webmux.NewMux()

	hGet := newTestHandler("GET /users")
	mux.Handle(http.MethodGet, "/users", hGet)
	hPost := newTestHandler("POST /users")
	mux.Handle(http.MethodPost, "/users", hPost)

	r := httptest.NewRequest(http.MethodPost, "/users", nil)

	match := mux.Lookup(r)

	assert.NotZero(t, match)

	assert.Equal(t, hGet, match.Handler(http.MethodGet))
	assert.Equal(t, hPost, match.Handler(http.MethodPost))
}

func TestServeMuxLookupMethodSetMatching(t *testing.T) {
	mux := webmux.NewMux()

	h := newTestHandler("GET|POST /users")
	mux.HandleMethods(webmux.Methods(http.MethodGet, http.MethodPost), "/users", h)

	r := httptest.NewRequest(http.MethodPost, "/users", nil)

	match := mux.Lookup(r)

	assert.NotZero(t, match)

	assert.Equal(t, h, match.Handler(http.MethodGet))
	assert.Equal(t, h, match.Handler(http.MethodPost))
}

func TestServeMuxLookupIntermediateNonMatch(t *testing.T) {
	mux := webmux.NewMux()

	h1 := newTestHandler("1")
	mux.Handle(http.MethodGet, "/*", h1)

	h2 := newTestHandler("2")
	mux.Handle(http.MethodGet, "/:foo/bar/baz", h2)

	r := httptest.NewRequest(http.MethodGet, "/1/bar", nil)

	match := mux.Lookup(r)

	assert.NotZero(t, match)

	assert.Equal(t, h1, match.Handler(http.MethodGet))
}

func ExampleHandleFunc() {
	mux := webmux.NewMux()

	greet := func(w http.ResponseWriter, r *http.Request) error {
		m, _ := webmux.FromContext(r.Context())

		name := m.Param("name")

		_, err := fmt.Fprintf(w, "Hello %s!", name)

		return err
	}

	mux.HandleFunc(http.MethodGet, "/greet/:name", greet)
}

func BenchmarkLookupBasic(b *testing.B) {
	h0 := newTestHandler("h0")
	h1 := newTestHandler("h1")
	h2 := newTestHandler("h2")

	mux := webmux.NewMux()

	mux.Handle(http.MethodGet, "/users/:id", h0)
	mux.Handle(http.MethodGet, "/foo/:id", h1)
	mux.Handle(http.MethodGet, "/bar/:id", h2)

	r := httptest.NewRequest(http.MethodPost, "/users/mattya", nil)

	b.ResetTimer()

	var blackhole any

	for i := 0; i < b.N; i++ {
		match := mux.Lookup(r)

		if match == nil {
			b.Error("Not found")
		}

		blackhole = match
	}

	_ = blackhole
}
