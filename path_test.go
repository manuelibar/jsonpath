package jsonpath

import "testing"

func TestPathBuilderRoot(t *testing.T) {
	pb := NewPathBuilder()
	if got := pb.String(); got != "$" {
		t.Errorf("root path = %q, want %q", got, "$")
	}
}

func TestPathBuilderChild(t *testing.T) {
	pb := NewPathBuilder().Child("users")
	if got := pb.String(); got != "$.users" {
		t.Errorf("child path = %q, want %q", got, "$.users")
	}
}

func TestPathBuilderNestedChild(t *testing.T) {
	pb := NewPathBuilder().Child("users").Child("name")
	if got := pb.String(); got != "$.users.name" {
		t.Errorf("nested path = %q, want %q", got, "$.users.name")
	}
}

func TestPathBuilderIndex(t *testing.T) {
	pb := NewPathBuilder().Child("users").Index(0)
	if got := pb.String(); got != "$.users[0]" {
		t.Errorf("index path = %q, want %q", got, "$.users[0]")
	}
}

func TestPathBuilderIndexThenChild(t *testing.T) {
	pb := NewPathBuilder().Child("users").Index(2).Child("name")
	if got := pb.String(); got != "$.users[2].name" {
		t.Errorf("index+child path = %q, want %q", got, "$.users[2].name")
	}
}

func TestPathBuilderBracketNotationDot(t *testing.T) {
	pb := NewPathBuilder().Child("a.b")
	if got := pb.String(); got != `$["a.b"]` {
		t.Errorf("dot key = %q, want %q", got, `$["a.b"]`)
	}
}

func TestPathBuilderBracketNotationSpace(t *testing.T) {
	pb := NewPathBuilder().Child("first name")
	if got := pb.String(); got != `$["first name"]` {
		t.Errorf("space key = %q, want %q", got, `$["first name"]`)
	}
}

func TestPathBuilderBracketNotationDigitStart(t *testing.T) {
	pb := NewPathBuilder().Child("0invalid")
	if got := pb.String(); got != `$["0invalid"]` {
		t.Errorf("digit-start key = %q, want %q", got, `$["0invalid"]`)
	}
}

func TestPathBuilderBracketNotationEmpty(t *testing.T) {
	pb := NewPathBuilder().Child("")
	if got := pb.String(); got != `$[""]` {
		t.Errorf("empty key = %q, want %q", got, `$[""]`)
	}
}

func TestPathBuilderImmutability(t *testing.T) {
	parent := NewPathBuilder().Child("a")
	child1 := parent.Child("b")
	child2 := parent.Child("c")

	if got := parent.String(); got != "$.a" {
		t.Errorf("parent mutated = %q, want %q", got, "$.a")
	}
	if got := child1.String(); got != "$.a.b" {
		t.Errorf("child1 = %q, want %q", got, "$.a.b")
	}
	if got := child2.String(); got != "$.a.c" {
		t.Errorf("child2 = %q, want %q", got, "$.a.c")
	}
}
