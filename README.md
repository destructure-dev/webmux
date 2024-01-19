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

### Quick start

The following example creates a new `ServeMux`, adds a handler function that takes a parameter, and starts a web server:

```go
mux := webmux.New()

greet := func(w http.ResponseWriter, r *http.Request) error {
    m, _ := webmux.FromContext(r.Context())

    name := m.Param("name")

    _, err := fmt.Fprintf(w, "Hello %s!", name)

    return err
}

mux.HandleFunc(http.MethodGet, "/greet/:name", greet)

log.Fatal(http.ListenAndServe(":3030", mux))
```

A full runnable example showing the necessary imports is available in the [_examples](./_examples/hello/main.go) directory.

### Registration

The multiplexer dispatches requests to handlers based on a HTTP method and URL path pattern.

A handler is registered using the `Handle` or `HandleFunc` methods, like this:

```go
mux.Handle(http.MethodGet, "/greet/:name", h)

mux.HandleFunc(http.MethodGet, "/greet/:name", func(w http.ResponseWriter, r *http.Request) error {
    // ...
})
```

When to use `Handle` vs. `HandleFunc` is explained [in the Handlers section](#handlers).

The first argument is a HTTP method. The second argument is a URL path to match, which may contain dynamic path segments. The third argument is the handler or handler function.

### Matching methods

The method is a HTTP method such as GET, POST, or DELETE. Typically methods are provided using the [`net/http` constants](https://pkg.go.dev/net/http#pkg-constants).

To respond to multiple methods with the same handler, use `HandleMethods` or `HandleMethodsFunc` instead.

The first argument must be a `MethodSet`. A `MethodSet` can be easily constructed using the `webmux.Methods` function.

```go
mux.HandleMethods(webmux.Methods(http.MethodGet, http.MethodPost), "/users", h)
```

To handle any of the common HTTP methods (this includes all of the methods defined by the `net/http` constants), use the `webmux.AnyMethods` function:

```go
mux.HandleMethods(webmux.AnyMethod(), "/users", h)
```

HTTP allows defining your own methods. For example, WebDAV uses methods such as COPY and LOCK. Because the method is just a string (and a method set is a set of strings) this is fully supported.

If a request matches a path but not a method, a 405 ["Method Not Allowed" response](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/405) should be returned -- not a 404 "Not Found". The default error handler does this automatically and includes the necessary Allow header.

### Matching paths

The path pattern matches the URL path, using a subset of the browser's [URL Pattern API syntax](https://developer.mozilla.org/en-US/docs/Web/API/URL_Pattern_API).

Patterns can contain:

- Literal strings which will be matched exactly.
- Wildcards (`/posts/*`) that match any character.
- Named groups (`/posts/:id`) which extract a part of the matched URL.

The simplest match is a literal match of an exact path:

```go
mux.Handle(http.MethodGet, "/users", h)
```

The pattern `/users` would only match `/users`. It would not match `/users/new`. Any trailing slash is ignored, so a request for `/users/` is interpreted identically to a request for `/users`.

A wildcard or named group may be used to match one or more path segments containing arbitrary strings.

```go
mux.Handle(http.MethodGet, "/users/:id", h)
```

The pattern `/users` would match `/users/1`, `/users/matt`, etc. It would not match `/users/1/settings` or `/users`. When the placeholder is specified with a colon (`:`) the pattern only matches characters until the next slash (`/`).

To greedily match one or more segments until the end of the path, use an asterisk (`*`) instead:

```go
mux.Handle(http.MethodGet, "/users/*", h)
```

The pattern `/users/*` would match `/users/1`, `users/1/settings`, etc. It would not match `/users`, because at least one segment must be matched by the wildcard.

Wildcards may be named:

```go
mux.Handle(http.MethodGet, "/users/*rest", h)
```

This can be useful when extracting the parameter value as explained below.

### Match priority

It can be useful to register patterns that overlap. Consider the following patterns for a hypothetical application:

- `/users/new`
- `/users/:id`
- `/*`

In this example `/users/:id` should match paths like `/users/1`, `/users/new` should only match that literal path, and `/*` should match anything else (typically used to fallback to a Single Page Application).

These patterns match like you would expect. The more exact match is always prioritized over the less exact match. Knowing that, `/users/new` matches over `/users/:id`, and `/users/:id` matches over `/*`.

### Match parameters

When a pattern is matched the path segments corresponding to each match are captured. To access a parameter, first retrieve the `MuxMatch` from the [Request context](https://pkg.go.dev/net/http#Request.Context):

```go
h := func(w http.ResponseWriter, r *http.Request) error {
    match, ok := webmux.FromContext(r.Context())

    // ...
}
```

The `ok` return value will always be `true` within a handler, and the match will not be nil.

Next, retrieve the parameter by calling `MuxMatch.Param` with the parameter's name:

```go
userID := m.Param("id") 
```

The parameter is always a string. It captures everything between the path segments where the parameter appears, or from the start of the path segment to the end of the path if the parameter is a wildcard (`*`).

If a parameter with the given name was not captured, `Param` returns the empty string.

To access all parameters as a slice, call `Params` instead:

```go
params := m.Params() 
```

Once the slice is retrieved, you can access parameters by position. This is useful when parameters are un-named, which is common for wildcards. For example, when matching `/assets/*`, you would get the value corresponding to the wildcard like this:

```go
params := m.Params() 

filepath := params[0]
```

### Handlers

The quick start example used a function or "HandlerFunc". A `HandlerFunc` is just an adapter for implementing the `Handler` interface, which looks like this:

```go
type Handler interface {
	ServeHTTPErr(http.ResponseWriter, *http.Request) error
}
```

Use `ServeMux.Handle` to register a `Handler`, and `ServeMux.HandleFunc` to register a `HandlerFunc`.

### Stdlib handlers

The `net/http` package in the standard library defines the following Handler interface:

```go
type Handler interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}
```

It's nearly identical to ours, but does not allow returning an error. This is incredibly inconvenient when you want to handle errors in one place and leads to a lot of boilerplate. However, a lot of useful packages are compatible with this interface.

To adapt a stdlib compatible Handler, use the `FallibleFunc` function like so:

```go
h := webmux.FallibleFunc(h)
```

The error returned by `h` will always be nil.

### HEAD requests

Responses to [HEAD requests](https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods/HEAD) must return the response headers as if a GET request had been made, but without returning a body.

Unless a HEAD handler is registered, the GET handler will be called for HEAD requests. It is not necessary to do anything different in the handler, as the default `http.ResponseWriter` will omit the body but write the Content-Length header.

When sending a large file of a known length it can be more efficient to check the request method in the handler, then only write the Content-Length header.

### OPTION requests

By default [OPTION requests](https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods/OPTIONS) are handled by sending a 204 No Content response and setting the Allow header. This does not take precendence over not found responses.

This behavior can be overriden by explicitly registering a handler for the OPTION method.

### Error handling

When a handler returns an error the error handler is called. The error handler is responsible for sending an appropriate response to the client and potentially reporting the error.

The default error handler returns "Internal Server Error" in plain text with a 500 status code. You will likely want to override this.

To override the error handler use `ServeMux.HandleError` or `ServeMux.HandleErrorFunc`:

```go
mux.HandleErrorFunc(func(w http.ResponseWriter, r *http.Request, err error) {
    // ...

	log.Printf("error: %s", err.Error())

    code := http.StatusInternalServerError

	http.Error(w, http.StatusText(code), code)
})
```

The error handler should also handle `ErrMuxNotFound` errors; see below.

### Not found errors

When a handler is not found an `ErrMuxNotFound` error is returned. The error handler can then return an appropriate response to the client.

The default error handler provides an example of handling the not found error correctly:

```go
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

// ...
```

Of note is that a 405 Method Not Allowed response is returned with the Allow header if the pattern matched but a handler was not bound for the request method. Otherwise a 404 Not found error is returned.

## FAQ

### Why another router?

There weren't any other routers that hit on all the right features.

Before Go 1.22 `net/http` couldn't match path patterns. Now it can but it made a lot of compromises to keep backwards compatibility. Because of those compromises it's API is confusing and error prone, and the implementation cannot be efficient.

The matching logic for a lot of third party routers is complicated. And complex matching rules slow down every single request. Some routers dont' allow "conflicting" routes like `/users/:id` and `/users/new`, when intuitively you would think that should be allowed, with the exact match taking priority. Others handle trailing and duplicate slashes in inconsistent ways when they shouldn't really matter. Some depend on the order the routes were registered in the code to determine priority.

Handling `OPTIONS` and `HEAD` requests correctly is important for APIs but most routers don't. A related and often overlooked issue is sending the `Allow` header in `405` responses. The router has to be designed for this up front or the method lookup will be slow, which is a problem for APIs where lots of requests get preflighted by the browser.

### Non-Features

There are a lot of features other routers have that aren't present in this package. Most (all?) of these were intentionally omitted.

#### Regex parameters

Regex parameters refers to route parameters of the form `/user/(\\d+)`. Regex parameters make the matching process much slower, even if the route that finally matches did not contain a regex parameter. If the regex does not match the user gets a 404 Not Found error with no context to understand what happened.

Instead of regex parameters, use normal parameters and validate the value within the handler. In the handler you can return a more useful error.

#### Partial segment matching

You can't match parts of a segment as separate parameters, like `/articles/{month}-{day}-{year}`. This is rarely useful for matching; just match on the whole segment and parse it within the handler.

#### Named routes

Some routers let you assign a name to a route, like `users.update` for `PATCH /users/:id`. You can then do "reverse routing", generating a URL by providing the name and parameters.

Calling `RouteURL("users.update", 1)` is not much easier than `/users/+strconv.itoa(1)` and it's less clear.

#### Route groups

This feature is commonly used for "RESTful" JSON APIs:

```go
r.Route("/articles", func(r Router) {
    r.Get("/", listArticles) // GET /articles
    r.Get("/:id", showArticle) // GET /articles/:id
})
```

It's slightly more convenient when writing but everyone that reads it has to re-assemble the path in their head. With more than one level of nesting it's a total mess.

[GoDoc Status]: https://godoc.org/go.destructure.dev/webmux?status.svg
[GoDoc]: https://pkg.go.devgo.destructure.dev/webmux/
