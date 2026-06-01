package antislop

import (
	"strings"
	"unicode/utf8"
)

var decorativeChars = map[rune]bool{
	'-': true, '=': true, '─': true, '━': true, '~': true,
	'_': true, '*': true, '#': true,
}

var crossRefPhrases = []string{
	"will then be", "used by", "called from", "called later",
	"replaces the", "we moved", "we used to",
}
var crossRefSeePhrase = []string{"see above", "see below", "see later", "see earlier"}
var crossRefRefactorPrefixes = []string{"refactored from", "refactor from"}

var justificationStartPhrases = []string{
	"the idea here", "the trick is", "this was needed", "originally",
}
var justificationThisNouns = []string{
	"function", "method", "class", "module", "component",
	"hook", "util", "helper", "handler", "service",
}
var justificationItVerbs = []string{
	"does", "handles", "takes", "returns", "processes", "reads", "writes",
	"sends", "fetches", "loads", "creates", "deletes", "updates", "parses", "validates",
}
var narrativeStepWords = []string{
	"first", "then", "finally", "next", "lastly", "subsequently",
}

var explanatoryOpenerVerbs = map[string]bool{
	"Matches": true, "Detects": true, "Represents": true, "Holds": true,
	"Stores": true, "Tracks": true, "Handles": true, "Manages": true,
	"Controls": true, "Contains": true, "Captures": true, "Encapsulates": true,
	"Wraps": true, "Describes": true,
}

var whyMarkerWords = map[string]bool{
	"because": true, "since": true, "otherwise": true, "workaround": true,
	"caveat": true, "warning": true, "important": true, "assume": true,
	"assumes": true, "bug": true, "issue": true, "necessary": true,
	"prevents": true, "prevent": true, "guarantee": true, "guarantees": true,
	"guaranteed": true, "regardless": true,
}

var whyMarkerPhrases = []string{
	"note:", "reason:", "hack for", "fix for", "to avoid", "to ensure",
	"to prevent", "in order to", "must run", "must be", "has to be",
	"required for", "required to", "required by", "for example",
	"see issue", "see above", "see below", "in prod", "in production",
	"breaks when", "break when", "fails when", "fail when",
	"useful for", "useful when", "intended to", "on purpose", "by design",
	"e.g.", "i.e.",
}

var goKeywordsNarr = map[string]bool{
	"return": true, "if": true, "for": true, "switch": true, "case": true,
	"default": true, "go": true, "select": true, "defer": true, "else": true,
	"break": true, "continue": true, "package": true, "import": true,
	"map": true, "chan": true, "range": true,
}

var goDeclKeywords = []string{"func", "type", "var", "const", "import"}

func isDecorativeSeparator(s string) bool {
	count := utf8.RuneCountInString(s)
	if count < 6 {
		return false
	}
	for _, r := range s {
		if !decorativeChars[r] {
			return false
		}
	}
	return true
}

func isDecorativeSectionHeader(s string) bool {
	runes := []rune(s)
	if len(runes) < 7 {
		return false
	}
	startDec := 0
	for startDec < len(runes) && decorativeChars[runes[startDec]] {
		startDec++
	}
	if startDec < 3 {
		return false
	}
	endDec := len(runes) - 1
	for endDec >= 0 && decorativeChars[runes[endDec]] {
		endDec--
	}
	if len(runes)-1-endDec < 3 {
		return false
	}
	return endDec >= startDec
}

func isSectionHeader(s string) bool {
	lower := strings.ToLower(s)
	prefixes := []string{"phase ", "step ", "section ", "part "}
	for _, p := range prefixes {
		if !strings.HasPrefix(lower, p) {
			continue
		}
		rest := lower[len(p):]
		if len(rest) == 0 || rest[0] < '0' || rest[0] > '9' {
			continue
		}
		i := 0
		for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
			i++
		}
		if i < len(rest) && (rest[i] == ':' || rest[i] == '.' || rest[i] == '-') {
			return true
		}
	}
	return false
}

