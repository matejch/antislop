package security

import (
	"testing"
)

func TestRisky_DetectsPickleLoad(t *testing.T) {
	ctx := setupSecCtx(t, map[string]string{
		"app.py": "import pickle\ndata = pickle.loads(raw_bytes)\n",
	})
	diags := DetectRiskyConstructs(ctx)
	got := secDiagsWithRule(diags, "security/pickle-load")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for pickle.load, got %d", len(got))
	}
}

func TestRisky_DetectsPythonExec(t *testing.T) {
	ctx := setupSecCtx(t, map[string]string{
		"app.py": "exec('import os; os.system(\"ls\")')\n",
	})
	diags := DetectRiskyConstructs(ctx)
	got := secDiagsWithRule(diags, "security/python-exec")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for exec(), got %d", len(got))
	}
}

func TestRisky_DetectsGoSQLInjection(t *testing.T) {
	ctx := setupSecCtx(t, map[string]string{
		"db.go": `package db

func GetUser(id string) {
    db.Query(fmt.Sprintf("SELECT * FROM users WHERE id = '%s'", id))
}
`,
	})
	diags := DetectRiskyConstructs(ctx)
	got := secDiagsWithRule(diags, "security/sql-injection")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for SQL injection, got %d", len(got))
	}
}

func TestRisky_DetectsGoShellInjection(t *testing.T) {
	ctx := setupSecCtx(t, map[string]string{
		"run.go": `package run

func Execute(cmd string) {
    exec.Command(fmt.Sprintf("bash -c %s", cmd))
}
`,
	})
	diags := DetectRiskyConstructs(ctx)
	got := secDiagsWithRule(diags, "security/shell-injection")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for shell injection, got %d", len(got))
	}
}

func TestRisky_DetectsGoSQLInjectionWithConcat(t *testing.T) {
	ctx := setupSecCtx(t, map[string]string{
		"db.go": `package db

func GetUser(id string) {
    db.Query("SELECT * FROM users WHERE id = '" + id + "'")
}
`,
	})
	diags := DetectRiskyConstructs(ctx)
	got := secDiagsWithRule(diags, "security/sql-injection")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for SQL concat injection, got %d", len(got))
	}
}

func TestRisky_DetectsQueryRow(t *testing.T) {
	ctx := setupSecCtx(t, map[string]string{
		"db.go": `package db

func GetUser(id string) {
    db.QueryRow(fmt.Sprintf("SELECT * FROM users WHERE id = '%s'", id))
}
`,
	})
	diags := DetectRiskyConstructs(ctx)
	got := secDiagsWithRule(diags, "security/sql-injection")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for QueryRow SQL injection, got %d", len(got))
	}
}

func TestRisky_DetectsExec(t *testing.T) {
	ctx := setupSecCtx(t, map[string]string{
		"db.go": `package db

func Delete(id string) {
    db.Exec(fmt.Sprintf("DELETE FROM users WHERE id = '%s'", id))
}
`,
	})
	diags := DetectRiskyConstructs(ctx)
	got := secDiagsWithRule(diags, "security/sql-injection")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for Exec SQL injection, got %d", len(got))
	}
}

func TestRisky_DoesNotFlagExecWithoutWord(t *testing.T) {
	ctx := setupSecCtx(t, map[string]string{
		"app.py": "execute('safe command')\n",
	})
	diags := DetectRiskyConstructs(ctx)
	got := secDiagsWithRule(diags, "security/python-exec")
	if len(got) != 0 {
		t.Fatalf("expected 0 diagnostics for 'execute', got %d", len(got))
	}
}

func TestRisky_DetectsPyShellInjection(t *testing.T) {
	ctx := setupSecCtx(t, map[string]string{
		"run.py": "import subprocess\nsubprocess.run(f'echo ${user_input}')\n",
	})
	diags := DetectRiskyConstructs(ctx)
	got := secDiagsWithRule(diags, "security/shell-injection")
	if len(got) < 1 {
		t.Fatalf("expected >=1 diagnostic for Python shell injection, got %d", len(got))
	}
}

func TestRisky_AllDiagsAreSeverityError(t *testing.T) {
	ctx := setupSecCtx(t, map[string]string{
		"app.py": "import pickle\ndata = pickle.loads(raw_bytes)\n",
	})
	diags := DetectRiskyConstructs(ctx)
	for _, d := range diags {
		if d.Severity != "error" {
			t.Errorf("expected error severity, got %s for rule %s", d.Severity, d.Rule)
		}
	}
}
