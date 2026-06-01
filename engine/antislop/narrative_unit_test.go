package antislop

import "testing"

func TestIsDecorativeSeparator(t *testing.T) {
	if !isDecorativeSeparator("──────") {
		t.Error("6 dashes should be decorative")
	}
	if !isDecorativeSeparator("======") {
		t.Error("6 equals should be decorative")
	}
	if isDecorativeSeparator("---") {
		t.Error("3 chars should not be enough")
	}
	if isDecorativeSeparator("hello") {
		t.Error("text should not be decorative")
	}
}

func TestIsDecorativeSectionHeader(t *testing.T) {
	if !isDecorativeSectionHeader("─── Title ───") {
		t.Error("should detect section header with title")
	}
	if !isDecorativeSectionHeader("=== Section ===") {
		t.Error("should detect equals section header")
	}
	if isDecorativeSectionHeader("──") {
		t.Error("too short should not match")
	}
	if isDecorativeSectionHeader("──────") {
		t.Error("no inner content should not match")
	}
}

func TestIsSectionHeader(t *testing.T) {
	if !isSectionHeader("Phase 1: Setup") {
		t.Error("should detect Phase 1:")
	}
	if !isSectionHeader("step 2. Build") {
		t.Error("should detect step 2.")
	}
	if !isSectionHeader("Section 3- Deploy") {
		t.Error("should detect Section 3-")
	}
	if isSectionHeader("Phases of the moon") {
		t.Error("should not match without digit")
	}
}

func TestContainsCrossRefPhrase(t *testing.T) {
	if !containsCrossRefPhrase("this will then be processed") {
		t.Error("should detect 'will then be'")
	}
	if !containsCrossRefPhrase("this is used by the handler") {
		t.Error("should detect 'used by'")
	}
	if !containsCrossRefPhrase("See above for details") {
		t.Error("should detect 'see above'")
	}
	if !containsCrossRefPhrase("we refactored from the old code") {
		t.Error("should detect 'refactored from'")
	}
	if containsCrossRefPhrase("this is normal code") {
		t.Error("should not match normal text")
	}
}

func TestHasWhyMarker(t *testing.T) {
	if !hasWhyMarker("because the API changed") {
		t.Error("should detect 'because'")
	}
	if !hasWhyMarker("note: this is intentional") {
		t.Error("should detect 'note:'")
	}
	if !hasWhyMarker("e.g. this example") {
		t.Error("should detect 'e.g.'")
	}
	if hasWhyMarker("just a regular comment") {
		t.Error("should not match regular text")
	}
}

func TestIsExplanatoryOpener(t *testing.T) {
	if !isExplanatoryOpener("Matches the user input against patterns") {
		t.Error("should detect 'Matches ...'")
	}
	if !isExplanatoryOpener("Detects broken links") {
		t.Error("should detect 'Detects ...'")
	}
	if !isExplanatoryOpener("Contains `config` values") {
		t.Error("should detect 'Contains `...'")
	}
	if isExplanatoryOpener("matches") {
		t.Error("no following char should not match")
	}
	if isExplanatoryOpener("Random text here") {
		t.Error("non-opener verb should not match")
	}
}

func TestHasJustificationOpener(t *testing.T) {
	if !hasJustificationOpener("The idea here is simple") {
		t.Error("should detect 'The idea here'")
	}
	if !hasJustificationOpener("This function handles requests") {
		t.Error("should detect 'This function ...'")
	}
	if !hasJustificationOpener("It handles all requests") {
		t.Error("should detect 'It handles ...'")
	}
	if !hasJustificationOpener("First it parses the input") {
		t.Error("should detect 'First it ...'")
	}
	if !hasJustificationOpener("Then we validate") {
		t.Error("should detect 'Then we ...'")
	}
	if hasJustificationOpener("Normal line of code") {
		t.Error("should not match normal text")
	}
}

func TestLooksLikeGoDoc(t *testing.T) {
	// func declaration with matching first word
	block := commentBlock{
		prose:            []string{"Load retrieves the entity."},
		nextNonBlankLine: "func Load() error {",
	}
	if !looksLikeGoDoc(block, ".go") {
		t.Error("should detect Go doc for func")
	}

	// Type declaration with matching first word
	block2 := commentBlock{
		prose:            []string{"Config holds settings."},
		nextNonBlankLine: "type Config struct {",
	}
	if !looksLikeGoDoc(block2, ".go") {
		t.Error("should detect Go doc for type")
	}

	// Method with receiver
	block3 := commentBlock{
		prose:            []string{"Save persists the data."},
		nextNonBlankLine: "func (s *Store) Save() error {",
	}
	if !looksLikeGoDoc(block3, ".go") {
		t.Error("should detect Go doc for method")
	}

	// Non-matching first word
	block4 := commentBlock{
		prose:            []string{"Retrieves the entity."},
		nextNonBlankLine: "func Load() error {",
	}
	if looksLikeGoDoc(block4, ".go") {
		t.Error("non-matching first word should not match")
	}

	// Struct field doc
	block5 := commentBlock{
		prose:            []string{"Name contains the display name."},
		nextNonBlankLine: "Name string",
	}
	if !looksLikeGoDoc(block5, ".go") {
		t.Error("should detect struct field doc")
	}

	// Not Go
	if looksLikeGoDoc(block, ".py") {
		t.Error("should not match for Python files")
	}

	// Empty prose
	block6 := commentBlock{
		prose:            []string{},
		nextNonBlankLine: "func Load() error {",
	}
	if looksLikeGoDoc(block6, ".go") {
		t.Error("empty prose should not match")
	}
}

func TestHasPreambleSlopSignal(t *testing.T) {
	// Explanatory opener
	b1 := commentBlock{prose: []string{"Detects broken patterns."}}
	if !hasPreambleSlopSignal(b1) {
		t.Error("should detect explanatory opener")
	}

	// Justification opener
	b2 := commentBlock{prose: []string{"This function handles requests."}}
	if !hasPreambleSlopSignal(b2) {
		t.Error("should detect justification opener")
	}

	// Cross-reference phrase
	b3 := commentBlock{prose: []string{"This", "will then be used elsewhere."}}
	if !hasPreambleSlopSignal(b3) {
		t.Error("should detect cross-reference phrase")
	}

	// Normal comment
	b4 := commentBlock{prose: []string{"just a normal comment"}}
	if hasPreambleSlopSignal(b4) {
		t.Error("should not match normal comment")
	}
}

func TestLooksLikeDeclarationPreamble(t *testing.T) {
	// Go
	if !looksLikeDeclarationPreamble("func main() {", ".go") {
		t.Error("should match func")
	}
	if !looksLikeDeclarationPreamble("type Config struct {", ".go") {
		t.Error("should match type")
	}
	if !looksLikeDeclarationPreamble("var x = 1", ".go") {
		t.Error("should match var")
	}
	if looksLikeDeclarationPreamble("x := 1", ".go") {
		t.Error("should not match assignment")
	}

	// Python
	if !looksLikeDeclarationPreamble("def foo():", ".py") {
		t.Error("should match def")
	}
	if !looksLikeDeclarationPreamble("class Foo:", ".py") {
		t.Error("should match class")
	}
	if !looksLikeDeclarationPreamble("async def bar():", ".py") {
		t.Error("should match async def")
	}
	if looksLikeDeclarationPreamble("x = 1", ".py") {
		t.Error("should not match assignment")
	}

	// Empty
	if looksLikeDeclarationPreamble("", ".go") {
		t.Error("empty should not match")
	}
}
