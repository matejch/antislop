package antislop

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/matej/antislop/engine"
)

var aiNamingPrefixes = map[string]bool{
	"helper": true, "util": true, "handler": true, "process": true,
	"do": true, "handle": true, "execute": true, "perform": true,
}

var aiNamingBases = map[string]bool{
	"data": true, "temp": true, "result": true, "value": true,
	"item": true, "obj": true, "arr": true, "str": true,
	"num": true, "val": true,
}

var frameworkMethods = map[string]bool{
	"setUp": true, "tearDown": true, "setUpClass": true, "tearDownClass": true,
	"setUpModule": true, "tearDownModule": true,
}

func isDunder(name string) bool {
	if len(name) < 5 {
		return false
	}
	if !strings.HasPrefix(name, "__") || !strings.HasSuffix(name, "__") {
		return false
	}
	mid := name[2 : len(name)-2]
	if len(mid) == 0 {
		return false
	}
	for i := 0; i < len(mid); i++ {
		if !isWordChar(mid[i]) {
			return false
		}
	}
	return true
}

// isReturnCallLine checks if a trimmed line is "return word(...)" or "return word.word(...)"
func isReturnCallLine(trimmed string) bool {
	if !strings.HasPrefix(trimmed, "return ") && !strings.HasPrefix(trimmed, "return\t") {
		return false
	}
	rest := strings.TrimLeft(trimmed[6:], " \t")
	// Extract called function (may be dotted)
	i := 0
	for i < len(rest) && (isWordChar(rest[i]) || rest[i] == '.') {
		i++
	}
	if i == 0 {
		return false
	}
	rest = rest[i:]
	rest = strings.TrimLeft(rest, " \t")
	if len(rest) == 0 || rest[0] != '(' {
		return false
	}
	return true
}

// extractGoFuncNameFromLine extracts func name from a line like "func Name(" or "func (r *T) Name("
func extractGoFuncNameFromLine(trimmed string) string {
	if !strings.HasPrefix(trimmed, "func ") && !strings.HasPrefix(trimmed, "func\t") {
		return ""
	}
	rest := strings.TrimLeft(trimmed[4:], " \t")
	// Skip optional receiver
	if len(rest) > 0 && rest[0] == '(' {
		depth := 0
		i := 0
		for i < len(rest) {
			if rest[i] == '(' {
				depth++
			} else if rest[i] == ')' {
				depth--
				if depth == 0 {
					rest = strings.TrimLeft(rest[i+1:], " \t")
					break
				}
			}
			i++
		}
	}
	return extractWordAt(rest, 0)
}

func detectThinWrappers(content, relPath, ext string) []engine.Diagnostic {
	var diags []engine.Diagnostic
	lines := strings.Split(content, "\n")

	switch ext {
	case ".go":
		diags = detectGoThinWrappers(lines, relPath)
	case ".py":
		diags = detectPyThinWrappers(lines, relPath)
	}
	return diags
}

func detectGoThinWrappers(lines []string, relPath string) []engine.Diagnostic {
	var diags []engine.Diagnostic

	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		funcName := extractGoFuncNameFromLine(trimmed)
		if funcName == "" {
			continue
		}
		// Check if function body is just: return something(...)
		// Need to find the opening { on this line or next
		if !strings.Contains(trimmed, "{") {
			continue
		}
		// Find next non-blank line
		j := i + 1
		for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
			j++
		}
		if j >= len(lines) {
			continue
		}
		bodyLine := strings.TrimSpace(lines[j])
		if !isReturnCallLine(bodyLine) {
			continue
		}
		// Find next non-blank line — should be closing }
		k := j + 1
		for k < len(lines) && strings.TrimSpace(lines[k]) == "" {
			k++
		}
		if k >= len(lines) || strings.TrimSpace(lines[k]) != "}" {
			continue
		}

		// Apply filters
		if isDunder(funcName) || frameworkMethods[funcName] {
			continue
		}
		if i >= 1 && strings.HasPrefix(strings.TrimSpace(lines[i-1]), "@") {
			continue
		}
		matchText := strings.Join(lines[i:k+1], "\n")
		if hasHardcodedArgs(matchText) {
			continue
		}

		diags = append(diags, engine.Diagnostic{
			FilePath: relPath,
			Engine:   engine.EngineAntislop,
			Rule:     "antislop/thin-wrapper",
			Severity: engine.SeverityWarning,
			Message:  fmt.Sprintf("Function '%s' is a thin wrapper that only calls another function", funcName),
			Help:     "Consider calling the inner function directly instead of wrapping it",
			Line:     i + 1,
			Category: "Antislop",
		})
	}
	return diags
}

