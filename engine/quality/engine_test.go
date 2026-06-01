package quality

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matej/antislop/engine"
)

func setupQualCtx(t *testing.T, files map[string]string, cfg engine.EngineConfig) engine.EngineContext {
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
		Config:    cfg,
	}
}

func defaultQualCfg() engine.EngineConfig {
	return engine.EngineConfig{
		Quality: engine.QualityConfig{
			MaxFunctionLoc: 80,
			MaxFileLoc:     400,
			MaxNesting:     5,
			MaxParams:      6,
		},
	}
}

func qualDiagsWithRule(diags []engine.Diagnostic, rule string) []engine.Diagnostic {
	var out []engine.Diagnostic
	for _, d := range diags {
		if d.Rule == rule {
			out = append(out, d)
		}
	}
	return out
}

func TestQuality_FileTooLong(t *testing.T) {
	cfg := defaultQualCfg()
	cfg.Quality.MaxFileLoc = 10

	lines := make([]string, 15)
	lines[0] = "package app"
	for i := 1; i < 15; i++ {
		lines[i] = fmt.Sprintf("var x%d = %d", i, i)
	}

	ctx := setupQualCtx(t, map[string]string{
		"big.go": strings.Join(lines, "\n") + "\n",
	}, cfg)

	cq := CodeQuality{}
	result := cq.Run(ctx)
	got := qualDiagsWithRule(result.Diagnostics, "code-quality/file-too-long")
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic for file-too-long, got %d", len(got))
	}
}

func TestQuality_FileAtExactLimitIsOK(t *testing.T) {
	cfg := defaultQualCfg()
	cfg.Quality.MaxFileLoc = 10

	lines := make([]string, 10)
	lines[0] = "package app"
	for i := 1; i < 10; i++ {
		lines[i] = fmt.Sprintf("var x%d = %d", i, i)
	}

	ctx := setupQualCtx(t, map[string]string{
		"ok.go": strings.Join(lines, "\n") + "\n",
	}, cfg)

	cq := CodeQuality{}
	result := cq.Run(ctx)
	got := qualDiagsWithRule(result.Diagnostics, "code-quality/file-too-long")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics at exact limit, got %d", len(got))
	}
}

func TestQuality_GoFunctionTooLong(t *testing.T) {
	cfg := defaultQualCfg()
	cfg.Quality.MaxFunctionLoc = 5

	var body strings.Builder
	body.WriteString("package app\n\nfunc processData(a string) string {\n")
	for i := 0; i < 8; i++ {
		body.WriteString(fmt.Sprintf("    x%d := %d\n", i, i))
	}
	body.WriteString("    return a\n}\n")

	ctx := setupQualCtx(t, map[string]string{"app.go": body.String()}, cfg)

	cq := CodeQuality{}
	result := cq.Run(ctx)
	got := qualDiagsWithRule(result.Diagnostics, "code-quality/function-too-long")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for function-too-long, got %d", len(got))
	}
}

func TestQuality_PythonFunctionTooLong(t *testing.T) {
	cfg := defaultQualCfg()
	cfg.Quality.MaxFunctionLoc = 5

	var body strings.Builder
	body.WriteString("def process(a):\n")
	for i := 0; i < 8; i++ {
		body.WriteString(fmt.Sprintf("    x%d = %d\n", i, i))
	}
	body.WriteString("    return a\n")

	ctx := setupQualCtx(t, map[string]string{"app.py": body.String()}, cfg)

	cq := CodeQuality{}
	result := cq.Run(ctx)
	got := qualDiagsWithRule(result.Diagnostics, "code-quality/function-too-long")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for python function-too-long, got %d", len(got))
	}
}

func TestQuality_GoDeepNesting(t *testing.T) {
	cfg := defaultQualCfg()
	cfg.Quality.MaxNesting = 2

	content := `package app

func deep() {
    if true {
        if true {
            if true {
                if true {
                    x := 1
                    _ = x
                }
            }
        }
    }
}
`
	ctx := setupQualCtx(t, map[string]string{"app.go": content}, cfg)

	cq := CodeQuality{}
	result := cq.Run(ctx)
	got := qualDiagsWithRule(result.Diagnostics, "code-quality/deep-nesting")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for deep nesting, got %d", len(got))
	}
}

func TestQuality_GoMethodFunctionTooLong(t *testing.T) {
	cfg := defaultQualCfg()
	cfg.Quality.MaxFunctionLoc = 5

	var body strings.Builder
	body.WriteString("package app\n\nfunc (s *Store) Process(a string) string {\n")
	for i := 0; i < 8; i++ {
		body.WriteString(fmt.Sprintf("    x%d := %d\n", i, i))
	}
	body.WriteString("    return a\n}\n")

	ctx := setupQualCtx(t, map[string]string{"app.go": body.String()}, cfg)
	cq := CodeQuality{}
	result := cq.Run(ctx)
	got := qualDiagsWithRule(result.Diagnostics, "code-quality/function-too-long")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for method too long, got %d", len(got))
	}
}

func TestQuality_PyAsyncFunctionTooLong(t *testing.T) {
	cfg := defaultQualCfg()
	cfg.Quality.MaxFunctionLoc = 5

	var body strings.Builder
	body.WriteString("async def process(a):\n")
	for i := 0; i < 8; i++ {
		body.WriteString(fmt.Sprintf("    x%d = %d\n", i, i))
	}
	body.WriteString("    return a\n")

	ctx := setupQualCtx(t, map[string]string{"app.py": body.String()}, cfg)
	cq := CodeQuality{}
	result := cq.Run(ctx)
	got := qualDiagsWithRule(result.Diagnostics, "code-quality/function-too-long")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for async function too long, got %d", len(got))
	}
}

func TestQuality_AllDiagsHaveCorrectEngine(t *testing.T) {
	cfg := defaultQualCfg()
	cfg.Quality.MaxFileLoc = 5

	lines := make([]string, 10)
	lines[0] = "package app"
	for i := 1; i < 10; i++ {
		lines[i] = fmt.Sprintf("var x%d = %d", i, i)
	}
	ctx := setupQualCtx(t, map[string]string{"big.go": strings.Join(lines, "\n") + "\n"}, cfg)
	cq := CodeQuality{}
	result := cq.Run(ctx)
	for _, d := range result.Diagnostics {
		if d.Engine != "code-quality" {
			t.Errorf("expected engine code-quality, got %s", d.Engine)
		}
	}
}

func TestQuality_ShortFunctionIsOK(t *testing.T) {
	ctx := setupQualCtx(t, map[string]string{
		"app.go": "package app\n\nfunc shortFn(a int) int {\n    return a + 1\n}\n",
	}, defaultQualCfg())

	cq := CodeQuality{}
	result := cq.Run(ctx)
	got := qualDiagsWithRule(result.Diagnostics, "code-quality/function-too-long")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for short function, got %d", len(got))
	}
}
