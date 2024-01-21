package hutil

import (
	"fmt"
	"testing"
)

func TestShiftPath(t *testing.T) {
	testCases := []struct {
		input string
		head  string
		tail  string
	}{
		{"/copy", "copy", "/"},
		{"/", "", "/"},
		{"/api/v1/hello", "api", "/v1/hello"},
		{".", "", "/"},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			head, tail := ShiftPath(tc.input)
			if got, exp := head, tc.head; got != exp {
				t.Fatalf("expected %q but got %q", exp, got)
			}
			if got, exp := tail, tc.tail; got != exp {
				t.Fatalf("expected %q but got %q", exp, got)
			}
		})
	}
}

func ExampleShiftPath() {
	var urlPath = "/search/myhome/inventory"

	var head string
	head, urlPath = ShiftPath(urlPath)
	switch head {
	case "search":
		// handler.search(w, req)
	case "feed":
		// handler.feed(w, req)
	default:
		// handler.home(w, req)
	}

	fmt.Printf("head: %s\n", head)
	fmt.Printf("url path: %s\n", urlPath)
	// Output:
	// head: search
	// url path: /myhome/inventory
}