func detectPyThinWrappers(lines []string, relPath string) []engine.Diagnostic {
	var diags []engine.Diagnostic

	for i := 0; i < len(lines); i++ {
		funcName, ok := isPyDefLine(lines[i])
		if !ok {
			continue
		}
		trimmed := strings.TrimSpace(lines[i])
		if !strings.HasSuffix(trimmed, ":") {
			continue
		}
		// Next non-blank line should be "return something(...)"
		j := i + 1
		for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
			j++
		}
		if j >= len(lines) {
			continue
		}
		bodyLine := strings.TrimSpace(lines[j])
		if !isReturnCallLine(bodyLine) {
			continue
		}
		indent := len(lines[i]) - len(strings.TrimLeft(lines[i], " \t"))
		bodyEnd := j + 1
		for bodyEnd < len(lines) {
			bt := strings.TrimSpace(lines[bodyEnd])
			if bt == "" {
				bodyEnd++
				continue
			}
			li := len(lines[bodyEnd]) - len(strings.TrimLeft(lines[bodyEnd], " \t"))
			if li <= indent {
				break
			}
			// There's more code in the body — not a thin wrapper
			bodyEnd = -1
			break
		}
		if bodyEnd == -1 {
			continue
		}

		if isDunder(funcName) || frameworkMethods[funcName] {
			continue
		}
		if i >= 1 && strings.HasPrefix(strings.TrimSpace(lines[i-1]), "@") {
			continue
		}
		matchText := strings.Join(lines[i:j+1], "\n")
		if hasHardcodedArgs(matchText) {
			continue
		}

		diags = append(diags, engine.Diagnostic{
			FilePath: relPath,
			Engine:   engine.EngineAntislop,
			Rule:     "antislop/thin-wrapper",
			Severity: engine.SeverityWarning,
			Message:  fmt.Sprintf("Function '%s' is a thin wrapper that only calls another function", funcName),
			Help:     "Consider calling the inner function directly instead of wrapping it",
			Line:     i + 1,
			Category: "Antislop",
		})
	}
	return diags
}

func hasHardcodedArgs(matchText string) bool {
	idx := strings.Index(matchText, "return ")
	if idx < 0 {
		return false
	}
	rest := matchText[idx+7:]
	parenIdx := strings.Index(rest, "(")
	if parenIdx < 0 {
		return false
	}
	closeIdx := strings.LastIndex(rest, ")")
	if closeIdx <= parenIdx {
		return false
	}
	args := rest[parenIdx+1 : closeIdx]
	if strings.ContainsAny(args, "\"'`") {
		return true
	}
	for _, tok := range strings.Split(args, ",") {
		tok = strings.TrimSpace(tok)
		if len(tok) > 0 && tok[0] >= '0' && tok[0] <= '9' {
			allDigits := true
			for _, c := range tok {
				if c < '0' || c > '9' {
					allDigits = false
					break
				}
			}
			if allDigits {
				return true
			}
		}
	}
	return false
}

func isAINaming(name string) bool {
	lower := strings.ToLower(name)
	for prefix := range aiNamingPrefixes {
		if !strings.HasPrefix(lower, prefix) {
			continue
		}
		rest := lower[len(prefix):]
		if rest == "" {
			continue
		}
		if rest[0] == '_' {
			rest = rest[1:]
		}
		if len(rest) > 0 && rest[0] >= '0' && rest[0] <= '9' {
			return true
		}
	}
	for base := range aiNamingBases {
		if !strings.HasPrefix(lower, base) {
			continue
		}
		rest := lower[len(base):]
		if len(rest) > 0 && rest[0] >= '0' && rest[0] <= '9' {
			return true
		}
	}
	return false
}

func detectAINaming(content, relPath string) []engine.Diagnostic {
	var diags []engine.Diagnostic
	declKeywords := []string{"const", "var", "func", "def"}
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		kw := startsWithKeyword(trimmed, declKeywords)
		if kw == "" {
			continue
		}
		rest := strings.TrimLeft(trimmed[len(kw):], " \t")
		name := extractWordAt(rest, 0)
		if name == "" {
			continue
		}
		if isAINaming(name) {
			diags = append(diags, engine.Diagnostic{
				FilePath: relPath,
				Engine:   engine.EngineAntislop,
				Rule:     "antislop/generic-naming",
				Severity: engine.SeverityInfo,
				Message:  fmt.Sprintf("'%s' uses generic AI-style naming", name),
				Help:     "Use descriptive names that explain what the code does",
				Line:     i + 1,
				Category: "Antislop",
			})
		}
	}
	return diags
}

func DetectOverAbstraction(ctx engine.EngineContext) []engine.Diagnostic {
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
		diags = append(diags, detectThinWrappers(content, relPath, ext)...)
		diags = append(diags, detectAINaming(content, relPath)...)
	}

	return diags
}
