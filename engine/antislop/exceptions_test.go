package antislop

import (
	"testing"
)

func TestExceptions_FlagsPythonBareExceptPass(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.py": "try:\n    do_thing()\nexcept:\n    pass\n",
	})
	diags := DetectSwallowedExceptions(ctx)
	got := diagsWithRule(diags, "antislop/swallowed-exception")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for bare except pass, got %d", len(got))
	}
}

func TestExceptions_FlagsPythonExceptWithPrint(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.py": "try:\n    do_thing()\nexcept Exception as e:\n    print(e)\n",
	})
	diags := DetectSwallowedExceptions(ctx)
	got := diagsWithRule(diags, "antislop/swallowed-exception")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for except-print, got %d", len(got))
	}
}

func TestExceptions_FlagsGoIgnoredError(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": `package app

func Run() {
    val, _ := doSomething()
    _ = val
}
`,
	})
	diags := DetectSwallowedExceptions(ctx)
	got := diagsWithRule(diags, "antislop/swallowed-exception")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for ignored error, got %d", len(got))
	}
}

func TestExceptions_DoesNotFlagIntentionalIgnoreNames(t *testing.T) {
	// The pattern checks the first variable name, not the _, so
	// "ignored, _ := ..." should not be flagged because "ignored" is
	// an intentional ignore name.
	ctx := setupCtx(t, map[string]string{
		"app.go": `package app

func Run() {
    ignored, _ := doSomething()
    _ = ignored
}
`,
	})
	diags := DetectSwallowedExceptions(ctx)
	got := diagsWithRule(diags, "antislop/swallowed-exception")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for intentional ignore, got %d", len(got))
	}
}

func TestExceptions_FlagsGoErrorIgnoreWithColonEquals(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": "package app\n\nfunc Run() {\n    data, _ := os.Open(\"file\")\n    _ = data\n}\n",
	})
	diags := DetectSwallowedExceptions(ctx)
	got := diagsWithRule(diags, "antislop/swallowed-exception")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for := error ignore, got %d", len(got))
	}
}

func TestExceptions_FlagsGoErrorIgnoreWithDottedCall(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": "package app\n\nfunc Run() {\n    val, _ = pkg.DoThing()\n    _ = val\n}\n",
	})
	diags := DetectSwallowedExceptions(ctx)
	got := diagsWithRule(diags, "antislop/swallowed-exception")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for dotted call error ignore, got %d", len(got))
	}
}

func TestExceptions_DoesNotFlagEqualityComparison(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": "package app\n\nfunc Run() bool {\n    x, _ == y, z\n    return true\n}\n",
	})
	diags := DetectSwallowedExceptions(ctx)
	got := diagsWithRule(diags, "antislop/swallowed-exception")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for == comparison, got %d", len(got))
	}
}

func TestExceptions_FlagsNormalVariableIgnoringError(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": `package app

func Run() {
    result, _ := fetchData()
    process(result)
}
`,
	})
	diags := DetectSwallowedExceptions(ctx)
	got := diagsWithRule(diags, "antislop/swallowed-exception")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for non-intentional error ignore, got %d", len(got))
	}
}
