package security

import (
	"os"
	"strings"

	"github.com/matej/antislop/engine"
)

var placeholderExact = map[string]bool{
	"changeme": true, "password": true, "secret": true,
	"xxx": true, "todo": true, "replace_me": true,
}

func isPlaceholder(matched string) bool {
	if strings.Contains(strings.ToLower(matched), "env(") {
		return true
	}
	if strings.Contains(matched, "os.environ") || strings.Contains(matched, "os.Getenv") {
		return true
	}
	if strings.Contains(matched, "${") || (strings.Contains(matched, "<") && strings.Contains(matched, ">")) {
		return true
	}
	if placeholderExact[strings.ToLower(matched)] {
		return true
	}
	return false
}

// Character set checkers
func isAlphaNum(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}
func isAlphaNumDash(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_' || b == '-'
}
func isAlphaNumPlus(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '/' || b == '+' || b == '='
}
func isUpperAlphaNum(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}
func isSecWordChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

func scanQuotedValue(content string, pos int) (string, int, bool) {
	i := pos
	// Skip whitespace
	for i < len(content) && (content[i] == ' ' || content[i] == '\t') {
		i++
	}
	// Expect : or =
	if i >= len(content) || (content[i] != ':' && content[i] != '=') {
		return "", 0, false
	}
	i++
	// Skip whitespace
	for i < len(content) && (content[i] == ' ' || content[i] == '\t') {
		i++
	}
	// Expect quote
	if i >= len(content) || (content[i] != '"' && content[i] != '\'') {
		return "", 0, false
	}
	quote := content[i]
	i++
	start := i
	for i < len(content) && content[i] != quote {
		if content[i] == '\\' {
			i++
		}
		i++
	}
	if i >= len(content) {
		return "", 0, false
	}
	return content[start:i], i + 1, true
}

// scanTokenWithCharset scans forward from pos collecting bytes that satisfy check.
func scanTokenWithCharset(content string, pos int, check func(byte) bool) string {
	end := pos
	for end < len(content) && check(content[end]) {
		end++
	}
	return content[pos:end]
}

type secretMatch struct {
	index int
	value string
}

// Keyword-assignment patterns: api_key = "...", password = "...", etc.
var keywordAssignmentPatterns = []struct {
	keywords []string
	name     string
	minLen   int
	charset  func(byte) bool
}{
	{[]string{"api_key", "api-key", "apikey"}, "API key", 20, isAlphaNumDash},
	{[]string{"aws_secret", "aws-secret", "secret_key", "secret-key"}, "AWS Secret Key", 40, isAlphaNumPlus},
	{[]string{"password", "passwd", "pwd", "secret"}, "Hardcoded password/secret", 8, nil},
	{[]string{"token", "bearer"}, "Authentication token", 20, isAlphaNumDash},
}

func scanKeywordAssignments(content string) []secretMatch {
	var results []secretMatch
	lower := strings.ToLower(content)

	for _, pat := range keywordAssignmentPatterns {
		for _, kw := range pat.keywords {
			start := 0
			for {
				idx := strings.Index(lower[start:], kw)
				if idx < 0 {
					break
				}
				idx += start
				// Must be at a word boundary
				if idx > 0 && isSecWordChar(content[idx-1]) && content[idx-1] != '_' && content[idx-1] != '-' {
					start = idx + len(kw)
					continue
				}
				afterKw := idx + len(kw)
				// Skip optional _- chars that are part of the keyword name
				for afterKw < len(content) && (content[afterKw] == '_' || content[afterKw] == '-' || isSecWordChar(content[afterKw])) {
					afterKw++
				}
				value, _, ok := scanQuotedValue(content, afterKw)
				if !ok {
					start = idx + len(kw)
					continue
				}
				if pat.minLen > 0 && len(value) < pat.minLen {
					start = idx + len(kw)
					continue
				}
				if pat.charset != nil {
					valid := true
					for i := 0; i < len(value); i++ {
						if !pat.charset(value[i]) {
							valid = false
							break
						}
					}
					if !valid {
						start = idx + len(kw)
						continue
					}
				}
				results = append(results, secretMatch{index: idx, value: value})
				start = idx + len(kw)
			}
		}
	}
	return results
}

func scanAWSAccessKey(content string) []secretMatch {
	var results []secretMatch
	start := 0
	for {
		idx := strings.Index(content[start:], "AKIA")
		if idx < 0 {
			break
		}
		idx += start
		after := idx + 4
		if after+16 > len(content) {
			break
		}
		token := content[after : after+16]
		valid := true
		for i := 0; i < len(token); i++ {
			if !isUpperAlphaNum(token[i]) {
				valid = false
				break
			}
		}
		if valid {
			results = append(results, secretMatch{index: idx, value: "AKIA" + token})
		}
		start = idx + 4
	}
	return results
}

func scanPrivateKey(content string) []secretMatch {
	var results []secretMatch
	start := 0
	marker := "-----BEGIN "
	for {
		idx := strings.Index(content[start:], marker)
		if idx < 0 {
			break
		}
		idx += start
		rest := content[idx+len(marker):]
		if strings.HasPrefix(rest, "RSA PRIVATE KEY-----") ||
			strings.HasPrefix(rest, "EC PRIVATE KEY-----") ||
			strings.HasPrefix(rest, "DSA PRIVATE KEY-----") ||
			strings.HasPrefix(rest, "PRIVATE KEY-----") {
			results = append(results, secretMatch{index: idx, value: ""})
		}
		start = idx + len(marker)
	}
	return results
}

