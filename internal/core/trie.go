package core

import (
	"github.com/mibar/jsonpath/internal/parser"
)

// trie.go implements a compiled prefix trie for efficient multi-pattern matching
// of JSONPath expressions.
//
// See the package-level documentation in query.go for the NFA/trie model.

type trieNode struct {
	names     map[string]*trieNode
	indexes   map[int]*trieNode
	wildcard  *trieNode
	slices    []*sliceChild
	epsilon   *trieNode
	accepting bool

	namesMerged map[string]*trieNode
	finalized   bool
}

type sliceChild struct {
	sel  parser.SliceSelector
	node *trieNode
}

func newTrieNode() *trieNode {
	return &trieNode{}
}

func buildTrie(paths []*parser.Path) *trieNode {
	root := newTrieNode()
	for _, p := range paths {
		root.insert(p.Segments, 0)
	}
	root.finalize()
	return root
}

func (n *trieNode) finalize() {
	if n.finalized {
		return
	}
	n.finalized = true

	for _, child := range n.names {
		child.finalize()
	}
	for _, child := range n.indexes {
		child.finalize()
	}
	if n.wildcard != nil {
		n.wildcard.finalize()
	}
	for _, sc := range n.slices {
		sc.node.finalize()
	}
	if n.epsilon != nil {
		n.epsilon.finalize()
	}

	if len(n.names) > 0 && n.wildcard != nil {
		n.namesMerged = make(map[string]*trieNode, len(n.names))
		for k, named := range n.names {
			n.namesMerged[k] = mergeNodes([]*trieNode{named, n.wildcard})
		}
	}
}

func (n *trieNode) insert(segments []parser.Segment, segIdx int) {
	if segIdx >= len(segments) {
		n.accepting = true
		return
	}

	seg := segments[segIdx]

	if seg.Descendant {
		if n.epsilon == nil {
			n.epsilon = newTrieNode()
		}
		nonDescSeg := seg.WithoutDescendant()
		remaining := make([]parser.Segment, 0, len(segments)-segIdx)
		remaining = append(remaining, nonDescSeg)
		remaining = append(remaining, segments[segIdx+1:]...)
		n.epsilon.insert(remaining, 0)
		return
	}

	for _, sel := range seg.Selectors {
		child := n.getOrCreateChild(sel)
		child.insert(segments, segIdx+1)
	}
}

func (n *trieNode) getOrCreateChild(sel parser.Selector) *trieNode {
	switch s := sel.(type) {
	case parser.NameSelector:
		if n.names == nil {
			n.names = make(map[string]*trieNode)
		}
		if child, ok := n.names[s.Name]; ok {
			return child
		}
		child := newTrieNode()
		n.names[s.Name] = child
		return child

	case parser.IndexSelector:
		if n.indexes == nil {
			n.indexes = make(map[int]*trieNode)
		}
		if child, ok := n.indexes[s.Index]; ok {
			return child
		}
		child := newTrieNode()
		n.indexes[s.Index] = child
		return child

	case parser.WildcardSelector:
		if n.wildcard == nil {
			n.wildcard = newTrieNode()
		}
		return n.wildcard

	case parser.SliceSelector:
		for _, sc := range n.slices {
			if sliceEqual(sc.sel, s) {
				return sc.node
			}
		}
		child := newTrieNode()
		n.slices = append(n.slices, &sliceChild{sel: s, node: child})
		return child
	}

	return newTrieNode()
}

func (n *trieNode) match(key string) *trieNode {
	if n.namesMerged != nil {
		if child, ok := n.namesMerged[key]; ok {
			return child
		}
		return n.wildcard
	}

	named := n.names[key]
	if n.wildcard == nil {
		return named
	}
	if named == nil {
		return n.wildcard
	}
	return mergeNodes([]*trieNode{named, n.wildcard})
}

func (n *trieNode) matchIndex(idx, arrLen int) *trieNode {
	if n.wildcard == nil && len(n.slices) == 0 {
		if n.indexes == nil {
			return nil
		}
		child := n.indexes[idx]
		negIdx := idx - arrLen
		if negIdx != idx {
			if negChild := n.indexes[negIdx]; negChild != nil {
				if child == nil {
					return negChild
				}
				return mergeNodes([]*trieNode{child, negChild})
			}
		}
		return child
	}

	var matches []*trieNode

	if n.indexes != nil {
		if child, ok := n.indexes[idx]; ok {
			matches = append(matches, child)
		}
		negIdx := idx - arrLen
		if negIdx != idx {
			if child, ok := n.indexes[negIdx]; ok {
				matches = append(matches, child)
			}
		}
	}

	if n.wildcard != nil {
		matches = append(matches, n.wildcard)
	}

	for _, sc := range n.slices {
		if sc.sel.Matches(idx, arrLen) {
			matches = append(matches, sc.node)
		}
	}

	return mergeNodes(matches)
}

func mergePair(a, b *trieNode) *trieNode {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	return mergeNodes([]*trieNode{a, b})
}

func mergeNodes(nodes []*trieNode) *trieNode {
	switch len(nodes) {
	case 0:
		return nil
	case 1:
		return nodes[0]
	}

	merged := newTrieNode()
	for _, node := range nodes {
		if node.accepting {
			merged.accepting = true
		}
		for k, v := range node.names {
			if merged.names == nil {
				merged.names = make(map[string]*trieNode)
			}
			if existing, ok := merged.names[k]; ok {
				merged.names[k] = mergeNodes([]*trieNode{existing, v})
			} else {
				merged.names[k] = v
			}
		}
		for k, v := range node.indexes {
			if merged.indexes == nil {
				merged.indexes = make(map[int]*trieNode)
			}
			if existing, ok := merged.indexes[k]; ok {
				merged.indexes[k] = mergeNodes([]*trieNode{existing, v})
			} else {
				merged.indexes[k] = v
			}
		}
		if node.wildcard != nil {
			if merged.wildcard != nil {
				merged.wildcard = mergeNodes([]*trieNode{merged.wildcard, node.wildcard})
			} else {
				merged.wildcard = node.wildcard
			}
		}
		merged.slices = append(merged.slices, node.slices...)
		if node.epsilon != nil {
			if merged.epsilon != nil {
				merged.epsilon = mergeNodes([]*trieNode{merged.epsilon, node.epsilon})
			} else {
				merged.epsilon = node.epsilon
			}
		}
	}
	return merged
}

func sliceEqual(a, b parser.SliceSelector) bool {
	return intPtrEqual(a.Start, b.Start) && intPtrEqual(a.End, b.End) && intPtrEqual(a.Step, b.Step)
}

func intPtrEqual(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
