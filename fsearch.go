package fsearch

import (
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"time"

	qradix "github.com/ihexxa/q-radix/v3"
)

var ErrStopped = errors.New("fsearch is stopped")

type FSearch struct {
	radix       *qradix.RTree
	tree        *Tree
	nodes       map[int64]*Node
	idsToDelete chan int64
	on          bool
	lock        *sync.RWMutex
	resultLimit int
}

// New creates a new Fsearch
// pathSeparator is the path separator in the path
// limit is the upper bound of matched results size, 0 means unlimited (not recommended).
func New(pathSeparator string, limit int) *FSearch {
	fs := &FSearch{
		on:          true,
		radix:       qradix.NewRTree(),
		tree:        NewTree(pathSeparator),
		idsToDelete: make(chan int64, 10240),
		nodes:       map[int64]*Node{},
		lock:        &sync.RWMutex{},
		resultLimit: limit,
	}
	go fs.purgeNodes()

	return fs
}

// purgeNodes is a daemon which receives targetNodeId and deletes the node and all its sub nodes.
func (fs *FSearch) purgeNodes() {
	var targetNodeId int64
	worker := func() {
		targetNodeId = <-fs.idsToDelete

		fs.lock.Lock()
		defer fs.lock.Unlock()

		node, ok := fs.nodes[targetNodeId]
		if !ok {
			return
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

	for fs.on {
		if len(fs.idsToDelete) > 0 {
			worker()
		} else {
			time.Sleep(200 * time.Millisecond)
		}
	}
}

// Stop stops FSearch and it blocks until all deleting operations are applied
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

// AddPath add pathname to the FSearch index
func (fs *FSearch) AddPath(pathname string) error {
	if !fs.on {
		return ErrStopped
	}
	fs.lock.Lock()
	defer fs.lock.Unlock()

	nodes, err := fs.tree.AddPath(pathname)
	if err != nil {
		return err
	}

	for _, node := range nodes {
		fs.nodes[node.id] = node
		err = fs.insertNodeIdToRadix(node.name, node.id)
		if err != nil {
			return err
		}
	}

	return nil
}

// DelPath deletes pathname asynchronously
// NOTE: the pathname is not deleted immediately, the index is eventual consistent.
func (fs *FSearch) DelPath(pathname string) error {
	if !fs.on {
		return ErrStopped
	}
	fs.lock.Lock()
	defer fs.lock.Unlock()

	deletedNode, err := fs.tree.DelPath(pathname)
	if err != nil {
		return err
	}

	parts := strings.Split(pathname, fs.tree.PathSeparator)
	for _, part := range parts {
		err = fs.deleteNodeIdFromRadix(part, deletedNode.id)
		if err != nil {
			return err
		}
	}

	fs.idsToDelete <- deletedNode.id
	return nil
}

// MovePath move the pathname under dstParentPath
func (fs *FSearch) MovePath(pathname, dstParentPath string) error {
	if !fs.on {
		return ErrStopped
	}
	fs.lock.Lock()
	defer fs.lock.Unlock()

	return fs.tree.MovePath(pathname, dstParentPath)
}

func (fs *FSearch) insertNodeIdToRadix(keyword string, nodeId int64) error {
	var err error
	var suffix string
	var nodeIdsVal interface{}

	runes := []rune(keyword)
	for i := 0; i < len(runes); i++ {
		suffix = string(runes[i:])
		nodeIdsVal, err = fs.radix.Get(suffix)
		if err != nil {
			if errors.Is(err, qradix.ErrNotExist) {
				nodeIdsVal = []int64{}
			} else {
				return err
			}
		}

		nodeIds := nodeIdsVal.([]int64)
		_, err = fs.radix.Insert(suffix, append(nodeIds, nodeId))
		if err != nil {
			// TODO: although it is impossible reach here
			// better to add a checking in searching side since not all keys are removed
			return err
		}
	}

	return nil
}

func (fs *FSearch) deleteNodeIdFromRadix(keyword string, nodeId int64) error {
	var err error
	var suffix string
	var nodeIdsVal interface{}

	runes := []rune(keyword)
	for i := 0; i < len(runes); i++ {
		suffix = string(runes[i:])
		nodeIdsVal, err = fs.radix.Get(suffix)
		if err != nil {
			if errors.Is(err, qradix.ErrNotExist) {
				continue
			} else {
				return err
			}
		}

		ids := nodeIdsVal.([]int64)
		for i, id := range ids {
			if id == nodeId {
				if len(ids) == 1 {
					fs.radix.Remove(suffix)
				} else {
					ids[i], ids[len(ids)-1] = ids[len(ids)-1], ids[i]
					_, err := fs.radix.Insert(suffix, ids[:len(ids)-1])
					if err != nil {
						return err
					}
				}
				break
			}
		}
	}

	return nil
}

// Rename renames the file/folder name
func (fs *FSearch) RenamePath(pathname, newName string) error {
	if !fs.on {
		return ErrStopped
	}
	fs.lock.Lock()
	defer fs.lock.Unlock()

	if strings.Contains(newName, fs.tree.PathSeparator) {
		return ErrInvalidPath
	}

	originalName := filepath.Base(pathname)
	if len(originalName) == 0 {
		return ErrInvalidPath
	}

	renamedNode, err := fs.tree.Rename(pathname, newName)
	if err != nil {
		return err
	}

	err = fs.deleteNodeIdFromRadix(originalName, renamedNode.id)
	if err != nil {
		return err
	}
	return fs.insertNodeIdToRadix(newName, renamedNode.id)
}

// Search searches keyword in the FSearch
// It returns pahtnames which contains keyword, the result size is limited by the resultLimit
func (fs *FSearch) Search(keyword string) ([]string, error) {
	if !fs.on {
		return nil, ErrStopped
	}
	fs.lock.RLock()
	defer fs.lock.RUnlock()

	segmentToIds := fs.radix.GetLongerMatches(keyword, fs.resultLimit)

	var ok bool
	var err error
	var node *Node
	var pathname string
	results := []string{}
	for segment, idsVal := range segmentToIds {
		nodeIds := idsVal.([]int64)
		validIds := []int64{}
		for _, nodeId := range nodeIds {
			node, ok = fs.nodes[nodeId]
			if !ok {
				continue
			} else {
				validIds = append(validIds, nodeId)
			}

			pathname, err = fs.tree.GetPath(node)
			if err != nil {
				return nil, err
			}
			results = append(results, pathname)
		}

		if len(validIds) < len(nodeIds) {
			if len(validIds) == 0 {
				fs.radix.Remove(segment)
			} else {
				_, err = fs.radix.Insert(segment, validIds)
				if err != nil {
					return nil, err
				}
			}
		}
		if fs.resultLimit != 0 && len(results) >= fs.resultLimit {
			break
		}
	}

	return results, nil
}

// Marshal serializes FSearch index into string rows
func (fs *FSearch) Marshal() chan string {
	// TODO: snapshot the tree to reduce unavailable time
	fs.lock.RLock()
	defer fs.lock.RUnlock()

	return fs.tree.Marshal()
}

// Marshal deserializes string rows and restore the FSearch index
func (fs *FSearch) Unmarshal(rows chan string) error {
	fs.tree.Unmarshal(rows)

	var keyword string
	var err error
	var nodeIdsVal interface{}
	queue := []*Node{fs.tree.root}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		if node.name != "" {
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

		for _, child := range node.children {
			queue = append(queue, child)
		}
	}

	return nil
}

func (fs *FSearch) Error() error {
	return fs.tree.Err()
}

func (fs *FSearch) String() string {
	return fs.tree.String()
}
