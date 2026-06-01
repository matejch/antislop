package security

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/matej/antislop/engine"
	"github.com/matej/antislop/engine/antislop"
)

func isWordCharRisky(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

type riskyDetector struct {
	detect     func(content string) []int
	extensions []string
	name       string
	message    string
	help       string
}

// detectPickleLoad finds pickle.load( or pickle.loads(
func detectPickleLoad(content string) []int {
	var indices []int
	start := 0
	for {
		idx := strings.Index(content[start:], "pickle.load")
		if idx < 0 {
			break
		}
		idx += start
		after := idx + len("pickle.load")
		if after < len(content) && content[after] == 's' {
			after++
		}
		for after < len(content) && (content[after] == ' ' || content[after] == '\t') {
			after++
		}
		if after < len(content) && content[after] == '(' {
			indices = append(indices, idx)
		}
		start = idx + len("pickle.load")
	}
	return indices
}

// detectExecCall finds exec( with word boundary
func detectExecCall(content string) []int {
	var indices []int
	word := antislop.ExecKeyword
	start := 0
	for {
		idx := strings.Index(content[start:], word)
		if idx < 0 {
			break
		}
		idx += start
		if idx > 0 && isWordCharRisky(content[idx-1]) {
			start = idx + len(word)
			continue
		}
		after := idx + len(word)
		if after < len(content) && content[after] == '.' {
			start = after
			continue
		}
		for after < len(content) && (content[after] == ' ' || content[after] == '\t') {
			after++
		}
		if after < len(content) && content[after] == '(' {
			indices = append(indices, idx)
		}
		start = idx + len(word)
	}
	return indices
}

// detectPyShellInjection finds subprocess/os.system/os.popen calls with ${ in args
func detectPyShellInjection(content string) []int {
	var indices []int
	funcs := []string{"subprocess", "os.system", "os.popen"}
	for _, fn := range funcs {
		start := 0
		for {
			idx := strings.Index(content[start:], fn)
			if idx < 0 {
				break
			}
			idx += start
			after := idx + len(fn)
			// Skip whitespace
			for after < len(content) && (content[after] == ' ' || content[after] == '\t') {
				after++
			}
			// Allow . after subprocess (e.g. subprocess.run)
			if after < len(content) && content[after] == '.' {
				after++
				for after < len(content) && isWordCharRisky(content[after]) {
					after++
				}
				for after < len(content) && (content[after] == ' ' || content[after] == '\t') {
					after++
				}
			}
			if after >= len(content) || content[after] != '(' {
				start = idx + len(fn)
				continue
			}
			// Scan to matching ) and look for ${
			depth := 1
			i := after + 1
			found := false
			for i < len(content) && depth > 0 {
				if content[i] == '(' {
					depth++
				} else if content[i] == ')' {
					depth--
				} else if content[i] == '$' && i+1 < len(content) && content[i+1] == '{' {
					found = true
				}
				i++
			}
			if found {
				indices = append(indices, idx)
			}
			start = idx + len(fn)
		}
	}
	return indices
}

var execCommandLiteral = antislop.ExecCommandPrefix

func detectGoShellInjection(content string) []int {
	var indices []int
	start := 0
	for {
		idx := strings.Index(content[start:], execCommandLiteral)
		if idx < 0 {
			break
		}
		idx += start
		after := idx + len(execCommandLiteral)
		for after < len(content) && (content[after] == ' ' || content[after] == '\t') {
			after++
		}
		if after >= len(content) || content[after] != '(' {
			start = idx + len(execCommandLiteral)
			continue
		}
		// Scan to matching ) and look for fmt.Sprintf
		depth := 1
		i := after + 1
		found := false
		for i < len(content) && depth > 0 {
			if content[i] == '(' {
				depth++
			} else if content[i] == ')' {
				depth--
			}
			if depth > 0 && i+len("fmt.Sprintf") <= len(content) && content[i:i+len("fmt.Sprintf")] == "fmt.Sprintf" {
				found = true
			}
			i++
		}
		if found {
			indices = append(indices, idx)
		}
		start = idx + len(execCommandLiteral)
	}
	return indices
}

// detectGoSQLInjection finds .Query/.Exec/.QueryRow with fmt.Sprintf or string concat
func detectGoSQLInjection(content string) []int {
	var indices []int
	methods := []string{".Query", ".Exec", ".QueryRow"}
	for _, method := range methods {
		start := 0
		for {
			idx := strings.Index(content[start:], method)
			if idx < 0 {
				break
			}
			idx += start
			after := idx + len(method)
			for after < len(content) && (content[after] == ' ' || content[after] == '\t') {
				after++
			}
			if after >= len(content) || content[after] != '(' {
				start = idx + len(method)
				continue
			}
			after++
			// Skip whitespace after (
			for after < len(content) && (content[after] == ' ' || content[after] == '\t') {
				after++
			}
			if after+len("fmt.Sprintf") <= len(content) && content[after:after+len("fmt.Sprintf")] == "fmt.Sprintf" {
				indices = append(indices, idx)
				start = idx + len(method)
				continue
			}
			if after < len(content) && content[after] == '"' {
				// Scan to closing quote
				q := after + 1
				for q < len(content) && content[q] != '"' {
					if content[q] == '\\' {
						q++
					}
					q++
				}
				if q < len(content) {
					rest := strings.TrimLeft(content[q+1:], " \t")
					if strings.HasPrefix(rest, "+") {
						indices = append(indices, idx)
					}
				}
			}
			start = idx + len(method)
		}
	}
	return indices
}

var allRiskyDetectors = []riskyDetector{
	{detectPickleLoad, []string{".py"}, "pickle-load", "pickle.load can execute arbitrary code — unsafe deserialization", "Use JSON, MessagePack, or other safe serialization formats for untrusted data"},
	{detectExecCall, []string{".py"}, "python-exec", "Use of exec() can execute arbitrary code", "Avoid exec — use safer alternatives"},
	{detectPyShellInjection, []string{".py"}, "shell-injection", "Possible shell injection — user input in command execution", "Use parameterized commands or subprocess with list arguments"},
	{detectGoShellInjection, []string{".go"}, "shell-injection", "Possible shell injection — formatted string in command execution", "Pass arguments as separate parameters, not as a formatted string"},
	{detectGoSQLInjection, []string{".go"}, "sql-injection", "Possible SQL injection — string formatting/concatenation in query", "Use parameterized queries with $1, $2 placeholders instead of string formatting"},
}

func DetectRiskyConstructs(ctx engine.EngineContext) []engine.Diagnostic {
	var diags []engine.Diagnostic

	for _, filePath := range ctx.Files {
		ext := filepath.Ext(filePath)
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		content := string(data)
		rel := relPath(ctx.RootDir, filePath)

		for _, det := range allRiskyDetectors {
			if !containsExt(det.extensions, ext) {
				continue
			}
			for _, idx := range det.detect(content) {
				line := strings.Count(content[:idx], "\n") + 1
				diags = append(diags, engine.Diagnostic{
					FilePath: rel,
					Engine:   engine.EngineSecurity,
					Rule:     "security/" + det.name,
					Severity: engine.SeverityError,
					Message:  det.message,
					Help:     det.help,
					Line:     line,
					Category: "Security",
				})
			}
		}
	}
	return diags
}

func containsExt(exts []string, ext string) bool {
	for _, e := range exts {
		if e == ext {
			return true
		}
	}
	return false
}
