# Fsearch
_An in-memory index which finds a keyword in millions of pathnames within milliseconds._

<a href="https://github.com/ihexxa/fsearch/actions">
    <img src="https://github.com/ihexxa/fsearch/workflows/ci-fsearch/badge.svg" />
</a>
<a href="https://goreportcard.com/report/github.com/ihexxa/fsearch">
    <img src="https://goreportcard.com/badge/github.com/ihexxa/fsearch" />
</a>

## Features
- Fast: search a keyword in millions of directories within milliseconds (see benchmark).
- Compact: indexing 1M pathnames with around 500MB memory.
- Simple: less than 5 APIs

## Examples
```golang
const maxResultSize = 50 // the upper bound of matched results size
const pathSeparator = "/"
fs := New(pathSeparator, maxResultSize)

// add paths
path1 := "a/keyword/c"
path2 := "a/b/keyword"
_ := fs.AddPath(path1)
_ := fs.AddPath(path2)

// search for a key word
matchedPaths, _ := fs.Search("keyword") // matchedPaths should contain both path1 and path2

// delete paths
_ := fs.DelPath(path1)
_ := fs.DelPath(path2)

// move a path
_ := fs.MovePath("a", "a/b/keyword") // "a", "a/keyword", "a/keyword/c" will be under path2
```
