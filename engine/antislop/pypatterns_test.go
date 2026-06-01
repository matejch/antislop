package antislop

import (
	"testing"

	"github.com/matej/antislop/engine"
)

func TestPython_FlagsBareExcept(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.py": "try:\n    do_something()\nexcept:\n    pass\n",
	})
	diags := DetectPythonPatterns(ctx)
	got := diagsWithRule(diags, "antislop/python-bare-except")
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(got))
	}
	if got[0].Line != 3 {
		t.Errorf("expected line 3, got %d", got[0].Line)
	}
}

func TestPython_DoesNotFlagSpecificExcept(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.py": "try:\n    do_something()\nexcept ValueError:\n    handle()\nexcept (KeyError, IndexError) as e:\n    log(e)\n",
	})
	diags := DetectPythonPatterns(ctx)
	got := diagsWithRule(diags, "antislop/python-bare-except")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics, got %d", len(got))
	}
}

func TestPython_FlagsBroadExceptWithPass(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.py": "try:\n    do_thing()\nexcept Exception:\n    pass\n",
	})
	diags := DetectPythonPatterns(ctx)
	got := diagsWithRule(diags, "antislop/python-broad-except")
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(got))
	}
}

func TestPython_DoesNotFlagBroadExceptWithRealHandler(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.py": "import logging\nlog = logging.getLogger(__name__)\ntry:\n    do_thing()\nexcept Exception as e:\n    log.error('failed', exc_info=e)\n    raise\n",
	})
	diags := DetectPythonPatterns(ctx)
	got := diagsWithRule(diags, "antislop/python-broad-except")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics, got %d", len(got))
	}
}

func TestPython_FlagsMutableDefaultList(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.py": "def append_to(items=[]):\n    items.append(1)\n    return items\n",
	})
	diags := DetectPythonPatterns(ctx)
	got := diagsWithRule(diags, "antislop/python-mutable-default")
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(got))
	}
	if got[0].Line != 1 {
		t.Errorf("expected line 1, got %d", got[0].Line)
	}
}

func TestPython_FlagsMutableDefaultDictAndSet(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.py": "def use_dict(opts={}):\n    return opts\ndef use_set(s=set()):\n    return s\n",
	})
	diags := DetectPythonPatterns(ctx)
	got := diagsWithRule(diags, "antislop/python-mutable-default")
	if len(got) != 2 {
		t.Fatalf("expected 2 diagnostics, got %d", len(got))
	}
}

func TestPython_DoesNotFlagImmutableDefaults(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.py": "def f(x=None, y=0, z='', flag=True):\n    return (x, y, z, flag)\n",
	})
	diags := DetectPythonPatterns(ctx)
	got := diagsWithRule(diags, "antislop/python-mutable-default")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics, got %d", len(got))
	}
}

func TestPython_FlagsMultiLineSignatureMutableDefault(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.py": "def long_signature(\n    a: int,\n    b: str = '',\n    c: list = [],\n    d: bool = True,\n):\n    return (a, b, c, d)\n",
	})
	diags := DetectPythonPatterns(ctx)
	got := diagsWithRule(diags, "antislop/python-mutable-default")
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(got))
	}
}

func TestPython_FlagsRangeLenLoop(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.py": "def names(users):\n    out = []\n    for i in range(len(users)):\n        out.append(users[i].name)\n    return out\n",
	})
	diags := DetectPythonPatterns(ctx)
	got := diagsWithRule(diags, "antislop/python-range-len-loop")
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(got))
	}
	if got[0].Line != 3 {
		t.Errorf("expected line 3, got %d", got[0].Line)
	}
	if got[0].Severity != engine.SeverityInfo {
		t.Errorf("expected info severity, got %s", got[0].Severity)
	}
}

func TestPython_DoesNotFlagDirectIteration(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.py": "def names(users):\n    for user in users:\n        yield user.name\n\ndef indexed(users):\n    for i, user in enumerate(users):\n        yield i, user.name\n",
	})
	diags := DetectPythonPatterns(ctx)
	got := diagsWithRule(diags, "antislop/python-range-len-loop")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics, got %d", len(got))
	}
}

func TestPython_FlagsChainedDictGet(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.py": "def timeout(config):\n    return config.get('service', {}).get('http', {}).get('timeout', 30)\n",
	})
	diags := DetectPythonPatterns(ctx)
	got := diagsWithRule(diags, "antislop/python-chained-dict-get")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic, got %d", len(got))
	}
}

func TestPython_FlagsRepetitiveDispatch(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.py": `def normalize(selector, node):
    if selector == 'string_literal':
        return node.kind in ('string', 'raw_string')
    elif selector == 'numeric_literal':
        return node.kind in ('number', 'integer', 'float')
    elif selector == 'boolean_literal':
        return node.kind in ('true', 'false')
    elif selector == 'null_literal':
        return node.kind in ('null', 'none')
    return False
`,
	})
	diags := DetectPythonPatterns(ctx)
	got := diagsWithRule(diags, "antislop/python-repetitive-dispatch")
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(got))
	}
}

func TestPython_FlagsIsinstanceLadder(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.py": `def encode(value):
    if isinstance(value, str):
        return value
    elif isinstance(value, int):
        return str(value)
    elif isinstance(value, float):
        return str(value)
    elif isinstance(value, bool):
        return 'true' if value else 'false'
    return ''
`,
	})
	diags := DetectPythonPatterns(ctx)
	got := diagsWithRule(diags, "antislop/python-isinstance-ladder")
	if len(got) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(got))
	}
}

func TestPython_DoesNotFlagShortBranchLadder(t *testing.T) {
	ctx := setupCtx(t, map[string]string{
		"app.py": `def encode(value):
    if isinstance(value, str):
        return value
    elif isinstance(value, int):
        return str(value)
    return ''
`,
	})
	diags := DetectPythonPatterns(ctx)
	got1 := diagsWithRule(diags, "antislop/python-isinstance-ladder")
	got2 := diagsWithRule(diags, "antislop/python-repetitive-dispatch")
	if len(got1) != 0 || len(got2) != 0 {
		t.Fatalf("expected 0 diagnostics for short ladder, got isinstance=%d dispatch=%d", len(got1), len(got2))
	}
}
