package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/matej/antislop/config"
	"github.com/matej/antislop/discover"
	"github.com/matej/antislop/engine"
	"github.com/matej/antislop/engine/antislop"
	"github.com/matej/antislop/engine/format"
	"github.com/matej/antislop/engine/lint"
	"github.com/matej/antislop/engine/quality"
	"github.com/matej/antislop/engine/security"
	"github.com/matej/antislop/output"
	"github.com/matej/antislop/scoring"
	"github.com/matej/antislop/util"
	"github.com/spf13/cobra"
)

var (
	jsonOutput  bool
	verbose     bool
	changesOnly bool
	stagedOnly  bool
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "antislop",
		Short: "Deterministic code quality scanner for Go and Python — catches AI slop patterns",
		Long: `antislop scans Go and Python codebases for AI-generated slop patterns,
complexity issues, security risks, and formatting/lint violations.

No LLM at runtime — same code always produces the same score.`,
	}

	root.AddCommand(newScanCmd())
	root.AddCommand(newCICmd())
	root.AddCommand(newInitCmd())
	root.AddCommand(newRulesCmd())
	return root
}

func newScanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan [dir]",
		Short: "Run a full code quality scan",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runScan,
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output results as JSON")
	cmd.Flags().BoolVarP(&verbose, "verbose", "d", false, "Verbose output")
	cmd.Flags().BoolVar(&changesOnly, "changes", false, "Only scan changed files (git diff)")
	cmd.Flags().BoolVar(&stagedOnly, "staged", false, "Only scan staged files")
	return cmd
}

func newCICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ci [dir]",
		Short: "CI-friendly scan with JSON output and exit codes",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonOutput = true
			return runScan(cmd, args)
		},
	}
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [dir]",
		Short: "Create a .antislop/config.yml file",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runInit,
	}
}

func newRulesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rules",
		Short: "List all available rules",
		RunE:  runRules,
	}
}

func runScan(cmd *cobra.Command, args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	absDir, err := resolveDir(dir)
	if err != nil {
		return err
	}

	cfg, err := config.Load(absDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: config error: %v (using defaults)\n", err)
		cfg = config.Default()
	}

	languages := discover.DetectLanguages(absDir)
	if len(languages) == 0 {
		fmt.Println("No Go or Python files found.")
		return nil
	}

	files, err := util.CollectSourceFiles(absDir, cfg.Include, cfg.Exclude, languages)
	if err != nil {
		return fmt.Errorf("collecting files: %w", err)
	}
	if len(files) == 0 {
		fmt.Println("No source files to scan.")
		return nil
	}

	ctx := buildContext(absDir, languages, files, cfg)
	results, score := executeScan(ctx, cfg)

	if jsonOutput {
		return output.PrintJSON(results, score)
	}

	output.PrintResults(results, score)
	if score.Score < cfg.CI.FailBelow {
		os.Exit(1)
	}
	return nil
}

func buildContext(absDir string, languages, files []string, cfg config.Config) engine.EngineContext {
	return engine.EngineContext{
		RootDir:        absDir,
		Languages:      languages,
		Files:          files,
		InstalledTools: detectTools(),
		Config:         cfg.ToEngineConfig(),
	}
}

func executeScan(ctx engine.EngineContext, cfg config.Config) ([]engine.EngineResult, scoring.ScoreResult) {
	engines := []engine.Engine{
		format.Format{},
		lint.Lint{},
		quality.CodeQuality{},
		antislop.AISlop{},
		security.Security{},
	}

	if !jsonOutput {
		fmt.Printf("\n  antislop — scanning %d files (%s)\n\n", len(ctx.Files), joinLangs(ctx.Languages))
	}

	results := engine.RunEngines(ctx, cfg.EnabledEngines(), engines)

	var allDiags []engine.Diagnostic
	for _, r := range results {
		allDiags = append(allDiags, r.Diagnostics...)
	}

	score := scoring.Calculate(
		allDiags,
		cfg.Scoring.Weights,
		cfg.Scoring.Thresholds.Good,
		cfg.Scoring.Thresholds.OK,
		len(ctx.Files),
		cfg.Scoring.Smoothing,
	)

	return results, score
}

