# webmux

[![GoDoc Status]][GoDoc]

Package webmux provides a HTTP request multiplexer for Go web servers.

## Features

- **Performant:** Very efficient, very fast
- **Flexible:** Match on methods, named parameters (`/users/:id`), and wildcards (`/img/*`)
- **Correct:**  Handles `HEAD` and `OPTIONS` requests, 405 responses, etc.
- **Compatible:** Compatible with `net/http` requests, responses, and handlers

## Installation

```
go get -u go.destructure.dev/webmux
```

## Usage

```go
mux := webmux.NewMux()

greet := func(w http.ResponseWriter, r *http.Request) error {
    m, _ := webmux.FromContext(r.Context())

    _, err := fmt.Fprintf(w, "Hello %s!", m.Param("name"))

    return err
}

mux.HandleFunc(http.MethodGet, "/greet/:name", greet)
```

## FAQ

### Why another router?

There weren't any other routers that hit on all the right features.

Before Go 1.22 `net/http` couldn't match path patterns. Now it can but it made a lot of compromises to keep backwards compatibility. Because of those compromises it's API is confusing and error prone, and the implementation cannot be efficient.

The matching logic for a lot of third party routers is complicated. And complex matching rules slow down every single request. Some routers dont' allow "conflicting" routes like `/users/:id` and `/users/new`, when intuitively you would think that should be allowed, with the exact match taking priority. Others handle trailing and duplicate slashes in inconsistent ways when they shouldn't really matter. Some depend on the order the routes were registered in the code to determine priority.

Handling `OPTIONS` and `HEAD` requests correctly is important for APIs but most routers don't. A related and often overlooked issue is sending the `Allow` header in `405` responses. The router has to be designed for this up front or the method lookup will be slow, which is a problem for APIs where lots of requests get preflighted by the browser.

[GoDoc Status]: https://godoc.org/go.destructure.dev/webmux?status.svg
[GoDoc]: https://pkg.go.devgo.destructure.dev/webmux/