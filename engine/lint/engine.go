package lint

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/matej/antislop/engine"
)

type Lint struct{}

func (l Lint) Name() engine.EngineName { return engine.EngineLint }

func (l Lint) Run(ctx engine.EngineContext) engine.EngineResult {
	var diags []engine.Diagnostic

	for _, lang := range ctx.Languages {
		switch lang {
		case "go":
			if ctx.InstalledTools["golangci-lint"] {
				diags = append(diags, runGolangciLint(ctx)...)
			}
		case "python":
			if ctx.InstalledTools["ruff"] {
				diags = append(diags, runRuffLint(ctx)...)
			}
		}
	}

	return engine.EngineResult{
		Engine:      engine.EngineLint,
		Diagnostics: diags,
	}
}

type golangciIssue struct {
	FromLinter string `json:"FromLinter"`
	Text       string `json:"Text"`
	Pos        struct {
		Filename string `json:"Filename"`
		Line     int    `json:"Line"`
		Column   int    `json:"Column"`
	} `json:"Pos"`
	Severity string `json:"Severity"`
}

type golangciOutput struct {
	Issues []golangciIssue `json:"Issues"`
}

func runGolangciLint(ctx engine.EngineContext) []engine.Diagnostic {
	cmd := exec.Command("golangci-lint", "run", "--out-format=json", "--timeout=60s", "./...")
	cmd.Dir = ctx.RootDir
	// golangci-lint exits non-zero when it finds issues but still produces valid JSON
	out, ignored := cmd.Output()
	_ = ignored

	var parsed golangciOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil
	}

	var diags []engine.Diagnostic
	for _, issue := range parsed.Issues {
		sev := engine.SeverityWarning
		if issue.Severity == "error" {
			sev = engine.SeverityError
		}
		diags = append(diags, engine.Diagnostic{
			FilePath: toSlashRel(ctx.RootDir, issue.Pos.Filename),
			Engine:   engine.EngineLint,
			Rule:     fmt.Sprintf("lint/%s", issue.FromLinter),
			Severity: sev,
			Message:  issue.Text,
			Help:     fmt.Sprintf("Fix issue reported by %s", issue.FromLinter),
			Line:     issue.Pos.Line,
			Column:   issue.Pos.Column,
			Category: "Lint",
		})
	}
	return diags
}

type ruffDiagnostic struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Filename string `json:"filename"`
	Location struct {
		Row    int `json:"row"`
		Column int `json:"column"`
	} `json:"location"`
	Fix *struct{} `json:"fix"`
}

func runRuffLint(ctx engine.EngineContext) []engine.Diagnostic {
	cmd := exec.Command("ruff", "check", "--output-format=json", ctx.RootDir)
	out, ignored := cmd.Output()
	_ = ignored

	var parsed []ruffDiagnostic
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil
	}

	var diags []engine.Diagnostic
	for _, rd := range parsed {
		diags = append(diags, engine.Diagnostic{
			FilePath: toSlashRel(ctx.RootDir, rd.Filename),
			Engine:   engine.EngineLint,
			Rule:     fmt.Sprintf("lint/ruff-%s", rd.Code),
			Severity: engine.SeverityWarning,
			Message:  rd.Message,
			Help:     fmt.Sprintf("Fix %s: %s", rd.Code, rd.Message),
			Line:     rd.Location.Row,
			Column:   rd.Location.Column,
			Category: "Lint",
			Fixable:  rd.Fix != nil,
		})
	}

	fileSet := map[string]bool{}
	for _, f := range ctx.Files {
		if filepath.Ext(f) == ".py" {
			fileSet[toSlashRel(ctx.RootDir, f)] = true
		}
	}

	var filtered []engine.Diagnostic
	for _, d := range diags {
		if fileSet[d.FilePath] {
			filtered = append(filtered, d)
		}
	}

	if len(filtered) > 0 {
		return filtered
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
