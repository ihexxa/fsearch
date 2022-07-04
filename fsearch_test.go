package fsearch

import (
	"fmt"
	"testing"
)

func TestFSearch(t *testing.T) {
	t.Run("", func(t *testing.T) {
		fs := New("/")

		var err error
		for _, pathname := range []string{
			"foo/bar/ab",
			"foo/ab/bar",
			"foo/abc/bar",
		} {
			err = fs.AddPath(pathname)
			if err != nil {
				t.Fatal(err)
			}
		}

		results, err := fs.Search("ab")
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println("search", results)
	})
}