func runInit(cmd *cobra.Command, args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	absDir, err := resolveDir(dir)
	if err != nil {
		return err
	}

	configDir := absDir + "/.antislop"
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}

	configPath := configDir + "/config.yml"
	if _, err := os.Stat(configPath); err == nil {
		fmt.Println("Config already exists:", configPath)
		return nil
	}

	content := `version: 1

engines:
  format: true
  lint: true
  code-quality: true
  antislop: true
  security: true

quality:
  maxFunctionLoc: 80
  maxFileLoc: 400
  maxNesting: 5
  maxParams: 6

scoring:
  weights:
    format: 0.3
    lint: 0.6
    code-quality: 0.8
    antislop: 2.5
    security: 1.5
  thresholds:
    good: 75
    ok: 50
  smoothing: 20

ci:
  failBelow: 70

exclude:
  - vendor
  - .git
  - node_modules
  - __pycache__
`

	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Println("Created", configPath)
	return nil
}

func runRules(cmd *cobra.Command, args []string) error {
	rules := []struct{ rule, engine, severity, description string }{
		{"antislop/trivial-comment", "antislop", "warning", "Trivial comment that restates the code"},
		{"antislop/narrative-comment", "antislop", "warning", "Narrative/decorative comment block"},
		{"antislop/swallowed-exception", "antislop", "error", "Empty catch/except block swallows errors"},
		{"antislop/thin-wrapper", "antislop", "warning", "Function that only delegates to another function"},
		{"antislop/generic-naming", "antislop", "info", "Generic AI-style variable/function naming"},
		{"antislop/debug-print", "antislop", "warning", "Debug print statement left in production code"},
		{"antislop/python-print-debug", "antislop", "warning", "print() in production Python code"},
		{"antislop/todo-stub", "antislop", "info", "Unresolved TODO/FIXME/HACK comment"},
		{"antislop/empty-function", "antislop", "info", "Empty function body"},
		{"antislop/go-library-panic", "antislop", "warning", "panic() in library (non-main) package"},
		{"antislop/python-bare-except", "antislop", "warning", "Bare except: swallows all exceptions"},
		{"antislop/python-broad-except", "antislop", "warning", "except Exception: pass silently drops errors"},
		{"antislop/python-mutable-default", "antislop", "warning", "Mutable default argument"},
		{"antislop/python-range-len-loop", "antislop", "info", "range(len()) loop pattern"},
		{"antislop/python-chained-dict-get", "antislop", "warning", "Chained .get({}).get() pattern"},
		{"antislop/python-repetitive-dispatch", "antislop", "warning", "Repetitive branch dispatch ladder"},
		{"antislop/python-isinstance-ladder", "antislop", "warning", "isinstance() branch ladder"},
		{"format/gofmt", "format", "warning", "File not formatted with gofmt"},
		{"format/ruff", "format", "warning", "File not formatted with ruff"},
		{"code-quality/file-too-long", "code-quality", "warning", "File exceeds line limit"},
		{"code-quality/function-too-long", "code-quality", "warning", "Function exceeds line limit"},
		{"code-quality/deep-nesting", "code-quality", "warning", "Excessive nesting depth"},
		{"security/hardcoded-secret", "security", "error", "Possible hardcoded secret in source"},
		{"security/pickle-load", "security", "error", "Unsafe pickle deserialization"},
		{"security/python-exec", "security", "error", "Use of exec()"},
		{"security/shell-injection", "security", "error", "Possible shell injection"},
		{"security/sql-injection", "security", "error", "Possible SQL injection"},
	}

	fmt.Printf("%-40s %-14s %-10s %s\n", "RULE", "ENGINE", "SEVERITY", "DESCRIPTION")
	fmt.Println(repeatStr("-", 100))
	for _, r := range rules {
		fmt.Printf("%-40s %-14s %-10s %s\n", r.rule, r.engine, r.severity, r.description)
	}
	return nil
}

func resolveDir(dir string) (string, error) {
	if dir == "." {
		d, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return d, nil
	}
	info, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("directory not found: %s", dir)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", dir)
	}
	return dir, nil
}

func detectTools() map[string]bool {
	tools := map[string]bool{}
	for _, name := range []string{"gofmt", "golangci-lint", "ruff"} {
		_, err := exec.LookPath(name)
		tools[name] = err == nil
	}
	return tools
}

func joinLangs(langs []string) string {
	if len(langs) == 0 {
		return "none"
	}
	result := ""
	for i, l := range langs {
		if i > 0 {
			result += ", "
		}
		result += l
	}
	return result
}

func repeatStr(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}
