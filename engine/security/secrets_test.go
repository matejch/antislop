package security

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matej/antislop/engine"
)

func setupSecCtx(t *testing.T, files map[string]string) engine.EngineContext {
	t.Helper()
	dir := t.TempDir()
	var paths []string
	for name, content := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	return engine.EngineContext{
		RootDir:   dir,
		Languages: []string{"go", "python"},
		Files:     paths,
	}
}

func secDiagsWithRule(diags []engine.Diagnostic, rule string) []engine.Diagnostic {
	var out []engine.Diagnostic
	for _, d := range diags {
		if d.Rule == rule {
			out = append(out, d)
		}
	}
	return out
}

func TestSecrets_DetectsAPIKey(t *testing.T) {
	ctx := setupSecCtx(t, map[string]string{
		"config.go": `package config
const apiKey = "abcdefghijklmnopqrstu12345"
`,
	})
	diags := ScanSecrets(ctx)
	if len(diags) == 0 {
		t.Fatal("expected >=1 diagnostic for API key")
	}
}

func TestSecrets_DetectsAWSAccessKey(t *testing.T) {
	ctx := setupSecCtx(t, map[string]string{
		"config.go": `package config
const accessKeyId = "AKIAIOSFODNN7EXAMPLE"
`,
	})
	diags := ScanSecrets(ctx)
	got := secDiagsWithRule(diags, "security/hardcoded-secret")
	if len(got) == 0 {
		t.Fatal("expected >=1 diagnostic for AWS access key")
	}
}

func TestSecrets_DetectsHardcodedPassword(t *testing.T) {
	ctx := setupSecCtx(t, map[string]string{
		"config.go": `package config
const password = "super-secret-password-123"
`,
	})
	diags := ScanSecrets(ctx)
	if len(diags) == 0 {
		t.Fatal("expected >=1 diagnostic for hardcoded password")
	}
}

func TestSecrets_DetectsPrivateKey(t *testing.T) {
	ctx := setupSecCtx(t, map[string]string{
		"key.go": "package config\nconst key = `-----BEGIN RSA PRIVATE KEY-----\nMIIEow...`\n",
	})
	diags := ScanSecrets(ctx)
	if len(diags) == 0 {
		t.Fatal("expected >=1 diagnostic for private key")
	}
}

func TestSecrets_DetectsJWT(t *testing.T) {
	ctx := setupSecCtx(t, map[string]string{
		"auth.go": `package auth
const jwt = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
`,
	})
	diags := ScanSecrets(ctx)
	if len(diags) == 0 {
		t.Fatal("expected >=1 diagnostic for JWT")
	}
}

func TestSecrets_DetectsGitHubToken(t *testing.T) {
	ctx := setupSecCtx(t, map[string]string{
		"ci.go": `package ci
const ghToken = "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijk"
`,
	})
	diags := ScanSecrets(ctx)
	if len(diags) == 0 {
		t.Fatal("expected >=1 diagnostic for GitHub token")
	}
}

func TestSecrets_DetectsDBConnectionString(t *testing.T) {
	ctx := setupSecCtx(t, map[string]string{
		"db.go": `package db
const dbUrl = "postgres://user:password123@localhost:5432/mydb"
`,
	})
	diags := ScanSecrets(ctx)
	if len(diags) == 0 {
		t.Fatal("expected >=1 diagnostic for DB connection string")
	}
}

func TestSecrets_DoesNotFlagPlaceholders(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"env_call", `package c; const k = "env(SECRET_KEY)"` + "\n"},
		{"os_getenv", `package c; const k = "os.Getenv(KEY)"` + "\n"},
		{"template_var", `package c; const k = "${SECRET_KEY}"` + "\n"},
		{"changeme", `package c; const password = "changeme"` + "\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := setupSecCtx(t, map[string]string{"config.go": tc.content})
			diags := ScanSecrets(ctx)
			if len(diags) != 0 {
				t.Fatalf("expected 0 diagnostics for placeholder %s, got %d", tc.name, len(diags))
			}
		})
	}
}

func TestSecrets_DetectsSlackToken(t *testing.T) {
	ctx := setupSecCtx(t, map[string]string{
		"config.go": `package config
const slackToken = "xoxb-fake-token-for-testing-only-aaaa"
`,
	})
	diags := ScanSecrets(ctx)
	if len(diags) == 0 {
		t.Fatal("expected >=1 diagnostic for Slack token")
	}
}

func TestSecrets_DoesNotFlagAngleBracketPlaceholder(t *testing.T) {
	ctx := setupSecCtx(t, map[string]string{
		"config.go": `package config
const apiKey = "<your-api-key-here>"
`,
	})
	diags := ScanSecrets(ctx)
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics for angle bracket placeholder, got %d", len(diags))
	}
}

func TestSecrets_MultipleSecretsInOneFile(t *testing.T) {
	ctx := setupSecCtx(t, map[string]string{
		"config.go": `package config
const apiKey = "abcdefghijklmnopqrstu12345"
const password = "super-secret-password-123"
`,
	})
	diags := ScanSecrets(ctx)
	if len(diags) < 2 {
		t.Fatalf("expected >=2 diagnostics for multiple secrets, got %d", len(diags))
	}
}