func scanJWT(content string) []secretMatch {
	var results []secretMatch
	start := 0
	for {
		idx := strings.Index(content[start:], "eyJ")
		if idx < 0 {
			break
		}
		idx += start
		pos := idx
		var segLens [3]int
		seg := 0
		ok := true
		for pos < len(content) && seg < 3 {
			b := content[pos]
			if isAlphaNumDash(b) {
				segLens[seg]++
				pos++
			} else if b == '.' && seg < 2 {
				if segLens[seg] < 10 {
					ok = false
					break
				}
				seg++
				pos++
			} else {
				break
			}
		}
		if ok && seg == 2 && segLens[0] >= 10 && segLens[1] >= 10 && segLens[2] >= 10 {
			results = append(results, secretMatch{index: idx, value: content[idx:pos]})
		}
		start = idx + 3
	}
	return results
}

func scanGitHubToken(content string) []secretMatch {
	var results []secretMatch
	prefixes := []string{"ghp_", "gho_", "ghu_", "ghs_", "ghr_"}
	for _, prefix := range prefixes {
		start := 0
		for {
			idx := strings.Index(content[start:], prefix)
			if idx < 0 {
				break
			}
			idx += start
			token := scanTokenWithCharset(content, idx+4, func(b byte) bool {
				return isAlphaNum(b) || b == '_'
			})
			if len(token) >= 36 {
				results = append(results, secretMatch{index: idx, value: prefix + token})
			}
			start = idx + 4
		}
	}
	return results
}

func scanSlackToken(content string) []secretMatch {
	var results []secretMatch
	prefixes := []string{"xoxb-", "xoxa-", "xoxp-", "xoxr-", "xoxs-"}
	for _, prefix := range prefixes {
		start := 0
		for {
			idx := strings.Index(content[start:], prefix)
			if idx < 0 {
				break
			}
			idx += start
			token := scanTokenWithCharset(content, idx+5, func(b byte) bool {
				return isAlphaNum(b) || b == '-'
			})
			if len(token) > 0 {
				results = append(results, secretMatch{index: idx, value: prefix + token})
			}
			start = idx + 5
		}
	}
	return results
}

func scanDBConnectionString(content string) []secretMatch {
	var results []secretMatch
	schemes := []string{"mongodb://", "postgres://", "mysql://", "redis://"}
	lower := strings.ToLower(content)
	for _, scheme := range schemes {
		start := 0
		for {
			idx := strings.Index(lower[start:], scheme)
			if idx < 0 {
				break
			}
			idx += start
			rest := content[idx+len(scheme):]
			// Look for user:pass@ pattern (non-whitespace, non-quote chars with : and @)
			atIdx := strings.Index(rest, "@")
			if atIdx < 0 {
				start = idx + len(scheme)
				continue
			}
			userPass := rest[:atIdx]
			if strings.ContainsAny(userPass, "\"' \t\n") {
				start = idx + len(scheme)
				continue
			}
			if strings.Contains(userPass, ":") {
				results = append(results, secretMatch{index: idx, value: ""})
			}
			start = idx + len(scheme)
		}
	}
	return results
}

var allScanners = []struct {
	name string
	scan func(string) []secretMatch
}{
	{"keyword-assignment", scanKeywordAssignments},
	{"AWS Access Key", scanAWSAccessKey},
	{"Private key", scanPrivateKey},
	{"JWT token", scanJWT},
	{"GitHub token", scanGitHubToken},
	{"Slack token", scanSlackToken},
	{"Database connection string with credentials", scanDBConnectionString},
}

func ScanSecrets(ctx engine.EngineContext) []engine.Diagnostic {
	var diags []engine.Diagnostic

	for _, filePath := range ctx.Files {
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		content := string(data)
		rel := relPath(ctx.RootDir, filePath)

		seen := map[int]bool{} // deduplicate by line

		for _, scanner := range allScanners {
			matches := scanner.scan(content)
			for _, m := range matches {
				if m.value != "" && isPlaceholder(m.value) {
					continue
				}
				line := strings.Count(content[:m.index], "\n") + 1
				if seen[line] {
					continue
				}
				seen[line] = true

				name := scanner.name
				if name == "keyword-assignment" {
					// Determine specific name from the match
					name = identifyKeywordMatch(content, m.index)
				}

				diags = append(diags, engine.Diagnostic{
					FilePath: rel,
					Engine:   engine.EngineSecurity,
					Rule:     "security/hardcoded-secret",
					Severity: engine.SeverityError,
					Message:  "Possible " + name + " detected in source code",
					Help:     "Move secrets to environment variables or a secrets manager",
					Line:     line,
					Category: "Security",
				})
			}
		}
	}
	return diags
}

func identifyKeywordMatch(content string, idx int) string {
	lower := strings.ToLower(content)
	for _, pat := range keywordAssignmentPatterns {
		for _, kw := range pat.keywords {
			if idx+len(kw) <= len(lower) && lower[idx:idx+len(kw)] == kw {
				return pat.name
			}
		}
	}
	return "Hardcoded secret"
}
