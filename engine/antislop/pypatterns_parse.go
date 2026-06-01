package antislop

import "strings"

func isBareExceptLine(line string) bool {
	t := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(t, "except") {
		return false
	}
	rest := t[6:]
	rest = strings.TrimLeft(rest, " \t")
	if len(rest) == 0 {
		return false
	}
	if rest[0] == ':' {
		after := strings.TrimSpace(rest[1:])
		return after == "" || strings.HasPrefix(after, "#")
	}
	return false
}

func parseBroadExceptLine(line string) string {
	t := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(t, "except ") && !strings.HasPrefix(t, "except\t") {
		return ""
	}
	rest := strings.TrimLeft(t[6:], " \t")
	cls := extractWordAt(rest, 0)
	if cls != "Exception" && cls != "BaseException" {
		return ""
	}
	rest = strings.TrimLeft(rest[len(cls):], " \t")
	if strings.HasPrefix(rest, "as ") || strings.HasPrefix(rest, "as\t") {
		rest = strings.TrimLeft(rest[2:], " \t")
		i := 0
		for i < len(rest) && isWordChar(rest[i]) {
			i++
		}
		rest = strings.TrimLeft(rest[i:], " \t")
	}
	if len(rest) == 0 || rest[0] != ':' {
		return ""
	}
	after := strings.TrimSpace(rest[1:])
	if after == "" || strings.HasPrefix(after, "#") {
		return cls
	}
	return ""
}

func findMutableDefault(signature string) (string, string) {
	for i := 0; i < len(signature); i++ {
		if signature[i] != '=' {
			continue
		}
		if i+1 < len(signature) && signature[i+1] == '=' {
			i++
			continue
		}
		after := strings.TrimLeft(signature[i+1:], " \t\n")
		var defaultVal string
		if strings.HasPrefix(after, "[]") || strings.HasPrefix(after, "[ ]") {
			if strings.HasPrefix(after, "[ ]") {
				defaultVal = "[ ]"
			} else {
				defaultVal = "[]"
			}
		} else if strings.HasPrefix(after, "{}") || strings.HasPrefix(after, "{ }") {
			if strings.HasPrefix(after, "{ }") {
				defaultVal = "{ }"
			} else {
				defaultVal = "{}"
			}
		} else if strings.HasPrefix(after, "set(") {
			rest := after[4:]
			rest = strings.TrimLeft(rest, " \t")
			if len(rest) > 0 && rest[0] == ')' {
				defaultVal = "set()"
			}
		}
		if defaultVal == "" {
			continue
		}

		j := i - 1
		for j >= 0 && (signature[j] == ' ' || signature[j] == '\t' || signature[j] == '\n') {
			j--
		}
		if j >= 0 && isWordChar(signature[j]) {
			wordEnd := j + 1
			wordStart := j
			for wordStart > 0 && (isWordChar(signature[wordStart-1]) || signature[wordStart-1] == '.' || signature[wordStart-1] == '[' || signature[wordStart-1] == ']' || signature[wordStart-1] == ',') {
				wordStart--
			}
			k := wordStart - 1
			for k >= 0 && (signature[k] == ' ' || signature[k] == '\t' || signature[k] == '\n') {
				k--
			}
			if k >= 0 && signature[k] == ':' {
				k--
				for k >= 0 && (signature[k] == ' ' || signature[k] == '\t' || signature[k] == '\n') {
					k--
				}
				nameEnd := k + 1
				for k >= 0 && isWordChar(signature[k]) {
					k--
				}
				nameStart := k + 1
				if nameStart < nameEnd {
					return signature[nameStart:nameEnd], defaultVal
				}
			} else {
				_ = wordEnd
				return signature[wordStart:wordEnd], defaultVal
			}
		}
	}
	return "", ""
}

func parseRangeLenLoop(line string) (string, string) {
	t := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(t, "for ") {
		return "", ""
	}
	rest := t[4:]
	varName := extractWordAt(rest, 0)
	if varName == "" {
		return "", ""
	}
	rest = strings.TrimLeft(rest[len(varName):], " \t")
	if !strings.HasPrefix(rest, "in ") && !strings.HasPrefix(rest, "in\t") {
		return "", ""
	}
	rest = strings.TrimLeft(rest[2:], " \t")
	if !strings.HasPrefix(rest, "range") {
		return "", ""
	}
	rest = strings.TrimLeft(rest[5:], " \t")
	if len(rest) == 0 || rest[0] != '(' {
		return "", ""
	}
	rest = strings.TrimLeft(rest[1:], " \t")
	if !strings.HasPrefix(rest, "len") {
		return "", ""
	}
	rest = strings.TrimLeft(rest[3:], " \t")
	if len(rest) == 0 || rest[0] != '(' {
		return "", ""
	}
	rest = strings.TrimLeft(rest[1:], " \t")
	i := 0
	for i < len(rest) && (isWordChar(rest[i]) || rest[i] == '.') {
		i++
	}
	if i == 0 {
		return "", ""
	}
	collection := rest[:i]
	rest = strings.TrimLeft(rest[i:], " \t")
	if !strings.HasPrefix(rest, ")") {
		return "", ""
	}
	rest = strings.TrimLeft(rest[1:], " \t")
	if !strings.HasPrefix(rest, ")") {
		return "", ""
	}
	rest = strings.TrimLeft(rest[1:], " \t")
	if !strings.HasPrefix(rest, ":") {
		return "", ""
	}
	return varName, collection
}

