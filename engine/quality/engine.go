package quality

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/matej/antislop/engine"
)

type CodeQuality struct{}

func (c CodeQuality) Name() engine.EngineName { return engine.EngineCodeQuality }

func (c CodeQuality) Run(ctx engine.EngineContext) engine.EngineResult {
	var diags []engine.Diagnostic
	diags = append(diags, detectComplexity(ctx)...)
	return engine.EngineResult{
		Engine:      engine.EngineCodeQuality,
		Diagnostics: diags,
	}
}

// isPyFuncStart checks for "def name(" or "async def name(" and returns the name.
func isPyFuncStart(line string) (string, bool) {
	t := strings.TrimLeft(line, " \t")
	if strings.HasPrefix(t, "async ") || strings.HasPrefix(t, "async\t") {
		t = strings.TrimLeft(t[5:], " \t")
	}
	if !strings.HasPrefix(t, "def ") && !strings.HasPrefix(t, "def\t") {
		return "", false
	}
	rest := strings.TrimLeft(t[3:], " \t")
	end := 0
	for end < len(rest) && isWordCharQ(rest[end]) {
		end++
	}
	if end == 0 {
		return "", false
	}
	return rest[:end], true
}

func isGoFuncStart(trimmed string) bool {
	return strings.HasPrefix(trimmed, "func ") || strings.HasPrefix(trimmed, "func\t")
}

func isWordCharQ(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

// extractGoFuncName extracts function name from "func (recv) Name(" or "func Name("
func extractGoFuncName(line string) string {
	rest := line
	if !strings.HasPrefix(rest, "func") {
		return ""
	}
	rest = rest[4:]
	rest = strings.TrimLeft(rest, " \t")
	// Skip optional receiver: (...)
	if len(rest) > 0 && rest[0] == '(' {
		depth := 0
		i := 0
		for i < len(rest) {
			if rest[i] == '(' {
				depth++
			} else if rest[i] == ')' {
				depth--
				if depth == 0 {
					rest = rest[i+1:]
					break
				}
			}
			i++
		}
		rest = strings.TrimLeft(rest, " \t")
	}
	end := 0
	for end < len(rest) && isWordCharQ(rest[end]) {
		end++
	}
	if end == 0 {
		return ""
	}
	return rest[:end]
}

func detectComplexity(ctx engine.EngineContext) []engine.Diagnostic {
	var diags []engine.Diagnostic

	for _, filePath := range ctx.Files {
		ext := filepath.Ext(filePath)
		if ext != ".go" && ext != ".py" {
			continue
		}
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		content := string(data)
		rel := toSlashRelQ(ctx.RootDir, filePath)
		lines := strings.Split(strings.TrimRight(content, "\n"), "\n")

		if len(lines) > ctx.Config.Quality.MaxFileLoc {
			diags = append(diags, engine.Diagnostic{
				FilePath: rel,
				Engine:   engine.EngineCodeQuality,
				Rule:     "code-quality/file-too-long",
				Severity: engine.SeverityWarning,
				Message:  fmt.Sprintf("File has %d lines (limit: %d)", len(lines), ctx.Config.Quality.MaxFileLoc),
				Help:     "Consider splitting this file into smaller, focused modules",
				Line:     1,
				Category: "Complexity",
			})
		}

		switch ext {
		case ".go":
			diags = append(diags, checkGoFunctions(lines, rel, ctx.Config)...)
		case ".py":
			diags = append(diags, checkPyFunctions(lines, rel, ctx.Config)...)
		}
	}
	return diags
}

func checkGoFunctions(lines []string, relPath string, cfg engine.EngineConfig) []engine.Diagnostic {
	var diags []engine.Diagnostic
	braceDepth := 0
	funcStart := -1
	funcName := ""

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if isGoFuncStart(trimmed) && funcStart == -1 {
			funcStart = i
			funcName = extractGoFuncName(trimmed)
		}

		for _, ch := range trimmed {
			if ch == '{' {
				braceDepth++
			} else if ch == '}' {
				braceDepth--
			}
		}

		if funcStart >= 0 && braceDepth == 0 {
			loc := i - funcStart + 1
			if loc > cfg.Quality.MaxFunctionLoc {
				diags = append(diags, engine.Diagnostic{
					FilePath: relPath,
					Engine:   engine.EngineCodeQuality,
					Rule:     "code-quality/function-too-long",
					Severity: engine.SeverityWarning,
					Message:  fmt.Sprintf("Function '%s' has %d lines (limit: %d)", funcName, loc, cfg.Quality.MaxFunctionLoc),
					Help:     "Break this function into smaller, focused functions",
					Line:     funcStart + 1,
					Category: "Complexity",
				})
			}

			maxNest := checkNesting(lines[funcStart : i+1])
			if maxNest > cfg.Quality.MaxNesting {
				diags = append(diags, engine.Diagnostic{
					FilePath: relPath,
					Engine:   engine.EngineCodeQuality,
					Rule:     "code-quality/deep-nesting",
					Severity: engine.SeverityWarning,
					Message:  fmt.Sprintf("Function '%s' has nesting depth %d (limit: %d)", funcName, maxNest, cfg.Quality.MaxNesting),
					Help:     "Reduce nesting with early returns, extract helper functions, or use guard clauses",
					Line:     funcStart + 1,
					Category: "Complexity",
				})
			}

			funcStart = -1
			funcName = ""
		}
	}
	return diags
}

func checkPyFunctions(lines []string, relPath string, cfg engine.EngineConfig) []engine.Diagnostic {
	var diags []engine.Diagnostic

	for i, line := range lines {
		funcName, ok := isPyFuncStart(line)
		if !ok {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))

		bodyLines := 0
		for j := i + 1; j < len(lines); j++ {
			trimmed := strings.TrimSpace(lines[j])
			if trimmed == "" {
				bodyLines++
				continue
			}
			lineIndent := len(lines[j]) - len(strings.TrimLeft(lines[j], " \t"))
			if lineIndent <= indent {
				break
			}
			bodyLines++
		}

		if bodyLines > cfg.Quality.MaxFunctionLoc {
			diags = append(diags, engine.Diagnostic{
				FilePath: relPath,
				Engine:   engine.EngineCodeQuality,
				Rule:     "code-quality/function-too-long",
				Severity: engine.SeverityWarning,
				Message:  fmt.Sprintf("Function '%s' has %d lines (limit: %d)", funcName, bodyLines, cfg.Quality.MaxFunctionLoc),
				Help:     "Break this function into smaller, focused functions",
				Line:     i + 1,
				Category: "Complexity",
			})
		}
	}
	return diags
}

func checkNesting(lines []string) int {
	maxDepth := 0
	depth := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		for _, ch := range trimmed {
			if ch == '{' {
				depth++
				if depth > maxDepth {
					maxDepth = depth
				}
			} else if ch == '}' {
				depth--
			}
		}
	}
	if maxDepth > 0 {
		maxDepth--
	}
	return maxDepth
}

func toSlashRelQ(root, p string) string {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return p
	}
	return filepath.ToSlash(rel)
}
