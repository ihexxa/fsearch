package fsearch

import (
	"fmt"
	"testing"
)

func BenchmarkFSearch(b *testing.B) {
	const (
		segmentLen = 4
		factor     = 10
		sampleSize = 10000 * 100
	)

	genTreePaths := func(sampleSize int) map[string]bool {
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

	pathnames := genTreePaths(sampleSize)
	fmt.Printf("bench sample size (%d)\n", len(pathnames))
	for i := 0; i < b.N; i++ {
		fs := New("/", 0)
		for pathname := range pathnames {
			err := fs.AddPath(pathname)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}
