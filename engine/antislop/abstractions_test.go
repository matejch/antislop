package antislop

import (
	"testing"
)

func TestAbstractions_FlagsGoThinWrapper(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": `package app

func GetData(id string) string {
    return fetchData(id)
}
`,
	})
	diags := DetectOverAbstraction(ctx)
	got := diagsWithRule(diags, "antislop/thin-wrapper")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for thin wrapper, got %d", len(got))
	}
}

func TestAbstractions_FlagsPythonThinWrapper(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.py": "def get_data(id):\n    return fetch_data(id)\n",
	})
	diags := DetectOverAbstraction(ctx)
	got := diagsWithRule(diags, "antislop/thin-wrapper")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for Python thin wrapper, got %d", len(got))
	}
}

func TestAbstractions_DoesNotFlagDecoratedFunction(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.py": "@decorator\ndef get_data(id):\n    return fetch_data(id)\n",
	})
	diags := DetectOverAbstraction(ctx)
	got := diagsWithRule(diags, "antislop/thin-wrapper")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for decorated function, got %d", len(got))
	}
}

func TestAbstractions_DoesNotFlagDunderMethod(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.py": "def __init__(self):\n    return super().__init__()\n",
	})
	diags := DetectOverAbstraction(ctx)
	got := diagsWithRule(diags, "antislop/thin-wrapper")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for dunder method, got %d", len(got))
	}
}

func TestAbstractions_FlagsGenericNaming(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"helper1", "package app\n\nfunc helper1(x int) int { return x * 2 }\n"},
		{"data2", "package app\n\nvar data2 = getData()\n"},
		{"temp1", "package app\n\nvar temp1 = compute()\n"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := setupCtx(t, map[string]string{"app.go": tc.content})
			diags := DetectOverAbstraction(ctx)
			got := diagsWithRule(diags, "antislop/generic-naming")
			if len(got) < 1 {
				t.Fatalf("expected >=1 diagnostic for %s, got %d", tc.name, len(got))
			}
		})
	}
}

func TestAbstractions_FlagsGoMethodThinWrapper(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": `package app

func (s *Store) GetData(id string) string {
    return s.fetch(id)
}
`,
	})
	diags := DetectOverAbstraction(ctx)
	got := diagsWithRule(diags, "antislop/thin-wrapper")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for method thin wrapper, got %d", len(got))
	}
}

func TestAbstractions_DoesNotFlagWrapperWithHardcodedArgs(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": `package app

func GetDefault() string {
    return fetch("default")
}
`,
	})
	diags := DetectOverAbstraction(ctx)
	got := diagsWithRule(diags, "antislop/thin-wrapper")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for hardcoded args, got %d", len(got))
	}
}

func TestAbstractions_DoesNotFlagPyWrapperWithBody(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.py": "def process(x):\n    y = transform(x)\n    return save(y)\n",
	})
	diags := DetectOverAbstraction(ctx)
	got := diagsWithRule(diags, "antislop/thin-wrapper")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for multi-line body, got %d", len(got))
	}
}

func TestAbstractions_FlagsGenericNamingPrefixUnderscore(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": "package app\n\nfunc handler_1() {}\n",
	})
	diags := DetectOverAbstraction(ctx)
	got := diagsWithRule(diags, "antislop/generic-naming")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for handler_1, got %d", len(got))
	}
}

func TestAbstractions_DoesNotFlagWrapperWithNumericArg(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": `package app

func GetTimeout() int {
    return getConfig(30)
}
`,
	})
	diags := DetectOverAbstraction(ctx)
	got := diagsWithRule(diags, "antislop/thin-wrapper")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for numeric hardcoded arg, got %d", len(got))
	}
}

func TestAbstractions_IsDunder(t *testing.T) {
	tests := []struct {
		name   string
		expect bool
	}{
		{"__init__", true},
		{"__str__", true},
		{"__x__", true},
		{"__", false},
		{"___", false},
		{"____", false},
		{"init", false},
		{"__init", false},
		{"init__", false},
	}
	for _, tc := range tests {
		if isDunder(tc.name) != tc.expect {
			t.Errorf("isDunder(%q) = %v, want %v", tc.name, !tc.expect, tc.expect)
		}
	}
}

func TestAbstractions_DoesNotFlagNormalNames(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.go": `package app

func processOrders(orders []Order) error { return nil }
var config = loadConfig()
`,
	})
	diags := DetectOverAbstraction(ctx)
	got := diagsWithRule(diags, "antislop/generic-naming")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for normal names, got %d", len(got))
	}
}
