package scoring

import (
	"math"

	"github.com/matej/antislop/engine"
)

type ScoreResult struct {
	Score int    `json:"score"`
	Label string `json:"label"`
}

func Calculate(diagnostics []engine.Diagnostic, weights map[string]float64, good, ok int, sourceFileCount int, smoothing float64) ScoreResult {
	if len(diagnostics) == 0 {
		return ScoreResult{Score: 100, Label: "Healthy"}
	}

	var deductions float64
	for _, d := range diagnostics {
		w := weights[string(d.Engine)]
		if w == 0 {
			w = 1.0
		}
		var sevPenalty float64
		switch d.Severity {
		case engine.SeverityError:
			sevPenalty = 3
		case engine.SeverityWarning:
			sevPenalty = 1
		case engine.SeverityInfo:
			sevPenalty = 0.25
		}
		deductions += sevPenalty * w
	}

	effectiveFiles := sourceFileCount
	if effectiveFiles <= 0 {
		seen := map[string]struct{}{}
		for _, d := range diagnostics {
			seen[d.FilePath] = struct{}{}
		}
		effectiveFiles = len(seen)
		if effectiveFiles == 0 {
			effectiveFiles = 1
		}
	}

	density := math.Min(1, float64(len(diagnostics))/float64(float64(effectiveFiles)+smoothing))
	scaled := deductions * math.Sqrt(density)

	score := 100.0 - (100.0*math.Log1p(scaled))/math.Log1p(100.0+scaled)
	s := int(math.Round(math.Max(0, score)))

	label := "Critical"
	if s >= good {
		label = "Healthy"
	} else if s >= ok {
		label = "Needs Work"
	}

	return ScoreResult{Score: s, Label: label}
}
