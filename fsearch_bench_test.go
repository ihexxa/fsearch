package fsearch

import (
	"fmt"
	"testing"
)

func genTreePaths(sampleSize, factor int) map[string]bool {
	paths := map[string]bool{"root": true}
	nextLevelPaths := map[string]bool{}

	for len(paths) <= sampleSize {
		for pathname := range paths {
			for i := 0; i < factor; i++ {
				subPath := fmt.Sprintf("%s/%s", pathname, randStr.Alphabets())
				nextLevelPaths[subPath] = true
			}

			if len(nextLevelPaths) > sampleSize {
				break
			}
		}

		paths = nextLevelPaths
		nextLevelPaths = map[string]bool{}
	}

	return paths
}

func BenchmarkFSearch(b *testing.B) {
	const (
		segmentLen = 4
		factor     = 10
		sampleSize = 10000 * 100
	)

	var pathnames = genTreePaths(sampleSize, factor)
	fmt.Printf("bench sample size (%d)\n", len(pathnames))
	fs := New("/", 0)
	for pathname := range pathnames {
		err := fs.AddPath(pathname)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		fs.Search("ab")
	}
}
