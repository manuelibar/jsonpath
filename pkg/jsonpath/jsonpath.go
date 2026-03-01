// Package jsonpath provides JSONPath parsing, compilation, and evaluation.
//
// It exposes a [Query] for compiling and walking JSON trees, a [PathBuilder]
// for constructing JSONPath strings incrementally during tree traversal,
// and safe defaults through [Limits], [MaxDepth], [MaxPathLength], and
// [MaxPathCount].
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

	"github.com/mibar/jsonpath/internal/core"
)

// ---------------------------------------------------------------------------
// Re-exports from internal/core
// ---------------------------------------------------------------------------

type (
	// Query describes a set of JSONPath expressions and a mode (include or exclude).
	// Queries are lazy: paths are stored as strings until Compile() or first Walk().
	Query = core.Query
	// Mode selects between include and exclude behaviour.
	Mode = core.Mode
	// ParseError describes a syntax error in a JSONPath expression.
	ParseError = core.ParseError
	// DepthError is returned when a JSON document exceeds the configured maximum depth.
	DepthError = core.DepthError
	// Limits configures safety limits for JSON tree walking.
	//
	// A nil field means "use the default constant" — safe by default.
	// To explicitly disable a check, set the field to Ptr(0).
	// Use [NoLimits] to disable all limits at once.
	Limits = core.Limits
)

const (
	// ModeInclude selects include behaviour — keep only matched paths.
	ModeInclude = core.ModeInclude
	// ModeExclude selects exclude behaviour — remove matched paths, keep everything else.
	ModeExclude = core.ModeExclude
)

const (
	// MaxDepth is the default maximum JSON nesting depth (1 000).
	MaxDepth = core.MaxDepth
	// MaxPathLength is the default maximum byte length of a single JSONPath expression (10 000).
	MaxPathLength = core.MaxPathLength
	// MaxPathCount is the default maximum number of JSONPath expressions per query (1 000).
	MaxPathCount = core.MaxPathCount
)

// Ptr returns a pointer to v. Useful for constructing [Limits] with custom
// values inline:
//
//	jsonpath.Limits{MaxDepth: jsonpath.Ptr(200)}
func Ptr[T any](v T) *T { return &v }

// DefaultLimits returns the default safety limits with each field set
// explicitly to its package-level constant. Equivalent to the zero-value
// Limits{} but makes the values visible for inspection or logging.
func DefaultLimits() Limits { return core.DefaultLimits() }

// NoLimits returns a [Limits] value that explicitly disables all safety
// checks. Use this only when you fully trust both the JSON input and the
// JSONPath expressions — for example, in tests or internal pipelines.
func NoLimits() Limits { return core.NoLimits() }

// Include returns an include-mode [Query] for the given JSONPath expressions.
func Include(paths ...string) Query { return core.Include(paths...) }

// Exclude returns an exclude-mode [Query] for the given JSONPath expressions.
func Exclude(paths ...string) Query { return core.Exclude(paths...) }

// MustCompile is like [Query.Compile] but panics on error.
func MustCompile(q Query) Query { return core.MustCompile(q) }

// ---------------------------------------------------------------------------
// PathBuilder
// ---------------------------------------------------------------------------

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
