package fsearch

import qradix "github.com/ihexxa/q-radix/v3"

type FSearch struct {
	radix *qradix.RTree
	tree  *Tree
	nodes map[int64]*Node
}
