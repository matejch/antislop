package antislop

import (
	"path/filepath"
	"strings"

	"github.com/matej/antislop/engine"
)

var intentionalIgnoreNames = map[string]bool{
	"ignored": true, "ignore": true, "tolerated": true, "expected": true,
	"unused": true, "_": true, "_e": true, "_err": true,
}

// isPyExceptLine checks if a line is an except clause (except: or except Type:)
func isPyExceptLine(line string) bool {
	t := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(t, "except") {
		return false
	}
	rest := t[6:]
	if len(rest) == 0 {
		return false
	}
	// "except:" or "except SomeType:" or "except SomeType as e:"
	rest = strings.TrimLeft(rest, " \t")
	if rest[0] == ':' {
		return true
	}
	// Has an exception type — scan to the colon
	return strings.Contains(rest, ":")
}

func isGoErrorIgnoreLine(line string) bool {
	idx := strings.Index(line, ", _")
	if idx < 0 {
		return false
	}
	// Check that _ is followed by whitespace then := or =
	after := line[idx+3:]
	after = strings.TrimLeft(after, " \t")
	if !strings.HasPrefix(after, ":=") && !strings.HasPrefix(after, "=") {
		return false
	}
	if strings.HasPrefix(after, ":=") {
		after = after[2:]
	} else {
		// Make sure it's = not ==
		if len(after) > 1 && after[1] == '=' {
			return false
		}
		after = after[1:]
	}
	after = strings.TrimLeft(after, " \t")
	// Expect a word followed by (
	i := 0
	for i < len(after) && isWordChar(after[i]) {
		i++
	}
	if i == 0 {
		return false
	}
	rest := strings.TrimLeft(after[i:], " \t")
	// Allow dotted calls like pkg.Func(
	for strings.HasPrefix(rest, ".") {
		rest = rest[1:]
		j := 0
		for j < len(rest) && isWordChar(rest[j]) {
			j++
		}
		rest = strings.TrimLeft(rest[j:], " \t")
	}
	return len(rest) > 0 && rest[0] == '('
}

func isIntentionalIgnore(line string) bool {
	commaIdx := strings.Index(line, ",")
	if commaIdx < 0 {
		return false
	}
	end := commaIdx
	start := end - 1
	for start >= 0 && isWordChar(line[start]) {
		start--
	}
	start++
	if start >= end {
		return false
	}
	varName := line[start:end]
	return intentionalIgnoreNames[strings.ToLower(varName)]
}

func DetectSwallowedExceptions(ctx engine.EngineContext) []engine.Diagnostic {
	var diags []engine.Diagnostic

	for _, filePath := range ctx.Files {
		ext := filepath.Ext(filePath)
		content, err := readFile(filePath)
		if err != nil {
			continue
		}
		relPath := relativePath(ctx.RootDir, filePath)
		lines := strings.Split(content, "\n")

		switch ext {
		case ".py":
			diags = append(diags, detectPySwallowed(lines, relPath)...)
		case ".go":
			diags = append(diags, detectGoSwallowed(lines, relPath)...)
		}
	}

	return diags
}

func detectPySwallowed(lines []string, relPath string) []engine.Diagnostic {
	var diags []engine.Diagnostic
	for i, line := range lines {
		if !isPyExceptLine(line) {
			continue
		}
		if i+1 >= len(lines) {
			continue
		}
		nextTrimmed := strings.TrimSpace(lines[i+1])
		// except ...: followed by just "pass"
		if nextTrimmed == "pass" {
			diags = append(diags, engine.Diagnostic{
				FilePath: relPath,
				Engine:   engine.EngineAntislop,
				Rule:     "antislop/swallowed-exception",
				Severity: engine.SeverityError,
				Message:  "Bare except with pass swallows errors silently",
				Help:     "Handle errors explicitly: log with context, rethrow, or return an error value",
				Line:     i + 1,
				Category: "Antislop",
			})
			continue
		}
		// except ...: followed by just "print("
		if strings.HasPrefix(nextTrimmed, "print(") {
			diags = append(diags, engine.Diagnostic{
				FilePath: relPath,
				Engine:   engine.EngineAntislop,
				Rule:     "antislop/swallowed-exception",
				Severity: engine.SeverityError,
				Message:  "Catch block only prints error without proper handling",
				Help:     "Handle errors explicitly: log with context, rethrow, or return an error value",
				Line:     i + 1,
				Category: "Antislop",
			})
		}
	}
	return diags
}

func detectGoSwallowed(lines []string, relPath string) []engine.Diagnostic {
	var diags []engine.Diagnostic
	for i, line := range lines {
		if !isGoErrorIgnoreLine(line) {
			continue
		}
		if isIntentionalIgnore(line) {
			continue
		}
		diags = append(diags, engine.Diagnostic{
			FilePath: relPath,
			Engine:   engine.EngineAntislop,
			Rule:     "antislop/swallowed-exception",
			Severity: engine.SeverityError,
			Message:  "Error return value is being ignored",
			Help:     "Handle errors explicitly: log with context, rethrow, or return an error value",
			Line:     i + 1,
			Category: "Antislop",
		})
	}
	return diags
}
