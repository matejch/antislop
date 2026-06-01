package antislop

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/matej/antislop/engine"
)

const branchLadderThreshold = 4

func flagBareExcept(lines []string, relPath string) []engine.Diagnostic {
	var diags []engine.Diagnostic
	for i, line := range lines {
		if isBareExceptLine(line) {
			diags = append(diags, engine.Diagnostic{
				FilePath: relPath,
				Engine:   engine.EngineAntislop,
				Rule:     "antislop/python-bare-except",
				Severity: engine.SeverityWarning,
				Message:  "Bare `except:` swallows every exception including KeyboardInterrupt and SystemExit.",
				Help:     "Catch the specific exception type you actually expect.",
				Line:     i + 1,
				Column:   1,
				Category: "Antislop",
			})
		}
	}
	return diags
}

func flagBroadExcept(lines []string, relPath string) []engine.Diagnostic {
	var diags []engine.Diagnostic
	for i, line := range lines {
		cls := parseBroadExceptLine(line)
		if cls == "" {
			continue
		}
		if i+1 >= len(lines) {
			continue
		}
		nextTrimmed := strings.TrimSpace(lines[i+1])
		isSilent := nextTrimmed == "pass" ||
			(strings.HasPrefix(nextTrimmed, "#") && i+2 < len(lines) && strings.TrimSpace(lines[i+2]) == "pass")
		if !isSilent {
			continue
		}
		diags = append(diags, engine.Diagnostic{
			FilePath: relPath,
			Engine:   engine.EngineAntislop,
			Rule:     "antislop/python-broad-except",
			Severity: engine.SeverityWarning,
			Message:  fmt.Sprintf("`except %s: pass` silently drops every exception.", cls),
			Help:     "Either narrow the exception class, log the error, or re-raise.",
			Line:     i + 1,
			Column:   1,
			Category: "Antislop",
		})
	}
	return diags
}

func flagMutableDefaults(lines []string, relPath string) []engine.Diagnostic {
	var diags []engine.Diagnostic

	for i := 0; i < len(lines); i++ {
		if _, ok := isPyDefLine(lines[i]); !ok {
			continue
		}
		startLine := i
		signature := lines[i]
		parenDepth := 0
		for _, ch := range signature {
			if ch == '(' {
				parenDepth++
			} else if ch == ')' {
				parenDepth--
			}
		}
		for parenDepth > 0 && i+1 < len(lines) {
			i++
			signature += "\n" + lines[i]
			for _, ch := range lines[i] {
				if ch == '(' {
					parenDepth++
				} else if ch == ')' {
					parenDepth--
				}
			}
		}

		paramName, defaultVal := findMutableDefault(signature)
		if paramName != "" {
			diags = append(diags, engine.Diagnostic{
				FilePath: relPath,
				Engine:   engine.EngineAntislop,
				Rule:     "antislop/python-mutable-default",
				Severity: engine.SeverityWarning,
				Message:  fmt.Sprintf("Mutable default argument `%s=%s`. The default is shared across all calls.", paramName, defaultVal),
				Help:     "Use `None` as the default and create the mutable value inside the body.",
				Line:     startLine + 1,
				Column:   1,
				Category: "Antislop",
			})
		}
	}
	return diags
}

func flagRangeLenLoops(lines []string, relPath string) []engine.Diagnostic {
	var diags []engine.Diagnostic
	for i, line := range lines {
		_, collection := parseRangeLenLoop(line)
		if collection == "" {
			continue
		}
		diags = append(diags, engine.Diagnostic{
			FilePath: relPath,
			Engine:   engine.EngineAntislop,
			Rule:     "antislop/python-range-len-loop",
			Severity: engine.SeverityInfo,
			Message:  fmt.Sprintf("`range(len(%s))` loop — usually a hand-rolled iteration pattern.", collection),
			Help:     "Prefer direct iteration (`for item in items`) or `enumerate(items)` when the index is needed.",
			Line:     i + 1,
			Column:   1,
			Category: "Antislop",
		})
	}
	return diags
}

func flagChainedDictGets(lines []string, relPath string) []engine.Diagnostic {
	var diags []engine.Diagnostic
	for i, line := range lines {
		if hasChainedDictGet(line) {
			diags = append(diags, engine.Diagnostic{
				FilePath: relPath,
				Engine:   engine.EngineAntislop,
				Rule:     "antislop/python-chained-dict-get",
				Severity: engine.SeverityWarning,
				Message:  "Chained `.get(..., {})` defaults hide missing-data cases.",
				Help:     "Normalize the input at the boundary, use a typed object, or split the lookup into explicit steps.",
				Line:     i + 1,
				Column:   1,
				Category: "Antislop",
			})
		}
	}
	return diags
}

func flagBranchLadders(lines []string, relPath string) []engine.Diagnostic {
	var diags []engine.Diagnostic
	reported := map[int]bool{}

	for i := 0; i < len(lines); i++ {
		if reported[i] {
			continue
		}
		if m, ok := parseSameValueBranch(lines[i]); ok {
			count := countBranchLadderImperative(lines, i, parseSameValueBranch, m.selector, m.indent)
			if count >= branchLadderThreshold {
				reported[i] = true
				diags = append(diags, engine.Diagnostic{
					FilePath: relPath,
					Engine:   engine.EngineAntislop,
					Rule:     "antislop/python-repetitive-dispatch",
					Severity: engine.SeverityWarning,
					Message:  fmt.Sprintf("%d repeated branches dispatch on `%s`.", count, m.selector),
					Help:     "Use a table, set membership, or handler map when branches share the same shape.",
					Line:     i + 1,
					Column:   1,
					Category: "Antislop",
				})
			}
			continue
		}

		if im, ok := parseIsinstanceBranch(lines[i]); ok {
			count := countBranchLadderImperative(lines, i, parseIsinstanceBranch, im.selector, im.indent)
			if count < branchLadderThreshold {
				continue
			}
			reported[i] = true
			diags = append(diags, engine.Diagnostic{
				FilePath: relPath,
				Engine:   engine.EngineAntislop,
				Rule:     "antislop/python-isinstance-ladder",
				Severity: engine.SeverityWarning,
				Message:  fmt.Sprintf("%d repeated `isinstance(%s, ...)` branches.", count, im.selector),
				Help:     "Prefer a handler map, protocol, or normalized intermediate representation.",
				Line:     i + 1,
				Column:   1,
				Category: "Antislop",
			})
		}
	}
	return diags
}

func DetectPythonPatterns(ctx engine.EngineContext) []engine.Diagnostic {
	var diags []engine.Diagnostic

	for _, filePath := range ctx.Files {
		if filepath.Ext(filePath) != ".py" {
			continue
		}
		content, err := readFile(filePath)
		if err != nil {
			continue
		}
		if isAutoGenerated(content) || isIgnoredFile(content) {
			continue
		}

		relPath := relativePath(ctx.RootDir, filePath)
		lines := strings.Split(content, "\n")

		diags = append(diags, flagBareExcept(lines, relPath)...)
		diags = append(diags, flagBroadExcept(lines, relPath)...)
		diags = append(diags, flagMutableDefaults(lines, relPath)...)
		diags = append(diags, flagRangeLenLoops(lines, relPath)...)
		diags = append(diags, flagChainedDictGets(lines, relPath)...)
		diags = append(diags, flagBranchLadders(lines, relPath)...)
	}

	return diags
}
