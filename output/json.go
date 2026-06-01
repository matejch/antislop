package output

import (
	"encoding/json"
	"fmt"

	"github.com/matej/antislop/engine"
	"github.com/matej/antislop/scoring"
)

type JSONOutput struct {
	Score       int                 `json:"score"`
	Label       string              `json:"label"`
	Diagnostics []engine.Diagnostic `json:"diagnostics"`
	Engines     []EngineInfo        `json:"engines"`
}

type EngineInfo struct {
	Name       string `json:"name"`
	IssueCount int    `json:"issueCount"`
	ElapsedMs  int64  `json:"elapsedMs"`
	Skipped    bool   `json:"skipped"`
	SkipReason string `json:"skipReason,omitempty"`
}

func PrintJSON(results []engine.EngineResult, score scoring.ScoreResult) error {
	var allDiags []engine.Diagnostic
	var engines []EngineInfo

	for _, r := range results {
		allDiags = append(allDiags, r.Diagnostics...)
		engines = append(engines, EngineInfo{
			Name:       string(r.Engine),
			IssueCount: len(r.Diagnostics),
			ElapsedMs:  r.Elapsed.Milliseconds(),
			Skipped:    r.Skipped,
			SkipReason: r.SkipReason,
		})
	}

	if allDiags == nil {
		allDiags = []engine.Diagnostic{}
	}

	out := JSONOutput{
		Score:       score.Score,
		Label:       score.Label,
		Diagnostics: allDiags,
		Engines:     engines,
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
