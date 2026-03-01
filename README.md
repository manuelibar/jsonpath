# jsonpath

JSONPath parsing, compilation, and evaluation for Go. Zero external dependencies.

## Install

```bash
go get github.com/mibar/jsonpath/pkg/jsonpath
```

## Usage

```go
import "github.com/mibar/jsonpath/pkg/jsonpath"

// Compile once, reuse across many documents
q, err := jsonpath.Include("$.name", "$.email").Compile()

// Walk a parsed JSON tree (any from json.Unmarshal)
result, err := q.Walk(tree) // returns any
```

### Include / exclude modes

```go
// Keep only matched paths
q := jsonpath.Include("$.user.name", "$.user.email")

// Remove matched paths, keep everything else
q := jsonpath.Exclude("$..password", "$..secret")
```

### PathBuilder

Build JSONPath strings incrementally during tree traversal:

```go
pb := jsonpath.NewPathBuilder()       // "$"
pb.Child("users")                     // "$.users"
pb.Child("users").Index(2)            // "$.users[2]"
pb.Child("users").Index(2).Child("name") // "$.users[2].name"
```

Keys with special characters use bracket notation automatically:

```go
pb.Child("first name") // "$[\"first name\"]"
pb.Child("a.b")        // "$[\"a.b\"]"
```

### Safety limits

Queries apply sensible limits by default:

| Limit | Default | Purpose |
|-------|---------|---------|
| `MaxDepth` | 1 000 | Prevents stack exhaustion on deeply nested JSON |
| `MaxPathLength` | 10 000 bytes | Caps parser work per expression |
| `MaxPathCount` | 1 000 | Bounds trie memory |

Customise or disable:

```go
q := jsonpath.Include("$.name").WithLimits(jsonpath.Limits{
    MaxDepth: jsonpath.Ptr(200),
})

// Disable all limits (trusted input only)
q := jsonpath.Include("$.name").WithLimits(jsonpath.NoLimits())
```

## Package layout

```
pkg/jsonpath/          Public API — Query, PathBuilder, Limits, Mode
internal/core/         Query, trie, walker implementation
internal/parser/       Recursive descent JSONPath parser (AST)
```

## Supported syntax

| Syntax | Example | Description |
|--------|---------|-------------|
| Dot notation | `$.name` | Object key |
| Bracket notation | `$['name']`, `$["name"]` | Object key (quoted) |
| Index | `$[0]`, `$[-1]` | Array index (negative supported) |
| Wildcard | `$.*`, `$[*]` | All keys or indices |
| Recursive descent | `$..name` | Match at any depth |
| Slice | `$[0:5]`, `$[::2]` | Array slice per RFC 9535 |
| Multi-selector | `$[0,2,4]`, `$['a','b']` | Multiple selectors |
