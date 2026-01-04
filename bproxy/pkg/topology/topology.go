package topology

import (
	"fmt"
	"sync"
	"time"
)

type NodeInfo struct {
	ID           string
	Hostname     string
	LocalIPs     []string
	OS           string
	Arch         string
	ParentID     string
	Children     []string
	LastSeen     time.Time
	IsActive     bool
}

type Topology struct {
	mu    sync.RWMutex
	nodes map[string]*NodeInfo
	edges map[string][]string
}

func NewTopology() *Topology {
	return &Topology{
		nodes: make(map[string]*NodeInfo),
		edges: make(map[string][]string),
	}
}

func (t *Topology) AddNode(id, hostname string, localIPs []string, os, arch string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if node, exists := t.nodes[id]; exists {
		node.LastSeen = time.Now()
		node.IsActive = true
		return
	}

	t.nodes[id] = &NodeInfo{
		ID:       id,
		Hostname: hostname,
		LocalIPs: localIPs,
		OS:       os,
		Arch:     arch,
		Children: []string{},
		LastSeen: time.Now(),
		IsActive: true,
	}
}

func (t *Topology) RemoveNode(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if node, exists := t.nodes[id]; exists {
		node.IsActive = false
	}

	delete(t.edges, id)
	for parent := range t.edges {
		children := t.edges[parent]
		for i, child := range children {
			if child == id {
				t.edges[parent] = append(children[:i], children[i+1:]...)
				break
			}
		}
	}
}

func (t *Topology) AddEdge(parentID, childID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.nodes[parentID]; !exists {
		return fmt.Errorf("parent node %s not found", parentID)
	}
	if _, exists := t.nodes[childID]; !exists {
		return fmt.Errorf("child node %s not found", childID)
	}

	if t.edges[parentID] == nil {
		t.edges[parentID] = []string{}
	}

	for _, child := range t.edges[parentID] {
		if child == childID {
			return nil
		}
	}

	t.edges[parentID] = append(t.edges[parentID], childID)
	t.nodes[childID].ParentID = parentID
	t.nodes[parentID].Children = append(t.nodes[parentID].Children, childID)

	return nil
}

func (t *Topology) GetNode(id string) (*NodeInfo, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	node, exists := t.nodes[id]
	return node, exists
}

func (t *Topology) GetAllNodes() []*NodeInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()

	nodes := make([]*NodeInfo, 0, len(t.nodes))
	for _, node := range t.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

func (t *Topology) GetPath(targetID string) []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	path := []string{}
	current := targetID

	for current != "" {
		path = append([]string{current}, path...)
		if node, exists := t.nodes[current]; exists {
			current = node.ParentID
		} else {
			break
		}
	}

	return path
}

func (t *Topology) UpdateHeartbeat(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if node, exists := t.nodes[id]; exists {
		node.LastSeen = time.Now()
		node.IsActive = true
	}
}

func (t *Topology) CheckDeadNodes(timeout time.Duration) []string {
	t.mu.Lock()
	defer t.mu.Unlock()

	deadNodes := []string{}
	now := time.Now()

	for id, node := range t.nodes {
		if node.IsActive && now.Sub(node.LastSeen) > timeout {
			node.IsActive = false
			deadNodes = append(deadNodes, id)
		}
	}

	return deadNodes
}