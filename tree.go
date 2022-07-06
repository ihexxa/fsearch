package fsearch

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

var (
	ErrInvalidPath = errors.New("path is invalid")
	ErrNotFound    = errors.New("path not found")
)

type Node struct {
	id       int64
	name     string
	parent   *Node
	children map[string]*Node
}

type Tree struct {
	root          *Node
	lock          *sync.RWMutex
	MaxId         int64  `json:"maxId"`
	PathSeparator string `json:"pathSeparator"`
	err           error
}

func NewTree(pathSeparator string) *Tree {
	return &Tree{
		root: &Node{
			// parent is nil
			id:       0,
			name:     "",
			children: map[string]*Node{},
		},
		lock:          &sync.RWMutex{},
		MaxId:         1,
		PathSeparator: pathSeparator,
	}
}

// AddPath adds pathname to the tree and return new node ids
func (t *Tree) AddPath(pathname string) ([]*Node, error) {
	pathname = filepath.Clean(pathname)
	parts := strings.Split(pathname, t.PathSeparator)
	if len(parts) == 0 {
		return nil, ErrInvalidPath
	}
	t.lock.Lock()
	defer t.lock.Unlock()

	// TODO: remove base

	node := t.root
	createdNodes := []*Node{}
	for _, part := range parts {
		child, ok := node.children[part]
		if !ok {
			newNode := &Node{
				id:       t.MaxId,
				name:     part,
				parent:   node,
				children: map[string]*Node{},
			}
			node.children[part] = newNode
			createdNodes = append(createdNodes, newNode)
			node = newNode
			t.MaxId++
		} else {
			node = child
		}
	}

	return createdNodes, nil
}

// DelPath deletes pathname from the tree
func (t *Tree) DelPath(pathname string) (*Node, error) {
	pathname = filepath.Clean(pathname)
	parts := strings.Split(pathname, t.PathSeparator)
	if len(parts) == 0 {
		return nil, ErrInvalidPath
	}
	t.lock.Lock()
	defer t.lock.Unlock()

	parent := t.root
	var deletedNode *Node
	for i, part := range parts {
		child, ok := parent.children[part]
		if !ok {
			return nil, ErrNotFound
		}
		if i == len(parts)-1 {
			deletedNode = child
			child.parent = nil
			delete(parent.children, part)
			break
		}
		parent = child
	}

	return deletedNode, nil
}

// MovePath moves pathname under dstParentPath
func (t *Tree) MovePath(pathname, dstParentPath string) error {
	pathname = filepath.Clean(pathname)
	parts := strings.Split(pathname, t.PathSeparator)
	if len(parts) == 0 {
		return ErrInvalidPath
	}

	dstPath := filepath.Clean(dstParentPath)
	dstParts := strings.Split(dstPath, t.PathSeparator)
	if len(dstParts) == 0 {
		return ErrInvalidPath
	}

	t.lock.Lock()
	defer t.lock.Unlock()

	// find targetNode
	parent := t.root
	var targetNode *Node
	for i, part := range parts {
		child, ok := parent.children[part]
		if !ok {
			return ErrNotFound
		}
		if i == len(parts)-1 {
			targetNode = child
			delete(parent.children, part)
			break
		}
		parent = child
	}

	// assign the targetNode to dstParent
	itemName := parts[len(parts)-1]
	parent = t.root
	for i, part := range dstParts {
		child, ok := parent.children[part]
		if !ok {
			return ErrNotFound
		}
		if i == len(dstParts)-1 {
			child.children[itemName] = targetNode
			targetNode.parent = child
			break
		}
		parent = child
	}

	return nil
}

// GetPath traverses the path which is from the targetNode to root, and construct the pathname
func (t *Tree) GetPath(targetNode *Node) (string, error) {
	node := targetNode
	parts := []string{}
	for node != nil {
		parts = append(parts, node.name)
		node = node.parent
	}

	if parts[len(parts)-1] != "" {
		// the path is already deleted so it can not reach to root
		return "", ErrNotFound
	}
	parts = parts[:len(parts)-1]
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, t.PathSeparator), nil
}

type Row struct {
	Id       int64
	ParentId int64
	Name     string
}

// Marshal marshals the tree into string rows
func (t *Tree) Marshal() chan string {
	c := make(chan string, 1024)
	// marshal tree
	treeBytes, err := json.Marshal(t)
	if err != nil {
		t.err = err
		close(c)
		return c
	}
	c <- string(treeBytes)

	worker := func() {
		defer close(c)

		queue := []*Node{
			t.root,
		}
		for len(queue) > 0 {
			node := queue[0]
			queue = queue[1:]

			for childName, child := range node.children {
				queue = append(queue, child)
				row := &Row{
					Id:       child.id,
					ParentId: node.id,
					Name:     childName,
				}
				rowBytes, err := json.Marshal(row)
				if err != nil {
					t.err = err
					return
				}

				c <- string(rowBytes)
			}
		}
	}
	go worker()

	return c
}

// Unmarshal restores the tree from rows
func (t *Tree) Unmarshal(rows chan string) {
	// restore tree
	rowStr := <-rows
	tmpTree := &Tree{}
	err := json.Unmarshal([]byte(rowStr), tmpTree)
	if err != nil {
		t.err = err
		return
	}
	t.MaxId = tmpTree.MaxId
	t.PathSeparator = tmpTree.PathSeparator

	// restore nodes
	row := &Row{}
	// TODO: currently all nodes are saved in mem
	// since each node only saves 2 integers
	createdNodes := map[int64]*Node{
		int64(0): t.root,
	}
	for rowStr := range rows {
		err = json.Unmarshal([]byte(rowStr), row)
		if err != nil {
			t.err = err
			return
		}

		parentNode := createdNodes[row.ParentId]
		newNode := &Node{
			id:       row.Id,
			name:     row.Name,
			parent:   parentNode,
			children: map[string]*Node{},
		}

		parentNode.children[row.Name] = newNode
		createdNodes[row.Id] = newNode
	}
}

// Err returns the last err (if it exists)
func (t *Tree) Err() error {
	err := t.err
	t.err = nil
	return err
}

func (t *Tree) String() string {
	result := ""
	queue := []*Node{t.root}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		if node != nil {
			result += fmt.Sprintln("node not found")
			continue
		}

		result += fmt.Sprintf("id(%d) name(%s) parent(%d)\n", node.id, node.name, node.parent.id)
		for _, child := range node.children {
			queue = append(queue, child)
			result += fmt.Sprintf("\tid(%d) name(%s)\n", node.id, node.name)
		}
	}

	return result
}
