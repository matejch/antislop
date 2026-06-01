package antislop

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matej/antislop/engine"
)

func setupCtx(t *testing.T, files map[string]string) engine.EngineContext {
	t.Helper()
	dir := t.TempDir()
	var paths []string
	for name, content := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	return engine.EngineContext{
		RootDir:   dir,
		Languages: []string{"go", "python"},
		Files:     paths,
		Config: engine.EngineConfig{
			Quality: engine.QualityConfig{
				MaxFunctionLoc: 80,
				MaxFileLoc:     400,
				MaxNesting:     5,
				MaxParams:      6,
			},
		},
	}
}

func diagsWithRule(diags []engine.Diagnostic, rule string) []engine.Diagnostic {
	var out []engine.Diagnostic
	for _, d := range diags {
		if d.Rule == rule {
			out = append(out, d)
		}
	}
	return out
}

func TestGoPatterns_FlagsPanicInNonMainPackage(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"store.go": `package store

func Load(name string) string {
    if name == "" {
        panic("empty name")
    }
    return name
}
`,
	})
	diags := DetectGoPatterns(ctx)
	got := diagsWithRule(diags, "antislop/go-library-panic")
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(got))
	}
	if got[0].Line != 5 {
		t.Errorf("expected line 5, got %d", got[0].Line)
	}
}

func TestGoPatterns_DoesNotFlagPackageMain(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"main.go": `package main

func main() {
    if err := run(); err != nil {
        panic(err)
    }
}
`,
	})
	diags := DetectGoPatterns(ctx)
	got := diagsWithRule(diags, "antislop/go-library-panic")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics, got %d", len(got))
	}
}

func TestGoPatterns_DoesNotFlagTestFiles(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"store_test.go": `package store_test

func TestLoad(t *testing.T) {
    panic("test panic")
}
`,
	})
	diags := DetectGoPatterns(ctx)
	got := diagsWithRule(diags, "antislop/go-library-panic")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics in test file, got %d", len(got))
	}
}

func TestGoPatterns_DoesNotFlagPanicInCommentOrString(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"lib.go": `package lib

// This function used to panic. We removed it.
const Message = "do not panic"

func Safe() string {
    return Message
}
`,
	})
	diags := DetectGoPatterns(ctx)
	got := diagsWithRule(diags, "antislop/go-library-panic")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for comment/string panic, got %d", len(got))
	}
}

func TestGoPatterns_DoesNotFlagPanicWithExplanatoryComment(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"cmd.go": `package cmd

// checkGroups validates a sub-command's group. If the group isn't defined
// we panic because it indicates a coding error that should be corrected.
func checkGroups() {
    panic("group not defined")
}
`,
	})
	diags := DetectGoPatterns(ctx)
	got := diagsWithRule(diags, "antislop/go-library-panic")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for documented panic, got %d", len(got))
	}
}

func TestGoPatterns_StillFlagsUndocumentedPanic(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"cmd.go": `package cmd

func DoThing(name string) {
    if name == "" {
        panic("empty")
    }
}
`,
	})
	diags := DetectGoPatterns(ctx)
	got := diagsWithRule(diags, "antislop/go-library-panic")
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic for undocumented panic, got %d", len(got))
	}
}

func TestGoPatterns_DoesNotFlagNilGuardPanic(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"cache.go": `package cache

func New(opts *Opts) {
    if opts == nil {
        panic("nil opts")
    }
    if opts.Log == nil {
        panic("nil Log")
    }
}
`,
	})
	diags := DetectGoPatterns(ctx)
	got := diagsWithRule(diags, "antislop/go-library-panic")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for nil-guard panics, got %d", len(got))
	}
}

func TestGoPatterns_StillFlagsLongStringPanicAfterNilGuard(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"lib.go": `package lib

func F(x *T) {
    if x == nil {
        panic("a really really long story about how this happened in production yesterday")
    }
}
`,
	})
	diags := DetectGoPatterns(ctx)
	got := diagsWithRule(diags, "antislop/go-library-panic")
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic for long-string panic, got %d", len(got))
	}
}

func TestGoPatterns_DoesNotFlagPanicWithNonStringArg(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"lib.go": `package lib

func F(x *T) {
    if x == nil {
        panic(fmt.Errorf("complex error: %w", err))
    }
}
`,
	})
	diags := DetectGoPatterns(ctx)
	// panic with non-string arg — nil guard check should see it's not a short string panic
	// and still flag it since it's not a nil guard panic
	got := diagsWithRule(diags, "antislop/go-library-panic")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for non-string panic arg, got %d", len(got))
	}
}

func TestGoPatterns_DoesNotFlagAutoGenerated(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"gen.go": "// Code generated by tool. DO NOT EDIT.\npackage lib\n\nfunc A() {\n    panic(\"a\")\n}\n",
	})
	diags := DetectGoPatterns(ctx)
	got := diagsWithRule(diags, "antislop/go-library-panic")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for auto-generated file, got %d", len(got))
	}
}

func TestGoPatterns_FlagsMultiplePanicsWithCorrectLines(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"lib.go": `package lib

func A() {
    panic("a")
}

func B() {
    panic("b")
}
`,
	})
	diags := DetectGoPatterns(ctx)
	got := diagsWithRule(diags, "antislop/go-library-panic")
	if len(got) != 2 {
		t.Fatalf("expected 2 diagnostics, got %d", len(got))
	}
	if got[0].Line != 4 || got[1].Line != 8 {
		t.Errorf("expected lines [4, 8], got [%d, %d]", got[0].Line, got[1].Line)
	}
}
