package fsearch

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ihexxa/randstr"
)

const (
	sep        = "/"
	pathLen    = 5
	sampleSize = 10
	segmentLen = 4
)

var randStr = randstr.NewRandStr([]string{}, true, segmentLen)

func genPaths(sampleSize int) map[string][]*Node {
	paths := map[string][]*Node{}
	for i := 0; i < sampleSize; i++ {
		parts := []string{}
		for j := 0; j < pathLen; j++ {
			parts = append(parts, randStr.Alphabets())
		}
		paths[strings.Join(parts, sep)] = []*Node{}
	}
	return paths
}

func TestTree(t *testing.T) {
	randStr.Seed(time.Now().UnixNano())

	t.Run("AddPath/DelPath/GetPath basic test", func(t *testing.T) {
		tree := NewTree(sep)
		paths := genPaths(sampleSize)

		for pathname := range paths {
			createdNodes, err := tree.AddPath(pathname)
			if err != nil {
				t.Fatal(err)
			}
			paths[pathname] = createdNodes

			for _, createdNode := range createdNodes {
				createdPath, err := tree.GetPath(createdNode)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.HasPrefix(pathname, createdPath) {
					t.Fatalf("%s is not a prefix of %s", createdPath, pathname)
				}
			}
		}

		for pathname, createdNodes := range paths {
			deletedNode, err := tree.DelPath(pathname)
			if err != nil {
				t.Fatal(err)
			}
			if deletedNode != createdNodes[len(createdNodes)-1] {
				t.Fatal("deleted node is not correct")
			}

			for i, createdNode := range createdNodes {
				createdPath, err := tree.GetPath(createdNode)
				if err != nil {
					if errors.Is(err, ErrNotFound) && i != len(createdNodes)-1 {
						t.Fatalf("the last node should not be found")
					}
				}
				if i != len(createdNodes)-1 {
					if !strings.HasPrefix(pathname, createdPath) {
						t.Fatalf("%s is not a prefix of %s", createdPath, pathname)
					}
				}
			}
		}
	})

	t.Run("marshal/unmarshal test", func(t *testing.T) {
		tree1 := NewTree(sep)
		paths := genPaths(sampleSize)

		for pathname := range paths {
			createdNodes, err := tree1.AddPath(pathname)
			if err != nil {
				t.Fatal(err)
			}
			paths[pathname] = createdNodes
		}

		rowsChan := tree1.Marshal()
		if err := tree1.Err(); err != nil {
			t.Fatal(err)
		}

		tree2 := NewTree(sep)
		tree2.Unmarshal(rowsChan)
		if err := tree2.Err(); err != nil {
			t.Fatal(err)
		}

		queue1 := []*Node{tree1.root}
		queue2 := []*Node{tree2.root}
		nodes1 := map[int64]*Node{}
		nodes2 := map[int64]*Node{}
		for len(queue1) > 0 {
			node1, node2 := queue1[0], queue2[0]
			queue1, queue2 = queue1[1:], queue2[1:]

			nodes1[node1.id] = node1
			nodes2[node2.id] = node2
			for _, child := range node1.children {
				queue1 = append(queue1, child)
			}
			for _, child := range node2.children {
				queue2 = append(queue2, child)
			}
		}

		if len(nodes1) != len(nodes2) {
			t.Fatalf("nodes are not equal (%d) (%d)", len(nodes1), len(nodes2))
		}
		if tree1.MaxId != tree2.MaxId {
			t.Fatalf("max Ids are not equal (%d) (%d)", tree1.MaxId, tree2.MaxId)
		}
		if tree1.PathSeparator != tree2.PathSeparator {
			t.Fatalf("path sep are not equal (%s) (%s)", tree1.PathSeparator, tree2.PathSeparator)
		}

		for id, node1 := range nodes1 {
			node2, ok := nodes2[id]
			if !ok {
				t.Fatalf("id %d is not found in nodes2", id)
			}
			if node1.id != node2.id || node1.name != node2.name {
				t.Fatalf("node are not equal (%+v) (%+v)", node1, node2)
			}

			for name, child1 := range node1.children {
				child2, ok := node2.children[name]
				if !ok {
					t.Fatalf("node1 is not found in node2 (%s)", name)
				}
				if child1.id != child2.id {
					t.Fatalf("nodes' ids are not equal (%d) (%d)", child1.id, child2.id)
				}
			}

			if node1.parent != nil && node2.parent != nil {
				if node1.parent.id != node2.parent.id {
					t.Fatalf("node parent are not equal (%+v) (%+v)", node1.parent.id, node2.parent.id)
				}
			} else if !(node1.parent == nil && node2.parent == nil) {
				t.Fatalf("node parent are not equal (%+v) (%+v)", node1.parent.id, node2.parent.id)
			}
		}
	})
}
