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

var (
	keywordRandStr *randstr.RandStr
	seed           int64
)

const resultSize = 1000000000

func init() {
	seed = time.Now().UnixNano()
	fmt.Printf("seed: %d\n", seed)
	keywordRandStr.Seed(seed)
}

func TestFSearchOperations(t *testing.T) {
	keywordRandStr := randstr.NewRandStr([]string{}, true, 2)

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

	t.Run("AddPath/Rename: rename segments test", func(t *testing.T) {
		fs := New("/", resultSize)
		newPathSeg := "renamed"

		for _, pathname := range []string{
			"a/b/c",
		} {
			err := fs.AddPath(pathname)
			if err != nil {
				t.Fatal(err)
			}

			parts := strings.Split(pathname, "/")
			renamedPrefixParts := []string{}
			for i, part := range parts {
				oldPrefixParts := append(renamedPrefixParts, part)
				oldPrefix := strings.Join(oldPrefixParts, "/")

				fmt.Println(oldPrefix, newPathSeg)
				fmt.Println(fs.tree.String())
				err := fs.RenamePath(oldPrefix, newPathSeg)
				if err != nil {
					t.Fatal(err)
				}

				oldPath := strings.Join(parts[:i+1], "/")
				newPath := strings.Join(append(append(renamedPrefixParts, newPathSeg), parts[i+1:]...), "/")
				fmt.Println("check", oldPath, newPath)
				checkPaths(t, map[string][]*Node{oldPath: nil}, fs, false)
				checkPaths(t, map[string][]*Node{newPath: nil}, fs, true)

				renamedPrefixParts = append(renamedPrefixParts, newPathSeg)
			}
		}
	})
}

func TestFSearchPersistency(t *testing.T) {
	t.Run("test persistency", func(t *testing.T) {
		fs := New("/", resultSize)
		paths := genPaths(128)

		expectedPaths := map[string][]*Node{}
		for pathname := range paths {
			err := fs.AddPath(pathname)
			if err != nil {
				t.Fatal(err)
			}
			expectedPaths[pathname] = nil
		}

		rowsChan := fs.Marshal()

		fs2 := New("/", resultSize)
		err := fs2.Unmarshal(rowsChan)
		if err != nil {
			t.Fatal(err)
		}

		checkPaths(t, expectedPaths, fs2, true)
	})
}

func TestFSearchRandom(t *testing.T) {
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

	t.Run("AddPath/Rename: rename root random test", func(t *testing.T) {
		fs := New("/", resultSize)
		paths := genPaths(128)

		oldRoot := "000"
		newRootName := "111"
		var err error
		for pathname := range paths {
			originalPath := fmt.Sprintf("%s/%s", oldRoot, pathname)
			err = fs.AddPath(originalPath)
			if err != nil {
				t.Fatal(err)
			}
		}

		err = fs.RenamePath(oldRoot, newRootName)
		if err != nil {
			t.Fatal(err)
		}

		originalPaths := map[string][]*Node{}
		renamedPaths := map[string][]*Node{}
		for pathname := range paths {
			originalPath := fmt.Sprintf("%s/%s", oldRoot, pathname)
			renamedPath := fmt.Sprintf("%s/%s", newRootName, pathname)
			originalPaths[originalPath] = nil
			renamedPaths[renamedPath] = nil
		}

		checkPaths(t, originalPaths, fs, false)
		checkPaths(t, renamedPaths, fs, true)
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
