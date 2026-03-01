package core

import "fmt"

// MaxDepth is the default maximum recursion depth the walker will traverse.
// Applied automatically when [Limits].MaxDepth is nil (the zero-value default).
const MaxDepth = 1000

// DepthError is returned when the walker exceeds the configured maximum depth.
type DepthError struct {
	Depth    int
	MaxDepth int
}

func (e *DepthError) Error() string {
	return fmt.Sprintf("maximum JSON depth %d exceeded at depth %d", e.MaxDepth, e.Depth)
}

type walker struct {
	include  bool
	maxDepth int
}

func newWalker(mode Mode, maxDepth int) walker {
	return walker{include: mode == ModeInclude, maxDepth: maxDepth}
}

func (w walker) walk(node any, trie *trieNode, depth int) (any, error) {
	if trie == nil {
		if w.include {
			return nil, nil
		}
		return node, nil
	}
	if w.maxDepth > 0 && depth > w.maxDepth {
		return nil, &DepthError{Depth: depth, MaxDepth: w.maxDepth}
	}
	if trie.accepting {
		if w.include {
			return node, nil
		}
		return nil, nil
	}

	switch v := node.(type) {
	case map[string]any:
		return w.walkObject(v, trie, depth)
	case []any:
		return w.walkArray(v, trie, depth)
	default:
		if w.include {
			return nil, nil
		}
		return node, nil
	}
}

func (w walker) walkObject(obj map[string]any, trie *trieNode, depth int) (any, error) {
	eps := trie.epsilon
	var result map[string]any
	if w.include {
		result = make(map[string]any)
	} else {
		result = make(map[string]any, len(obj))
	}

	for key, val := range obj {
		child := trie.match(key)
		if eps != nil {
			child = mergePair(child, eps.match(key))
		}

		r, err := w.resolveMatch(val, child, depth)
		if err != nil {
			return nil, err
		}

		if eps != nil {
			if w.include {
				found, ferr := w.walkSearchEpsilon(val, eps, depth+1)
				if ferr != nil {
					return nil, ferr
				}
				r = mergeValues(r, found)
			} else if r != nil {
				r, err = w.walkFilterEpsilon(r, eps, depth+1)
				if err != nil {
					return nil, err
				}
			}
		}

		if r != nil {
			result[key] = r
		}
	}

	if w.include && len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

func (w walker) walkArray(arr []any, trie *trieNode, depth int) (any, error) {
	eps := trie.epsilon
	arrLen := len(arr)
	var result []any
	if !w.include {
		result = make([]any, 0, arrLen)
	}

	for idx, val := range arr {
		child := trie.matchIndex(idx, arrLen)
		if eps != nil {
			child = mergePair(child, eps.matchIndex(idx, arrLen))
		}

		r, err := w.resolveMatch(val, child, depth)
		if err != nil {
			return nil, err
		}

		if eps != nil {
			if w.include {
				found, ferr := w.walkSearchEpsilon(val, eps, depth+1)
				if ferr != nil {
					return nil, ferr
				}
				r = mergeValues(r, found)
			} else if r != nil {
				r, err = w.walkFilterEpsilon(r, eps, depth+1)
				if err != nil {
					return nil, err
				}
			}
		}

		if r != nil {
			result = append(result, r)
		}
	}

	if w.include && len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

func (w walker) resolveMatch(val any, child *trieNode, depth int) (any, error) {
	if child == nil {
		if w.include {
			return nil, nil
		}
		return val, nil
	}
	if child.accepting {
		if w.include {
			return val, nil
		}
		return nil, nil
	}
	return w.walk(val, child, depth+1)
}

func (w walker) walkSearchEpsilon(node any, epsTrie *trieNode, depth int) (any, error) {
	if w.maxDepth > 0 && depth > w.maxDepth {
		return nil, &DepthError{Depth: depth, MaxDepth: w.maxDepth}
	}

	switch v := node.(type) {
	case map[string]any:
		result := make(map[string]any)
		for key, val := range v {
			childTrie := epsTrie.match(key)
			if childTrie != nil {
				childResult, err := w.walk(val, childTrie, depth+1)
				if err != nil {
					return nil, err
				}
				if childResult != nil {
					result[key] = childResult
				}
			}
			epsResult, err := w.walkSearchEpsilon(val, epsTrie, depth+1)
			if err != nil {
				return nil, err
			}
			if epsResult != nil {
				if existing, ok := result[key]; ok {
					result[key] = mergeValues(existing, epsResult)
				} else {
					result[key] = epsResult
				}
			}
		}
		if len(result) == 0 {
			return nil, nil
		}
		return result, nil

	case []any:
		var result []any
		arrLen := len(v)
		for idx, val := range v {
			childTrie := epsTrie.matchIndex(idx, arrLen)
			var childResult any
			var err error
			if childTrie != nil {
				childResult, err = w.walk(val, childTrie, depth+1)
				if err != nil {
					return nil, err
				}
			}
			epsResult, err := w.walkSearchEpsilon(val, epsTrie, depth+1)
			if err != nil {
				return nil, err
			}
			merged := mergeValues(childResult, epsResult)
			if merged != nil {
				result = append(result, merged)
			}
		}
		if len(result) == 0 {
			return nil, nil
		}
		return result, nil
	}

	return nil, nil
}

func (w walker) walkFilterEpsilon(node any, epsTrie *trieNode, depth int) (any, error) {
	if w.maxDepth > 0 && depth > w.maxDepth {
		return nil, &DepthError{Depth: depth, MaxDepth: w.maxDepth}
	}

	switch v := node.(type) {
	case map[string]any:
		result := make(map[string]any, len(v))
		for key, val := range v {
			childTrie := epsTrie.match(key)
			if childTrie != nil && childTrie.accepting {
				continue
			}
			if childTrie != nil {
				childResult, err := w.walk(val, childTrie, depth+1)
				if err != nil {
					return nil, err
				}
				epsResult, err := w.walkFilterEpsilon(childResult, epsTrie, depth+1)
				if err != nil {
					return nil, err
				}
				if epsResult != nil {
					result[key] = epsResult
				}
			} else {
				epsResult, err := w.walkFilterEpsilon(val, epsTrie, depth+1)
				if err != nil {
					return nil, err
				}
				result[key] = epsResult
			}
		}
		return result, nil

	case []any:
		result := make([]any, 0, len(v))
		arrLen := len(v)
		for idx, val := range v {
			childTrie := epsTrie.matchIndex(idx, arrLen)
			if childTrie != nil && childTrie.accepting {
				continue
			}
			if childTrie != nil {
				childResult, err := w.walk(val, childTrie, depth+1)
				if err != nil {
					return nil, err
				}
				epsResult, err := w.walkFilterEpsilon(childResult, epsTrie, depth+1)
				if err != nil {
					return nil, err
				}
				result = append(result, epsResult)
			} else {
				epsResult, err := w.walkFilterEpsilon(val, epsTrie, depth+1)
				if err != nil {
					return nil, err
				}
				result = append(result, epsResult)
			}
		}
		return result, nil
	}

	return node, nil
}

func mergeValues(a, b any) any {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}

	aObj, aIsObj := a.(map[string]any)
	bObj, bIsObj := b.(map[string]any)
	if aIsObj && bIsObj {
		merged := make(map[string]any, len(aObj)+len(bObj))
		for k, v := range aObj {
			merged[k] = v
		}
		for k, v := range bObj {
			if existing, ok := merged[k]; ok {
				merged[k] = mergeValues(existing, v)
			} else {
				merged[k] = v
			}
		}
		return merged
	}

	return a
}