func hasChainedDictGet(line string) bool {
	idx := 0
	for {
		pos := strings.Index(line[idx:], ".get")
		if pos < 0 {
			return false
		}
		pos += idx
		after := pos + 4
		for after < len(line) && (line[after] == ' ' || line[after] == '\t') {
			after++
		}
		if after >= len(line) || line[after] != '(' {
			idx = pos + 4
			continue
		}
		depth := 1
		i := after + 1
		for i < len(line) && depth > 0 {
			if line[i] == '(' {
				depth++
			} else if line[i] == ')' {
				depth--
			}
			if depth > 0 {
				i++
			}
		}
		if depth != 0 {
			idx = pos + 4
			continue
		}
		args := line[after+1 : i]
		trimmedArgs := strings.TrimSpace(args)
		if strings.HasSuffix(trimmedArgs, ", {}") || strings.HasSuffix(trimmedArgs, ",{}") ||
			strings.HasSuffix(trimmedArgs, ", { }") {
			rest := strings.TrimLeft(line[i+1:], " \t")
			if strings.HasPrefix(rest, ".get") {
				after2 := 4
				for after2 < len(rest) && (rest[after2] == ' ' || rest[after2] == '\t') {
					after2++
				}
				if after2 < len(rest) && rest[after2] == '(' {
					return true
				}
			}
		}
		idx = pos + 4
	}
}

type branchMatch struct {
	indent   string
	selector string
}

func parseSameValueBranch(line string) (branchMatch, bool) {
	indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	t := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(t, "if ") && !strings.HasPrefix(t, "elif ") {
		return branchMatch{}, false
	}
	if strings.HasPrefix(t, "if ") {
		t = t[3:]
	} else {
		t = t[5:]
	}
	t = strings.TrimLeft(t, " \t")
	i := 0
	for i < len(t) && (isWordChar(t[i]) || t[i] == '.') {
		i++
	}
	if i == 0 {
		return branchMatch{}, false
	}
	selector := t[:i]
	rest := strings.TrimLeft(t[i:], " \t")
	if !strings.HasPrefix(rest, "==") {
		return branchMatch{}, false
	}
	rest = strings.TrimLeft(rest[2:], " \t")
	if len(rest) == 0 || (rest[0] != '\'' && rest[0] != '"') {
		return branchMatch{}, false
	}
	quote := rest[0]
	end := strings.IndexByte(rest[1:], quote)
	if end < 0 {
		return branchMatch{}, false
	}
	rest = strings.TrimLeft(rest[end+2:], " \t")
	if !strings.HasPrefix(rest, ":") {
		return branchMatch{}, false
	}
	return branchMatch{indent: indent, selector: selector}, true
}

func parseIsinstanceBranch(line string) (branchMatch, bool) {
	indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	t := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(t, "if ") && !strings.HasPrefix(t, "elif ") {
		return branchMatch{}, false
	}
	if strings.HasPrefix(t, "if ") {
		t = t[3:]
	} else {
		t = t[5:]
	}
	t = strings.TrimLeft(t, " \t")
	if !strings.HasPrefix(t, "isinstance") {
		return branchMatch{}, false
	}
	t = strings.TrimLeft(t[10:], " \t")
	if len(t) == 0 || t[0] != '(' {
		return branchMatch{}, false
	}
	t = strings.TrimLeft(t[1:], " \t")
	i := 0
	for i < len(t) && isWordChar(t[i]) {
		i++
	}
	if i == 0 {
		return branchMatch{}, false
	}
	selector := t[:i]
	rest := strings.TrimLeft(t[i:], " \t")
	if len(rest) == 0 || rest[0] != ',' {
		return branchMatch{}, false
	}
	closeIdx := strings.LastIndex(rest, ")")
	if closeIdx < 0 {
		return branchMatch{}, false
	}
	after := strings.TrimLeft(rest[closeIdx+1:], " \t")
	if !strings.HasPrefix(after, ":") {
		return branchMatch{}, false
	}
	return branchMatch{indent: indent, selector: selector}, true
}

type branchParser func(line string) (branchMatch, bool)

func countBranchLadderImperative(lines []string, start int, parser branchParser, selector, indent string) int {
	count := 1
	for i := start + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		m, ok := parser(lines[i])
		if ok && m.indent == indent && m.selector == selector {
			count++
			continue
		}
		if strings.HasPrefix(lines[i], indent+"elif ") {
			break
		}
		lineIndent := len(lines[i]) - len(strings.TrimLeft(lines[i], " \t"))
		if lineIndent <= len(indent) && !strings.HasPrefix(lines[i], indent+"else") {
			break
		}
	}
	return count
}
