package webmux

import (
	"strings"
)

// shiftPath shifts the next segment off the front of the path, returning the
// shifted path segment and the remaining path.
func shiftPath(p string) (head string, tail string) {
	i := strings.IndexByte(p[1:], '/') + 1

	if i <= 0 {
		return p[1:], ""
	}

	return p[1:i], p[i:]
}

// cleanPath returns the canonical URL path for p.
func cleanPath(p string) string {
	if p == "" {
		return "/"

	}

	if p[0] != '/' {
		p = "/" + p

	}

	return p
}
