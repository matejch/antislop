package format

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/matej/antislop/engine"
)

type Format struct{}

func (f Format) Name() engine.EngineName { return engine.EngineFormat }

func (f Format) Run(ctx engine.EngineContext) engine.EngineResult {
	var diags []engine.Diagnostic

	for _, lang := range ctx.Languages {
		switch lang {
		case "go":
			diags = append(diags, checkGoFmt(ctx)...)
		case "python":
			if ctx.InstalledTools["ruff"] {
				diags = append(diags, checkRuffFormat(ctx)...)
			}
		}
	}

	return engine.EngineResult{
		Engine:      engine.EngineFormat,
		Diagnostics: diags,
	}
}

func checkGoFmt(ctx engine.EngineContext) []engine.Diagnostic {
	var diags []engine.Diagnostic

	for _, filePath := range ctx.Files {
		if filepath.Ext(filePath) != ".go" {
			continue
		}
		if strings.HasSuffix(filePath, "_test.go") {
			continue
		}

		cmd := exec.Command("gofmt", "-l", filePath)
		out, err := cmd.Output()
		if err != nil {
			continue
		}
		if len(bytes.TrimSpace(out)) > 0 {
			diags = append(diags, engine.Diagnostic{
				FilePath: toSlashRel(ctx.RootDir, filePath),
				Engine:   engine.EngineFormat,
				Rule:     "format/gofmt",
				Severity: engine.SeverityWarning,
				Message:  "File is not formatted with gofmt",
				Help:     "Run `gofmt -w` on this file",
				Line:     1,
				Category: "Formatting",
				Fixable:  true,
			})
		}
	}
	return diags
}

func checkRuffFormat(ctx engine.EngineContext) []engine.Diagnostic {
	var diags []engine.Diagnostic

	var pyFiles []string
	for _, f := range ctx.Files {
		if filepath.Ext(f) == ".py" {
			pyFiles = append(pyFiles, f)
		}
	}
	if len(pyFiles) == 0 {
		return nil
	}

	args := append([]string{"format", "--check", "--diff"}, pyFiles...)
	cmd := exec.Command("ruff", args...)
	cmd.Dir = ctx.RootDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err == nil && len(out) == 0 {
		return nil
	}

	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") {
			file := strings.TrimPrefix(line, "+++ ")
			file = strings.TrimPrefix(file, "--- ")
			file = strings.TrimSpace(file)
			if file != "" {
				diags = append(diags, engine.Diagnostic{
					FilePath: toSlashRel(ctx.RootDir, file),
					Engine:   engine.EngineFormat,
					Rule:     "format/ruff",
					Severity: engine.SeverityWarning,
					Message:  "File is not formatted with ruff format",
					Help:     "Run `ruff format` on this file",
					Line:     1,
					Category: "Formatting",
					Fixable:  true,
				})
			}
		}
	}
	return diags
}

func toSlashRel(root, p string) string {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return p
	}
	return filepath.ToSlash(rel)
}
