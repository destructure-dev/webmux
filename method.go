package webmux

import (
	"net/http"
	"slices"
	"strings"
)

// Common HTTP methods defined in RFC 7231 section 4.3 & RFC 5789.
var commonMethods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodConnect,
	http.MethodOptions,
	http.MethodTrace,
}

// MethodSet is a set of HTTP methods.
type MethodSet []string

// Methods combines the given HTTP methods into a MethodSet.
// Duplicates are exluded to preserve set semantics.
func Methods(methods ...string) MethodSet {
	s := make([]string, 0, len(methods))

	for _, m := range methods {
		if !slices.Contains(s, m) {
			s = append(s, m)
		}
	}

	return MethodSet(s)
}

// AnyMethod returns a new MethodSet of all of the commonly known HTTP methods.
// The set of all methods is not known, thus this uses the more common interpretation
// of any method defined in RFC 7231 section 4.3 & RFC 5789.
func AnyMethod() MethodSet {
	return MethodSet(commonMethods)
}

// Add adds method to m and returns a new MethodSet.
func (m MethodSet) Add(method string) MethodSet {
	if !slices.Contains(m, method) {
		m = append(m, method)
	}

	return m
}

// Has returns true if m contains method.
func (m MethodSet) Has(method string) bool {
	return slices.Contains(m, method)
}

// String implements [fmt.Stringer].
func (m MethodSet) String() string {
	return strings.Join(m, ", ")
}
