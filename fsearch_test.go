package fsearch

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/ihexxa/randstr"
)

func TestFSearch(t *testing.T) {
	keywordRandStr := randstr.NewRandStr([]string{}, true, 2)
	seed := time.Now().UnixNano()
	fmt.Printf("seed: %d\n", seed)
	keywordRandStr.Seed(seed)
	const resultSize = 1000000000

	t.Run("test Search", func(t *testing.T) {
		fs := New("/", resultSize)

		paths := genPaths(128)
		var err error
		for pathname := range paths {
			err = fs.AddPath(pathname)
			if err != nil {
				t.Fatal(err)
			}
		}

		keyword := keywordRandStr.Alphabets()
		fsResults, err := fs.Search(keyword)
		if err != nil {
			t.Fatal(err)
		}

		matchedResults := map[string]bool{}
		matchedLen := math.MaxInt
		for pathname := range paths {
			parts := strings.Split(pathname, "/")
			for _, part := range parts {
				partRunes := []rune(part)
				for i := 1; i <= len(partRunes); i++ {
					for j := 0; j < i; j++ {
						segment := string(partRunes[j:i])
						if strings.HasPrefix(segment, keyword) {
							if len(segment) < matchedLen {
								matchedLen = len(segment)
								matchedResults = map[string]bool{pathname: true}
							} else if len(segment) == matchedLen {
								matchedResults[pathname] = true
							}
							break
						}
					}
				}
			}
		}

		for _, pathname := range fsResults {
			matched := false
			for matchedKey := range matchedResults {
				if strings.HasPrefix(matchedKey, pathname) {
					matched = true
					fmt.Printf("(%s): %s is a prefix of %s\n", keyword, pathname, matchedKey)
					break
				}
			}
			if !matched {
				fmt.Printf("fsResults: %+v\n", fsResults)
				fmt.Printf("matchedResults: %+v\n", matchedResults)
				t.Errorf("length not equal (%d) (%d)", len(fsResults), len(matchedResults))
			}
		}
	})

	type TestCase struct {
		paths       map[string]bool
		targetPaths map[string]bool
		results     map[string]map[string]bool
	}

	t.Run("test Add/Del/Search", func(t *testing.T) {
		for _, tc := range []*TestCase{
			{
				paths: map[string]bool{
					"/ab/b/c": true,
					"/ab/b/d": true,
					"/bc/b/e": true,
				},
				targetPaths: map[string]bool{
					"/ab/b": true,
				},
				results: map[string]map[string]bool{
					"bc": map[string]bool{
						"/bc": true,
					},
				},
			},
			{
				paths: map[string]bool{
					"/ab/1/c": true,
					"/ab/2/d": true,
					"/bc/3/e": true,
				},
				targetPaths: map[string]bool{
					"/ab/1": true,
				},
				results: map[string]map[string]bool{
					"a": map[string]bool{
						"/ab": true,
					},
				},
			},
		} {
			fs := New("/", resultSize)
			for pathname := range tc.paths {
				err := fs.AddPath(pathname)
				if err != nil {
					t.Fatal(err)
				}
			}

			for pathname := range tc.targetPaths {
				err := fs.DelPath(pathname)
				if err != nil {
					t.Fatal(err)
				}
			}

			for {
				if len(fs.idsToDelete) == 0 {
					break
				}
				fmt.Printf("waiting for drain: %d\n", len(fs.idsToDelete))
				time.Sleep(500 * time.Millisecond)
			}

			for keyword, expectedResults := range tc.results {
				results, err := fs.Search(keyword)
				if err != nil {
					t.Fatal(err)
				}
				if len(results) != len(expectedResults) {
					t.Fatalf("(%+v) (%+v) not equal", results, expectedResults)
				}

				for _, result := range results {
					if !expectedResults[result] {
						t.Fatalf("(%s) is not expected", result)
					}
				}
			}
		}
	})

	type MoveTestCase struct {
		paths      map[string]bool
		movedPaths map[string]string
		results    map[string]map[string]bool
	}

	t.Run("test Add/Move/Search", func(t *testing.T) {
		for _, tc := range []*MoveTestCase{
			{
				paths: map[string]bool{
					"/ab/key/a": true,
					"/ab/key/b": true,
					"/root":     true,
				},
				movedPaths: map[string]string{
					"/ab/key": "/root",
				},
				results: map[string]map[string]bool{
					"key": map[string]bool{
						"/root/key": true,
					},
				},
			},
		} {
			fs := New("/", resultSize)
			for pathname := range tc.paths {
				err := fs.AddPath(pathname)
				if err != nil {
					t.Fatal(err)
				}
			}

			for pathname, targetParent := range tc.movedPaths {
				err := fs.MovePath(pathname, targetParent)
				if err != nil {
					t.Fatal(err)
				}
			}

			for {
				if len(fs.idsToDelete) == 0 {
					break
				}
				fmt.Printf("wating for drain: %d\n", len(fs.idsToDelete))
				time.Sleep(500 * time.Millisecond)
			}

			for keyword, expectedResults := range tc.results {
				results, err := fs.Search(keyword)
				if err != nil {
					t.Fatal(err)
				}

				if len(results) != len(expectedResults) {
					t.Fatalf("(%+v) (%+v) not equal", results, expectedResults)
				}

				for _, result := range results {
					if !expectedResults[result] {
						t.Fatalf("(%s) is not expected", result)
					} else {
						fmt.Println("moved", result)
					}
				}
			}
		}
	})

	t.Run("AddPath/DelPath random test", func(t *testing.T) {
		fs := New("/", resultSize)
		paths := genPaths(128)

		var err error
		for pathname := range paths {
			err = fs.AddPath(pathname)
			if err != nil {
				t.Fatal(err)
			}
		}

		checkPaths(t, paths, fs, true)

		for pathname := range paths {
			parts := strings.Split(pathname, "/")
			for i := len(parts) - 1; i >= 0; i-- {
				prefix := strings.Join(parts[:i+1], "/")
				err = fs.DelPath(prefix)
				if err != nil {
					// TODO: check not found, since some path share prefix
					t.Fatal(err)
				}
			}
		}

		checkPaths(t, paths, fs, false)
	})

	t.Run("AddPath/MovePath random test", func(t *testing.T) {
		fs := New("/", resultSize)
		paths := genPaths(128)

		var err error
		for pathname := range paths {
			err = fs.AddPath(pathname)
			if err != nil {
				t.Fatal(err)
			}
		}

		newRoot := "000"
		err = fs.AddPath(newRoot)
		if err != nil {
			t.Fatal(err)
		}

		checkPaths(t, paths, fs, true)

		for pathname := range paths {
			// only move pathname's base to newRoot
			parts := strings.Split(pathname, "/")
			err := fs.MovePath(parts[0], newRoot)
			if err != nil {
				t.Fatal(err)
			}
		}

		movedPaths := map[string][]*Node{}
		for pathname := range paths {
			movedPaths[fmt.Sprintf("%s/%s", newRoot, pathname)] = nil
		}

		checkPaths(t, movedPaths, fs, true)
		checkPaths(t, paths, fs, false)
	})
}

