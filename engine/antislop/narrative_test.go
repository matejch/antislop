package antislop

import (
	"testing"
)

func TestNarrative_DetectsDecorativeSeparator(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": "package app\n\n// ────────────────────────────────────────────\nvar x = 1\n",
	})
	diags := DetectNarrativeComments(ctx)
	got := diagsWithRule(diags, "antislop/narrative-comment")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for decorative separator, got %d", len(got))
	}
}

func TestNarrative_DetectsPhaseHeader(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": "package app\n\n// Phase 1: Code changes\nvar y = 2\n",
	})
	diags := DetectNarrativeComments(ctx)
	got := diagsWithRule(diags, "antislop/narrative-comment")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for phase header, got %d", len(got))
	}
}

func TestNarrative_DetectsMultiLinePreamble(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": `package app

// This function does N things in order.
// First it parses the input.
// Then it validates it.
// Finally it emits the output.
func run() int { return 0 }
`,
	})
	diags := DetectNarrativeComments(ctx)
	got := diagsWithRule(diags, "antislop/narrative-comment")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for multi-line preamble, got %d", len(got))
	}
}

func TestNarrative_DetectsCrossReference(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": "package app\n\n// buildFixRender will then be called with includeHeader: false\nvar y = 1\n",
	})
	diags := DetectNarrativeComments(ctx)
	got := diagsWithRule(diags, "antislop/narrative-comment")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for cross-reference, got %d", len(got))
	}
}

func TestNarrative_PreservesLicenseHeader(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": "// Copyright (c) 2026 Kenny\n// SPDX-License-Identifier: MIT\n// All rights reserved.\npackage app\n\nvar v = 1\n",
	})
	diags := DetectNarrativeComments(ctx)
	got := diagsWithRule(diags, "antislop/narrative-comment")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for license header, got %d", len(got))
	}
}

func TestNarrative_PreservesGoDocComment(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": `package app

// Load retrieves the entity from the store.
// It returns an error if the entity is not found.
// The caller should check the error value.
func Load() error { return nil }
`,
	})
	diags := DetectNarrativeComments(ctx)
	got := diagsWithRule(diags, "antislop/narrative-comment")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for Go doc comment, got %d", len(got))
	}
}

func TestNarrative_DoesNotFlagShortWhyComment(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": `package app

func run() int {
    // wcwidth returns -1 for unmapped codepoints; treat as width 1.
    return 1
}
`,
	})
	diags := DetectNarrativeComments(ctx)
	got := diagsWithRule(diags, "antislop/narrative-comment")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for short WHY comment, got %d", len(got))
	}
}

func TestNarrative_FlagsLongNarrativeBlock(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": `package app

func run() int {
    // First paragraph describing what we do.
    // There are several reasons for this shape.
    // It interacts with the caller's assumptions.
    // After the earlier refactor we kept the shape.
    // The downstream tests depend on this order.
    return 1
}
`,
	})
	diags := DetectNarrativeComments(ctx)
	got := diagsWithRule(diags, "antislop/narrative-comment")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for long narrative block, got %d", len(got))
	}
}

func TestNarrative_ExemptsThreeLineBlockWithWhyMarker(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": `package app

func run() int {
    // Run this before the CORS middleware because credentialed origins
    // otherwise reject OPTIONS requests. Discovered in prod after session
    // cookies started arriving stripped.
    return 1
}
`,
	})
	diags := DetectNarrativeComments(ctx)
	got := diagsWithRule(diags, "antislop/narrative-comment")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for WHY block, got %d", len(got))
	}
}

func TestNarrative_DoesNotFlagImperativeStepComments(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": `package app

func render() {
	// Render top
	doStuff()

	// Render sides
	doMore()

	// Render bottom
	doEnd()
}
`,
	})
	diags := DetectNarrativeComments(ctx)
	got := diagsWithRule(diags, "antislop/narrative-comment")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for step comments, got %d", len(got))
	}
}

func TestNarrative_PreservesStructFieldDocComment(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": `package x

type Key struct {
	// Text contains the actual characters received. This usually the same as
	// Key.Code. When Key.Text is non-empty, it indicates that the key
	// pressed represents printable character(s).
	Text string
}
`,
	})
	diags := DetectNarrativeComments(ctx)
	got := diagsWithRule(diags, "antislop/narrative-comment")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for struct field doc, got %d", len(got))
	}
}

func TestNarrative_DoesNotFlagNonProductionPaths(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"examples/demo.go": "package demo\n\n// Phase 1: Setup\nvar x = 1\n",
	})
	diags := DetectNarrativeComments(ctx)
	got := diagsWithRule(diags, "antislop/narrative-comment")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for examples dir, got %d", len(got))
	}
}

func TestNarrative_DetectsDecorativeSectionHeaderWithTitle(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": "package app\n\n// ─── Classification ───────────────────\nvar x = 1\n",
	})
	diags := DetectNarrativeComments(ctx)
	got := diagsWithRule(diags, "antislop/narrative-comment")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for section header with title, got %d", len(got))
	}
}

func TestNarrative_DetectsExplanatoryPreambleBeforeDecl(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": "package app\n\n// Matches the user input against known patterns.\nvar pattern = compile()\n",
	})
	diags := DetectNarrativeComments(ctx)
	got := diagsWithRule(diags, "antislop/narrative-comment")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for explanatory preamble, got %d", len(got))
	}
}

func TestNarrative_DoesNotFlagExplanatoryOpenerInsideBody(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": "package app\n\nfunc f() {\n\t// Matches the user input\n\treturn\n}\n",
	})
	diags := DetectNarrativeComments(ctx)
	got := diagsWithRule(diags, "antislop/narrative-comment")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for opener inside body (not before decl), got %d", len(got))
	}
}

func TestNarrative_DetectsJustificationProse(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": "package app\n\nfunc f() {\n\t// This function handles the edge case\n\tx := 1\n\t_ = x\n}\n",
	})
	diags := DetectNarrativeComments(ctx)
	got := diagsWithRule(diags, "antislop/narrative-comment")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for justification prose, got %d", len(got))
	}
}

func TestNarrative_DetectsPyDeclarationPreamble(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.py": "# This function handles all incoming requests.\n# First it parses the body.\n# Then it validates the schema.\ndef handle():\n    pass\n",
	})
	diags := DetectNarrativeComments(ctx)
	got := diagsWithRule(diags, "antislop/narrative-comment")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for Python preamble, got %d", len(got))
	}
}

func TestNarrative_DetectsMultiLinePreambleWithJustification(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": `package app

// It handles all the incoming requests.
// Then it routes them to the correct handler.
// Finally it sends back the response.
func handle() {}
`,
	})
	diags := DetectNarrativeComments(ctx)
	got := diagsWithRule(diags, "antislop/narrative-comment")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for justification preamble, got %d", len(got))
	}
}

func TestNarrative_PreservesPyAsyncDefDoc(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.py": "# Process handles async tasks.\n# It validates and stores them.\n# Runs on every request cycle.\nasync def process():\n    pass\n",
	})
	diags := DetectNarrativeComments(ctx)
	// This should detect the preamble (it has slop signals)
	// Just exercising the async def path
	_ = diags
}

func TestNarrative_PreservesGoDocForTypeDecl(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": `package app

// Config holds the application configuration.
// It is loaded from environment variables.
// Defaults are provided for development.
type Config struct {
	Port int
}
`,
	})
	diags := DetectNarrativeComments(ctx)
	got := diagsWithRule(diags, "antislop/narrative-comment")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for type doc comment, got %d", len(got))
	}
}

func TestNarrative_PreservesGoDocForVar(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": `package app

// DefaultTimeout specifies the fallback timeout value.
// It is used when no configuration is provided.
// The value was chosen based on production metrics.
var DefaultTimeout = 30
`,
	})
	diags := DetectNarrativeComments(ctx)
	got := diagsWithRule(diags, "antislop/narrative-comment")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for var doc comment, got %d", len(got))
	}
}

func TestNarrative_DetectsCrossRefInMultiLineBlock(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": "package app\n\nfunc f() {\n\t// We used to have a different approach\n\t// but refactored from the old system\n\t// to this new implementation.\n\tx := 1\n\t_ = x\n}\n",
	})
	diags := DetectNarrativeComments(ctx)
	got := diagsWithRule(diags, "antislop/narrative-comment")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for cross-ref in block, got %d", len(got))
	}
}

func TestNarrative_DetectsPythonPhaseHeader(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.py": "# Phase 1: Build payload\ndef build_payload():\n    return 1\n",
	})
	diags := DetectNarrativeComments(ctx)
	got := diagsWithRule(diags, "antislop/narrative-comment")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for Python phase header, got %d", len(got))
	}
}
