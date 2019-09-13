package hutil

import (
	"path"
	"strings"
)

// ShiftPath splits the path into the first component and the rest of
// the path.
// The returned head will never have a slash in it, if the path has no tail head will be empty.
// The tail will never have a trailing slash.
func ShiftPath(p string) (head string, tail string) {
	p = path.Clean("/" + p)

	pos := strings.Index(p[1:], "/")
	if pos == -1 {
		return p[1:], "/"
	}

	p = p[1:]

	return p[:pos], p[pos:]
}
