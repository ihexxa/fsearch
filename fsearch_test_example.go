package fsearch

import (
	"fmt"
	"testing"
)

func TestFSearchExample(t *testing.T) {
	t.Run("test example", func(t *testing.T) {
		const maxResultSize = 50 // the upper bound of matched results size
		const pathSeparator = "/"
		fs := New(pathSeparator, maxResultSize)

		// add paths
		path1 := "a/keyword/c"
		path2 := "a/b/keyword"
		err := fs.AddPath(path1)
		if err != nil {
			t.Fatal(err)
		}
		err = fs.AddPath(path2)
		if err != nil {
			t.Fatal(err)
		}

		// search for a key word
		matchedPaths, err := fs.Search("keyword") // matchedPaths should contain both path1 and path2
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("%+v", matchedPaths)

		// move a path
		err = fs.MovePath("a/keyword", "a/b/keyword") // "a/keyword", "a/keyword/c" will be under path2
		if err != nil {
			t.Fatal(err)
		}

		// rename a path
		err = fs.RenamePath("a/b/keyword", "keyword2") // entry "a/b/keyword" is renamed to "a/b/keyword2"
		if err != nil {
			t.Fatal(err)
		}

		// delete paths
		err = fs.DelPath("a/b/keyworde")
		if err != nil {
			t.Fatal(err)
		}
	})
}
