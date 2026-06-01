package antislop

import (
	"testing"
)

func TestTrivialComments_FlagsThisFunction(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": `package app

// This function calculates the total
func calculateTotal() int { return 0 }
`,
	})
	diags := DetectTrivialComments(ctx)
	got := diagsWithRule(diags, "antislop/trivial-comment")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic, got %d", len(got))
	}
}

func TestTrivialComments_FlagsReturnComment(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": `package app

func getValue() int {
    // Return the computed value
    return 42
}
`,
	})
	diags := DetectTrivialComments(ctx)
	got := diagsWithRule(diags, "antislop/trivial-comment")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic, got %d", len(got))
	}
}

func TestTrivialComments_FlagsLoopComment(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": `package app

// Loop through all users
func process() {}
`,
	})
	diags := DetectTrivialComments(ctx)
	got := diagsWithRule(diags, "antislop/trivial-comment")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic, got %d", len(got))
	}
}

func TestTrivialComments_FlagsBareImperatives(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": `package app

func f() {
	// Cleanup
	cleanup()
	// Setup
	init()
	// Parse
	parse(x)
}
`,
	})
	diags := DetectTrivialComments(ctx)
	got := diagsWithRule(diags, "antislop/trivial-comment")
	if len(got) < 3 {
		t.Fatalf("expected >=3 diagnostics for bare imperatives, got %d", len(got))
	}
}

func TestTrivialComments_DoesNotFlagWhyMarkers(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": `package app

// Run this before the middleware because credentialed origins reject OPTIONS
func registerHook() {}
`,
	})
	diags := DetectTrivialComments(ctx)
	got := diagsWithRule(diags, "antislop/trivial-comment")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for WHY comment, got %d", len(got))
	}
}

func TestTrivialComments_DoesNotFlagNonProductionPaths(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"examples/demo.go": `package demo

// This function creates a demo
func createDemo() {}
`,
	})
	diags := DetectTrivialComments(ctx)
	got := diagsWithRule(diags, "antislop/trivial-comment")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for examples dir, got %d", len(got))
	}
}

func TestTrivialComments_DoesNotFlagSectionDividerFollowedByBlank(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": `package app

// Setup

func init() {}
`,
	})
	diags := DetectTrivialComments(ctx)
	got := diagsWithRule(diags, "antislop/trivial-comment")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for section divider, got %d", len(got))
	}
}

func TestTrivialComments_FlagsSetUpComment(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": "package app\n\nfunc f() {\n\t// Setting up the connection\n\tinit()\n}\n",
	})
	diags := DetectTrivialComments(ctx)
	got := diagsWithRule(diags, "antislop/trivial-comment")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for 'Setting up', got %d", len(got))
	}
}

func TestTrivialComments_DoesNotFlagUnicodeDivider(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": "package app\n\n// ───\nvar x = 1\n",
	})
	diags := DetectTrivialComments(ctx)
	got := diagsWithRule(diags, "antislop/trivial-comment")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for unicode divider, got %d", len(got))
	}
}

func TestTrivialComments_DoesNotFlagDashDivider(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": "package app\n\n// ---\nvar x = 1\n",
	})
	diags := DetectTrivialComments(ctx)
	got := diagsWithRule(diags, "antislop/trivial-comment")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for dash divider, got %d", len(got))
	}
}

func TestTrivialComments_DoesNotFlagCodeChars(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": "package app\n\n// Return (x, y)\nfunc f() {}\n",
	})
	diags := DetectTrivialComments(ctx)
	got := diagsWithRule(diags, "antislop/trivial-comment")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for comment with parens, got %d", len(got))
	}
}

func TestTrivialComments_FlagsPythonTrivial(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.py": `# This function calculates the total
def calculate_total():
    return 0
`,
	})
	diags := DetectTrivialComments(ctx)
	got := diagsWithRule(diags, "antislop/trivial-comment")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for Python trivial comment, got %d", len(got))
	}
}
