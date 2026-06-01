package output

import (
	"fmt"
	"sort"
	"strings"

	"github.com/matej/antislop/engine"
	"github.com/matej/antislop/scoring"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorGreen  = "\033[32m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

func PrintResults(results []engine.EngineResult, score scoring.ScoreResult) {
	var allDiags []engine.Diagnostic
	for _, r := range results {
		if r.Skipped {
			fmt.Printf("%s⊘ %s skipped: %s%s\n", colorGray, r.Engine, r.SkipReason, colorReset)
			continue
		}
		allDiags = append(allDiags, r.Diagnostics...)
		fmt.Printf("  %-14s %s%d issues%s  (%s)\n",
			r.Engine,
			severityColor(len(r.Diagnostics)),
			len(r.Diagnostics),
			colorReset,
			r.Elapsed.Round(1e6),
		)
	}

	fmt.Println()

	if len(allDiags) == 0 {
		fmt.Printf("%s%s✓ No issues found%s\n\n", colorBold, colorGreen, colorReset)
		printScore(score)
		return
	}

	// Group by file
	byFile := map[string][]engine.Diagnostic{}
	for _, d := range allDiags {
		byFile[d.FilePath] = append(byFile[d.FilePath], d)
	}

	var files []string
	for f := range byFile {
		files = append(files, f)
	}
	sort.Strings(files)

	for _, file := range files {
		diags := byFile[file]
		sort.Slice(diags, func(i, j int) bool { return diags[i].Line < diags[j].Line })

		fmt.Printf("%s%s%s\n", colorBold, file, colorReset)
		for _, d := range diags {
			sev := severityIcon(d.Severity)
			fmt.Printf("  %s %s:%d %s%s%s\n", sev, colorGray, d.Line, colorReset, d.Message, "")
			if d.Help != "" {
				fmt.Printf("    %s%s%s\n", colorGray, d.Help, colorReset)
			}
		}
		fmt.Println()
	}

	printScore(score)
}

func printScore(score scoring.ScoreResult) {
	color := colorGreen
	switch score.Label {
	case "Needs Work":
		color = colorYellow
	case "Critical":
		color = colorRed
	}
	fmt.Printf("%s%sScore: %d/100 — %s%s\n\n", colorBold, color, score.Score, score.Label, colorReset)
}

func severityIcon(sev engine.Severity) string {
	switch sev {
	case engine.SeverityError:
		return colorRed + "✗" + colorReset
	case engine.SeverityWarning:
		return colorYellow + "⚠" + colorReset
	case engine.SeverityInfo:
		return colorCyan + "ℹ" + colorReset
	}
	return " "
}

func severityColor(count int) string {
	if count == 0 {
		return colorGreen
	}
	return colorYellow
}

func PrintSummary(results []engine.EngineResult) {
	var total int
	counts := map[engine.Severity]int{}
	for _, r := range results {
		for _, d := range r.Diagnostics {
			counts[d.Severity]++
			total++
		}
	}

	parts := []string{}
	if c := counts[engine.SeverityError]; c > 0 {
		parts = append(parts, fmt.Sprintf("%s%d errors%s", colorRed, c, colorReset))
	}
	if c := counts[engine.SeverityWarning]; c > 0 {
		parts = append(parts, fmt.Sprintf("%s%d warnings%s", colorYellow, c, colorReset))
	}
	if c := counts[engine.SeverityInfo]; c > 0 {
		parts = append(parts, fmt.Sprintf("%s%d info%s", colorCyan, c, colorReset))
	}

	if total == 0 {
		fmt.Printf("%sNo issues found.%s\n", colorGreen, colorReset)
	} else {
		fmt.Printf("Found %d issues: %s\n", total, strings.Join(parts, ", "))
	}
}
