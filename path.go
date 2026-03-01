// Package jsonpath provides JSONPath parsing, compilation, and evaluation.
//
// It includes a recursive descent parser ([parser.ParsePath]), a [Query]
// type for compiling and walking JSON trees, and a [PathBuilder] for
// constructing JSONPath strings incrementally during JSON tree traversal.
//
// # Quick start
//
//	q, err := jsonpath.Include("$.name", "$.email").Compile()
//	result, err := q.Walk(tree) // tree is any from json.Unmarshal
//
// # Safety
//
// Queries apply sensible limits by default ([MaxDepth], [MaxPathLength],
// [MaxPathCount]). Use [NoLimits] to disable all checks, or [Limits] to
// customise individual thresholds.
package jsonpath

import (
	"fmt"
	"strings"
)

// PathBuilder constructs JSONPath strings incrementally during tree traversal.
// It maintains a stack of segments to avoid repeated string concatenation.
type PathBuilder struct {
	segments []string
}

// NewPathBuilder creates a PathBuilder rooted at "$".
func NewPathBuilder() *PathBuilder {
	return &PathBuilder{segments: []string{"$"}}
}

// Child returns a new PathBuilder with a child object key appended.
// Keys requiring bracket notation (containing dots, brackets, spaces, or
// starting with a digit) use the ["key"] form.
func (pb *PathBuilder) Child(key string) *PathBuilder {
	next := make([]string, len(pb.segments), len(pb.segments)+1)
	copy(next, pb.segments)
	if needsBracket(key) {
		next = append(next, fmt.Sprintf("[%q]", key))
	} else {
		next = append(next, "."+key)
	}
	return &PathBuilder{segments: next}
}

// Index returns a new PathBuilder with an array index appended.
func (pb *PathBuilder) Index(i int) *PathBuilder {
	next := make([]string, len(pb.segments), len(pb.segments)+1)
	copy(next, pb.segments)
	next = append(next, fmt.Sprintf("[%d]", i))
	return &PathBuilder{segments: next}
}

// String returns the full JSONPath string.
func (pb *PathBuilder) String() string {
	return strings.Join(pb.segments, "")
}

// needsBracket reports whether a key requires bracket notation in JSONPath.
func needsBracket(key string) bool {
	if len(key) == 0 {
		return true
	}
	for i, r := range key {
		if i == 0 && r >= '0' && r <= '9' {
			return true
		}
		switch r {
		case '.', '[', ']', ' ', '"', '\'', '*', ',', ':', '?', '@', '(', ')':
			return true
		}
	}
	return false
}