func containsCrossRefPhrase(text string) bool {
	lower := strings.ToLower(text)
	for _, p := range crossRefPhrases {
		if strings.Contains(lower, p) {
			return true
		}
	}
	for _, p := range crossRefSeePhrase {
		if strings.Contains(lower, p) {
			return true
		}
	}
	for _, p := range crossRefRefactorPrefixes {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func hasWhyMarker(text string) bool {
	lower := strings.ToLower(text)
	for _, p := range whyMarkerPhrases {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return containsAnyWord(lower, whyMarkerWords)
}

func isExplanatoryOpener(line string) bool {
	word := extractFirstWord(line)
	if !explanatoryOpenerVerbs[word] {
		return false
	}
	rest := line[len(word):]
	if len(rest) == 0 || rest[0] != ' ' {
		return false
	}
	rest = rest[1:]
	if len(rest) == 0 {
		return false
	}
	c := rest[0]
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '`' || c == '\'' || c == '"'
}

func hasJustificationOpener(line string) bool {
	lower := strings.ToLower(line)
	for _, p := range justificationStartPhrases {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	if strings.HasPrefix(lower, "this ") {
		rest := lower[5:]
		for _, noun := range justificationThisNouns {
			if strings.HasPrefix(rest, noun) {
				return true
			}
		}
	}
	if strings.HasPrefix(lower, "it ") {
		rest := lower[3:]
		for _, verb := range justificationItVerbs {
			if strings.HasPrefix(rest, verb) {
				return true
			}
		}
	}
	for _, sw := range narrativeStepWords {
		if !strings.HasPrefix(lower, sw) {
			continue
		}
		rest := strings.TrimLeft(lower[len(sw):], ", ")
		if strings.HasPrefix(rest, "it ") || strings.HasPrefix(rest, "we ") ||
			strings.HasPrefix(rest, "the function") || strings.HasPrefix(rest, "the method") ||
			strings.HasPrefix(rest, "the class") {
			return true
		}
	}
	return false
}

func looksLikeDeclarationPreamble(nextLine, ext string) bool {
	if nextLine == "" {
		return false
	}
	t := strings.TrimLeft(nextLine, " \t")
	switch ext {
	case ".go":
		return startsWithKeyword(t, goDeclKeywords) != ""
	case ".py":
		if strings.HasPrefix(t, "async ") {
			rest := strings.TrimLeft(t[5:], " \t")
			return strings.HasPrefix(rest, "def ")
		}
		return strings.HasPrefix(t, "def ") || strings.HasPrefix(t, "class ")
	}
	return false
}

func looksLikeGoDoc(block commentBlock, ext string) bool {
	if ext != ".go" || len(block.prose) == 0 {
		return false
	}
	nextTrimmed := strings.TrimSpace(block.nextNonBlankLine)
	firstProse := ""
	for _, l := range block.prose {
		if l != "" {
			firstProse = l
			break
		}
	}
	firstWord := extractFirstWord(firstProse)
	if firstWord == "" {
		return false
	}

	t := strings.TrimLeft(nextTrimmed, " \t")
	kw := startsWithKeyword(t, goDeclKeywords)
	if kw != "" {
		rest := strings.TrimLeft(t[len(kw):], " \t")
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
		declName := extractWordAt(rest, 0)
		if declName != "" && firstWord == declName {
			return true
		}
	}

	fieldWords := strings.Fields(nextTrimmed)
	if len(fieldWords) >= 2 && !goKeywordsNarr[fieldWords[0]] && firstWord == fieldWords[0] {
		return true
	}

	return false
}

func hasPreambleSlopSignal(block commentBlock) bool {
	for _, l := range block.prose {
		if isExplanatoryOpener(l) || hasJustificationOpener(l) {
			return true
		}
	}
	return containsCrossRefPhrase(strings.Join(block.prose, " "))
}
