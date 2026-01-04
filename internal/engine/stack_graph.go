package engine

import "slices"

// StackNode represents a branch within a stack snapshot.
type StackNode struct {
	Branch   Branch
	Parent   string
	Children []string
	Depth    int
	IsTrunk  bool
}

// StackGraph is an immutable snapshot of the branch stack relationships.
// It is built once from the engine and then used for traversals/rendering without
// further engine calls.
type StackGraph struct {
	Nodes        map[string]*StackNode
	Roots        []string
	Current      string
	Trunk        string
	SortStrategy SortStrategy
}

// BuildStackGraph constructs a StackGraph using the provided engine reader and sorting strategy.
// The optional filter is applied to branches; filtered-out branches are omitted along with their subtrees.
func BuildStackGraph(eng BranchReader, strategy SortStrategy, filter func(Branch) bool) *StackGraph {
	branches := eng.AllBranches()

	trunk := eng.Trunk().GetName()
	current := ""
	if cb := eng.CurrentBranch(); cb != nil {
		current = cb.GetName()
	}

	allowed := make(map[string]Branch)
	for _, b := range branches {
		if filter != nil && !filter(b) {
			continue
		}
		allowed[b.GetName()] = b
	}

	graph := &StackGraph{
		Nodes:        make(map[string]*StackNode),
		Current:      current,
		Trunk:        trunk,
		SortStrategy: strategy,
	}

	// Seed nodes with parent references (only if parent is also allowed)
	for name, branch := range allowed {
		parentName := ""
		if parent := branch.GetParent(); parent != nil {
			if _, ok := allowed[parent.GetName()]; ok {
				parentName = parent.GetName()
			}
		}

		graph.Nodes[name] = &StackNode{
			Branch:  branch,
			Parent:  parentName,
			IsTrunk: name == trunk,
		}
	}

	// Populate children using the requested strategy, honoring the filter set.
	for _, node := range graph.Nodes {
		children := eng.GetChildrenWithStrategy(node.Branch, strategy)
		for _, child := range children {
			childName := child.GetName()
			if _, ok := graph.Nodes[childName]; !ok {
				continue
			}
			node.Children = append(node.Children, childName)
		}
	}

	// Determine roots (nodes without an allowed parent)
	for name, node := range graph.Nodes {
		if node.Parent == "" {
			graph.Roots = append(graph.Roots, name)
		}
	}

	// Compute depth for each node (distance from root)
	depthCache := make(map[string]int)
	var computeDepth func(string) int
	computeDepth = func(name string) int {
		if d, ok := depthCache[name]; ok {
			return d
		}
		node := graph.Nodes[name]
		if node == nil || node.Parent == "" {
			depthCache[name] = 0
			return 0
		}
		depthCache[name] = computeDepth(node.Parent) + 1
		return depthCache[name]
	}
	for name, node := range graph.Nodes {
		node.Depth = computeDepth(name)
	}

	// Sort roots for deterministic traversal
	slices.Sort(graph.Roots)

	return graph
}

// Node returns the StackNode for the given branch name.
func (g *StackGraph) Node(name string) *StackNode {
	return g.Nodes[name]
}

// Children returns the child branch names for the given branch.
func (g *StackGraph) Children(name string) []string {
	if node := g.Nodes[name]; node != nil {
		return node.Children
	}
	return nil
}

// Parent returns the parent branch name (empty string if none).
func (g *StackGraph) Parent(name string) string {
	if node := g.Nodes[name]; node != nil {
		return node.Parent
	}
	return ""
}

// Range returns branches matching the provided StackRange, ordered the same as the legacy
// GetRelativeStack implementation: ancestors (oldest to nearest), current, then descendants.
// Descendants are traversed depth-first using the graph's pre-sorted children.
func (g *StackGraph) Range(start string, rng StackRange) []Branch {
	startNode := g.Nodes[start]
	if startNode == nil {
		return nil
	}

	var result []Branch

	// Ancestors (excluding trunk)
	if rng.RecursiveParents {
		current := startNode.Parent
		var ancestors []Branch
		for current != "" && current != g.Trunk {
			node := g.Nodes[current]
			if node == nil {
				break
			}
			ancestors = append(ancestors, node.Branch)
			current = node.Parent
		}
		// Reverse to go from trunk-ward to the starting branch
		for i, j := 0, len(ancestors)-1; i < j; i, j = i+1, j-1 {
			ancestors[i], ancestors[j] = ancestors[j], ancestors[i]
		}
		result = append(result, ancestors...)
	}

	// Current branch
	if rng.IncludeCurrent {
		result = append(result, startNode.Branch)
	}

	// Descendants (depth-first, ordered by pre-sorted children)
	if rng.RecursiveChildren {
		visited := map[string]bool{start: true}
		var collectDescendants func(string)
		collectDescendants = func(name string) {
			node := g.Nodes[name]
			if node == nil {
				return
			}
			for _, childName := range node.Children {
				if visited[childName] {
					continue
				}
				visited[childName] = true
				if child := g.Nodes[childName]; child != nil {
					result = append(result, child.Branch)
					collectDescendants(childName)
				}
			}
		}
		collectDescendants(start)
	}

	return result
}
