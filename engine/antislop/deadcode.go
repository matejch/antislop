package antislop

import (
	"path/filepath"
	"strings"

	"github.com/matej/antislop/engine"
)

// containsFmtPrint checks for fmt.Print, fmt.Println, fmt.Printf with word boundary
func containsFmtPrint(line string) bool {
	idx := strings.Index(line, "fmt.Print")
	if idx < 0 {
		return false
	}
	// Check word boundary before "fmt"
	if idx > 0 && isWordChar(line[idx-1]) {
		return false
	}
	after := line[idx+len("fmt.Print"):]
	// Valid continuations: ( ln( f(
	if strings.HasPrefix(after, "(") || strings.HasPrefix(after, "ln") || strings.HasPrefix(after, "f") {
		return true
	}
	return false
}

// containsPyPrint checks for print( at statement level
func containsPyPrint(line string) bool {
	t := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(t, "print") {
		return false
	}
	rest := t[len("print"):]
	rest = strings.TrimLeft(rest, " \t")
	return len(rest) > 0 && rest[0] == '('
}

// hasPyMainGuard checks for "if __name__ == "__main__":"
func hasPyMainGuard(line string) bool {
	t := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(t, "if ") && !strings.HasPrefix(t, "if\t") {
		return false
	}
	return strings.Contains(t, "__name__") && strings.Contains(t, "__main__") && strings.Contains(t, ":")
}

// isEmptyGoFunc checks for "func ...() {}" on a single line
func isEmptyGoFunc(trimmed string) bool {
	if !strings.HasPrefix(trimmed, "func ") {
		return false
	}
	if !strings.HasSuffix(trimmed, "{}") && !strings.HasSuffix(trimmed, "{ }") {
		return false
	}
	return true
}

func detectGoDebugPrints(content, relPath string, lines []string) []engine.Diagnostic {
	var diags []engine.Diagnostic

	base := filepath.Base(relPath)
	if strings.Contains(strings.ToLower(base), "log") {
		return nil
	}
	if IsNonProductionPath(relPath) {
		return nil
	}
	// Output/terminal packages legitimately use fmt.Print for CLI display
	if strings.Contains(relPath, "output/") || strings.Contains(relPath, "cmd/") {
		return nil
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		if containsFmtPrint(trimmed) {
			diags = append(diags, engine.Diagnostic{
				FilePath: relPath,
				Engine:   engine.EngineAntislop,
				Rule:     "antislop/debug-print",
				Severity: engine.SeverityWarning,
				Message:  "fmt.Print statement left in production code",
				Help:     "Remove debugging print statements or replace with a structured logger",
				Line:     i + 1,
				Category: "Antislop",
				Fixable:  true,
			})
		}
	}
	return diags
}

func detectPyDebugPrints(content, relPath, basename string, lines []string) []engine.Diagnostic {
	var diags []engine.Diagnostic

	if isTestFilePy(relPath, basename) || isScriptOrEntrypoint(basename) {
		return nil
	}
	if IsNonProductionPath(relPath) {
		return nil
	}
	// Files with __main__ guard treat print() as CLI output
	for _, l := range lines {
		if hasPyMainGuard(l) {
			return nil
		}
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if containsPyPrint(line) {
			diags = append(diags, engine.Diagnostic{
				FilePath: relPath,
				Engine:   engine.EngineAntislop,
				Rule:     "antislop/python-print-debug",
				Severity: engine.SeverityWarning,
				Message:  "`print()` in production code — usually a leftover debug statement",
				Help:     "Use the project's logger (`logging.getLogger(__name__).info(...)`)",
				Line:     i + 1,
				Category: "Antislop",
				Fixable:  true,
			})
		}
	}
	return diags
}

func detectTodoStubs(lines []string, relPath string) []engine.Diagnostic {
	var diags []engine.Diagnostic

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "#") &&
			!strings.HasPrefix(trimmed, "*") && !strings.HasPrefix(trimmed, "/*") {
			continue
		}
		// Extract words and check against todo keywords
		words := strings.Fields(trimmed)
		for _, w := range words {
			// Strip comment markers and punctuation
			w = strings.TrimLeft(w, "/#*")
			w = strings.TrimRight(w, ":.,;!?")
			if TodoKeywords[strings.ToUpper(w)] {
				diags = append(diags, engine.Diagnostic{
					FilePath: relPath,
					Engine:   engine.EngineAntislop,
					Rule:     "antislop/todo-stub",
					Severity: engine.SeverityInfo,
					Message:  "Unresolved TODO/FIXME/HACK comment indicates incomplete code",
					Help:     "Resolve the TODO or create a tracked issue for it",
					Line:     i + 1,
					Category: "Antislop",
				})
				break
			}
		}
	}
	return diags
}

func detectGoEmptyFunctions(lines []string, relPath string) []engine.Diagnostic {
	var diags []engine.Diagnostic

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isEmptyGoFunc(trimmed) {
			diags = append(diags, engine.Diagnostic{
				FilePath: relPath,
				Engine:   engine.EngineAntislop,
				Rule:     "antislop/empty-function",
				Severity: engine.SeverityInfo,
				Message:  "Empty function body — possible stub or unfinished implementation",
				Help:     "Implement the function body or add a comment explaining why it's empty",
				Line:     i + 1,
				Category: "Antislop",
			})
		}
	}
	return diags
}

func DetectDeadPatterns(ctx engine.EngineContext) []engine.Diagnostic {
	var diags []engine.Diagnostic

	for _, filePath := range ctx.Files {
		ext := filepath.Ext(filePath)
		if ext != ".go" && ext != ".py" {
			continue
		}
		content, err := readFile(filePath)
		if err != nil {
			continue
		}
		relPath := relativePath(ctx.RootDir, filePath)
		if isAutoGenerated(content) || isIgnoredFile(content) {
			continue
		}
		lines := strings.Split(content, "\n")

		diags = append(diags, detectTodoStubs(lines, relPath)...)

		switch ext {
		case ".go":
			diags = append(diags, detectGoDebugPrints(content, relPath, lines)...)
			diags = append(diags, detectGoEmptyFunctions(lines, relPath)...)
		case ".py":
			basename := filepath.Base(filePath)
			diags = append(diags, detectPyDebugPrints(content, relPath, basename, lines)...)
		}
	}

	return diags
}

func isTestFilePy(relPath, basename string) bool {
	if strings.HasPrefix(basename, "test_") || strings.HasSuffix(basename, "_test.py") || basename == "conftest.py" {
		return true
	}
	parts := strings.Split(filepath.ToSlash(relPath), "/")
	for _, seg := range parts {
		if seg == "tests" || seg == "test" {
			return true
		}
	}
	return false
}

func isScriptOrEntrypoint(basename string) bool {
	return basename == "__main__.py" || basename == "manage.py" || basename == "setup.py"
}
