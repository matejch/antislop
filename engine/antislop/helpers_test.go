package antislop

import "testing"

func TestIsWordChar(t *testing.T) {
	for _, c := range "azAZ09_" {
		if !isWordChar(byte(c)) {
			t.Errorf("isWordChar(%c) should be true", c)
		}
	}
	for _, c := range " .-!@#$%^&*()" {
		if isWordChar(byte(c)) {
			t.Errorf("isWordChar(%c) should be false", c)
		}
	}
}

func TestHasWordBoundaryBefore(t *testing.T) {
	if !hasWordBoundaryBefore("panic()", 0) {
		t.Error("position 0 should be word boundary")
	}
	if !hasWordBoundaryBefore(" panic", 1) {
		t.Error("space before should be word boundary")
	}
	if hasWordBoundaryBefore("xpanic", 1) {
		t.Error("letter before should not be word boundary")
	}
}

func TestContainsWordThenParen(t *testing.T) {
	tests := []struct {
		line, word string
		expect     int
	}{
		{"panic(x)", "panic", 0},
		{"  panic (x)", "panic", 2},
		{"nopanic(x)", "panic", -1},
		{"fmt.Sprintf(x)", "Sprintf", 4},
		{"foo()", "foo", 0},
		{"barfoo()", "foo", -1},
	}
	for _, tc := range tests {
		got := containsWordThenParen(tc.line, tc.word)
		if got != tc.expect {
			t.Errorf("containsWordThenParen(%q, %q) = %d, want %d", tc.line, tc.word, got, tc.expect)
		}
	}
}

func TestExtractFirstWord(t *testing.T) {
	tests := []struct {
		input, expect string
	}{
		{"hello world", "hello"},
		{"  hello", "hello"},
		{"word", "word"},
		{"", ""},
		{"  ", ""},
	}
	for _, tc := range tests {
		got := extractFirstWord(tc.input)
		if got != tc.expect {
			t.Errorf("extractFirstWord(%q) = %q, want %q", tc.input, got, tc.expect)
		}
	}
}

func TestExtractWordAt(t *testing.T) {
	if extractWordAt("hello world", 0) != "hello" {
		t.Error("expected 'hello'")
	}
	if extractWordAt("hello world", 6) != "world" {
		t.Error("expected 'world'")
	}
	if extractWordAt("", 0) != "" {
		t.Error("expected empty")
	}
}

func TestStartsWithKeyword(t *testing.T) {
	kws := []string{"func", "type", "var"}
	if startsWithKeyword("func main()", kws) != "func" {
		t.Error("expected func")
	}
	if startsWithKeyword("function()", kws) != "" {
		t.Error("function should not match func")
	}
	if startsWithKeyword("var x", kws) != "var" {
		t.Error("expected var")
	}
	if startsWithKeyword("variable", kws) != "" {
		t.Error("variable should not match var")
	}
}

func TestContainsAnyWord(t *testing.T) {
	set := map[string]bool{"because": true, "since": true}
	if !containsAnyWord("this is because of that", set) {
		t.Error("should find 'because'")
	}
	if containsAnyWord("this is fine", set) {
		t.Error("should not find anything")
	}
	if !containsAnyWord("since yesterday", set) {
		t.Error("should find 'since'")
	}
}

func TestIsPyDefLine(t *testing.T) {
	tests := []struct {
		line       string
		expectName string
		expectOk   bool
	}{
		{"def foo():", "foo", true},
		{"  def bar(x):", "bar", true},
		{"async def baz():", "baz", true},
		{"  async def qux(x, y):", "qux", true},
		{"class Foo:", "", false},
		{"define()", "", false},
		{"", "", false},
	}
	for _, tc := range tests {
		name, ok := isPyDefLine(tc.line)
		if ok != tc.expectOk || name != tc.expectName {
			t.Errorf("isPyDefLine(%q) = (%q, %v), want (%q, %v)", tc.line, name, ok, tc.expectName, tc.expectOk)
		}
	}
}

func TestIsNonProductionPath(t *testing.T) {
	if !IsNonProductionPath("examples/demo.go") {
		t.Error("examples should be non-production")
	}
	if !IsNonProductionPath("third_party/lib.go") {
		t.Error("third_party should be non-production")
	}
	if IsNonProductionPath("internal/server.go") {
		t.Error("internal should be production")
	}
}

func TestMatchesTrivialVerbPattern(t *testing.T) {
	tests := []struct {
		body   string
		expect bool
	}{
		{"Return the value", true},
		{"Returns the value", true},
		{"Returning the value", true},
		{"Check for errors", true},
		{"Checking for errors", true},
		{"Delete the record", true},
		{"Set up the connection", true},
		{"Because of the bug", false},
		{"Note: important caveat", false},
		{"", false},
	}
	for _, tc := range tests {
		got := matchesTrivialVerbPattern(tc.body)
		if got != tc.expect {
			t.Errorf("matchesTrivialVerbPattern(%q) = %v, want %v", tc.body, got, tc.expect)
		}
	}
}

func TestMatchesTrivialThisPattern(t *testing.T) {
	tests := []struct {
		body   string
		expect bool
	}{
		{"This function does X", true},
		{"This method handles Y", true},
		{"This is important because", false},
		{"These functions", false},
		{"this CLASS does", true},
	}
	for _, tc := range tests {
		got := matchesTrivialThisPattern(tc.body)
		if got != tc.expect {
			t.Errorf("matchesTrivialThisPattern(%q) = %v, want %v", tc.body, got, tc.expect)
		}
	}
}