func checkPaths(t *testing.T, pathnames map[string][]*Node, fs *FSearch, shouldExist bool) {
	hasPath := func(keyword, expectedPath string, fs *FSearch) (bool, error) {
		pathnames, err := fs.Search(keyword)
		if err != nil {
			return false, err
		}
		found := false
		for _, pathname := range pathnames {
			if pathname == expectedPath {
				found = true
				break
			}
		}

		if !found {
			return false, nil
		}
		return true, nil
	}

	for pathname := range pathnames {
		parts := strings.Split(pathname, "/")
		for i, part := range parts {
			matchedPrefix := strings.Join(parts[:i+1], "/")

			runes := []rune(part)
			for j := 0; j < len(runes); j++ {
				keywordSuffix := string(runes[j:])
				keywordPrefix := string(runes[:j+1])

				ok1, err1 := hasPath(keywordSuffix, matchedPrefix, fs)
				ok2, err2 := hasPath(keywordPrefix, matchedPrefix, fs)
				if shouldExist {
					if err1 != nil {
						t.Fatal(err1)
					}
					if !ok1 {
						t.Fatalf("path(%s) not found for keywordSuffix(%s)", matchedPrefix, keywordSuffix)
					}
					if err2 != nil {
						t.Fatal(err2)
					}
					if !ok2 {
						t.Fatalf("path(%s) not found for keywordPrefix(%s)", matchedPrefix, keywordPrefix)
					}

				} else {
					if err1 != nil && !errors.Is(err1, ErrNotFound) {
						t.Fatal(err1)
					}
					if ok1 {
						t.Fatalf("path(%s) should not be found for keywordSuffix(%s)", matchedPrefix, keywordSuffix)
					}
					if err2 != nil && !errors.Is(err2, ErrNotFound) {
						t.Fatal(err2)
					}
					if ok2 {
						t.Fatalf("path(%s) should not be found for keywordPrefix(%s)", matchedPrefix, keywordPrefix)
					}
				}
			}
		}
	}
}
