package fsearch

import (
	"errors"
	"sync"
	"time"

	qradix "github.com/ihexxa/q-radix/v3"
)

type FSearch struct {
	radix       *qradix.RTree
	tree        *Tree
	nodes       map[int64]*Node
	idsToDelete chan int64
	on          bool
	lock        *sync.RWMutex
}

func New(pathSeparator string) *FSearch {
	fs := &FSearch{
		on:    true,
		radix: qradix.NewRTree(),
		tree:  NewTree(pathSeparator),
		nodes: map[int64]*Node{},
		lock:  &sync.RWMutex{},
	}
	go fs.purgeNodes()

	return fs
}

func (fs *FSearch) purgeNodes() {
	var targetNodeId int64
	for fs.on {
		targetNodeId = <-fs.idsToDelete
		node, ok := fs.nodes[targetNodeId]
		if !ok {
			continue
		}

		queue := []*Node{node}
		idsToDelete := []int64{node.id}
		for len(queue) > 0 {
			node := queue[0]
			queue = queue[1:]

			for _, child := range node.children {
				queue = append(queue, child)
				idsToDelete = append(idsToDelete, child.id)
			}
		}

		for _, nodeId := range idsToDelete {
			delete(fs.nodes, nodeId)
		}
	}
}

func (fs *FSearch) Stop() {
	fs.on = false
	for {
		if len(fs.idsToDelete) > 0 {
			time.Sleep(time.Duration(100) * time.Millisecond)
		} else {
			break
		}
	}
}

func (fs *FSearch) AddPath(pathname string) error {
	nodes, err := fs.tree.AddPath(pathname)
	if err != nil {
		return err
	}

	var keyword string
	var nodeIdsVal interface{}
	for _, node := range nodes {
		fs.nodes[node.id] = node
		runes := []rune(node.name)

		for i := 0; i < len(runes); i++ {
			keyword = string(runes[i:])
			nodeIdsVal, err = fs.radix.Get(keyword)
			if err != nil {
				if errors.Is(err, qradix.ErrNotExist) {
					nodeIdsVal = []int64{}
				} else {
					return err
				}
			}

			nodeIds := nodeIdsVal.([]int64)
			_, err = fs.radix.Insert(keyword, append(nodeIds, node.id))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (fs *FSearch) DelPath(pathname string) error {
	deleteNode, err := fs.tree.DelPath(pathname)
	if err != nil {
		return err
	}

	fs.idsToDelete <- deleteNode.id
	return nil
}

func (fs *FSearch) MovePath(pathname, dstParentPath string) error {
	return fs.tree.MovePath(pathname, dstParentPath)
}

func (fs *FSearch) Search(keyword string) ([]string, error) {
	_, nodeIdsVal, ok := fs.radix.GetBestMatch(keyword)
	if !ok {
		return nil, nil
	}

	var err error
	var node *Node
	var pathname string
	results := []string{}
	nodeIds := nodeIdsVal.([]int64)
	for _, nodeId := range nodeIds {
		node, ok = fs.nodes[nodeId]
		if !ok {
			// TODO: delete the nodeId in the trie
			continue
		}

		pathname, err = fs.tree.GetPath(node)
		if err != nil {
			return nil, err
		}
		results = append(results, pathname)
	}

	return results, nil
}

func (fs *FSearch) Marshal() chan string {
	// TODO: check tree.Err()
	return fs.tree.Marshal()
}

func (fs *FSearch) Unmarshal(rows chan string) {
	fs.tree.Unmarshal(rows)
}
