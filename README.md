# Fsearch
_An in-memory index which finds a keyword from millions of pathnames within milliseconds._

<a href="https://github.com/ihexxa/fsearch/actions">
    <img src="https://github.com/ihexxa/fsearch/workflows/ci-fsearch/badge.svg" />
</a>
<a href="https://goreportcard.com/report/github.com/ihexxa/fsearch">
    <img src="https://goreportcard.com/badge/github.com/ihexxa/fsearch" />
</a>
<a href="https://pkg.go.dev/github.com/ihexxa/fsearch" title="doc">
    <img src="https://pkg.go.dev/badge/github.com/ihexxa/fsearch" />
</a>

## Features
- Fast: search a keyword from millions of directories within milliseconds (see benchmark).
- Compact: indexing 1M pathnames with around 500MB memory.
- Serializable: the index can be serialized and persisted.
- Simple: AddPath, DelPath, MovePath, Rename and so on.

## Examples
```golang
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
```
