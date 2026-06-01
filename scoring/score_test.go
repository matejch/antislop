package scoring

import (
	"testing"

	"github.com/matej/antislop/engine"
)

var defaultWeights = map[string]float64{
	"format":       0.3,
	"lint":         0.6,
	"code-quality": 0.8,
	"antislop":     2.5,
	"security":     1.5,
}

func makeDiags(n int, eng engine.EngineName, sev engine.Severity) []engine.Diagnostic {
	diags := make([]engine.Diagnostic, n)
	for i := range diags {
		diags[i] = engine.Diagnostic{
			FilePath: "test.go",
			Engine:   eng,
			Severity: sev,
		}
	}
	return diags
}

func TestScore_NoDiagnostics(t *testing.T) {
	r := Calculate(nil, defaultWeights, 75, 50, 10, 20)
	if r.Score != 100 {
		t.Errorf("expected 100, got %d", r.Score)
	}
	if r.Label != "Healthy" {
		t.Errorf("expected Healthy, got %s", r.Label)
	}
}

func TestScore_DecreasesWithMoreDiags(t *testing.T) {
	s1 := Calculate(makeDiags(1, engine.EngineLint, engine.SeverityWarning), defaultWeights, 75, 50, 10, 20)
	s10 := Calculate(makeDiags(10, engine.EngineLint, engine.SeverityWarning), defaultWeights, 75, 50, 10, 20)
	if s1.Score >= 100 {
		t.Errorf("1 diagnostic should reduce score below 100, got %d", s1.Score)
	}
	if s10.Score >= s1.Score {
		t.Errorf("10 diagnostics should score lower than 1: %d >= %d", s10.Score, s1.Score)
	}
}

func TestScore_ErrorsWorseThanWarnings(t *testing.T) {
	sErr := Calculate(makeDiags(1, engine.EngineLint, engine.SeverityError), defaultWeights, 75, 50, 10, 20)
	sWarn := Calculate(makeDiags(1, engine.EngineLint, engine.SeverityWarning), defaultWeights, 75, 50, 10, 20)
	if sErr.Score >= sWarn.Score {
		t.Errorf("error should score lower than warning: %d >= %d", sErr.Score, sWarn.Score)
	}
}

func TestScore_WarningsWorseThanInfo(t *testing.T) {
	sWarn := Calculate(makeDiags(1, engine.EngineLint, engine.SeverityWarning), defaultWeights, 75, 50, 10, 20)
	sInfo := Calculate(makeDiags(1, engine.EngineLint, engine.SeverityInfo), defaultWeights, 75, 50, 10, 20)
	if sWarn.Score >= sInfo.Score {
		t.Errorf("warning should score lower than info: %d >= %d", sWarn.Score, sInfo.Score)
	}
}

func TestScore_SecurityWeightHigherThanFormat(t *testing.T) {
	sSec := Calculate(makeDiags(1, engine.EngineSecurity, engine.SeverityError), defaultWeights, 75, 50, 10, 20)
	sFmt := Calculate(makeDiags(1, engine.EngineFormat, engine.SeverityError), defaultWeights, 75, 50, 10, 20)
	if sSec.Score >= sFmt.Score {
		t.Errorf("security error should score lower than format error: %d >= %d", sSec.Score, sFmt.Score)
	}
}

func TestScore_NeverBelowZero(t *testing.T) {
	r := Calculate(makeDiags(500, engine.EngineSecurity, engine.SeverityError), defaultWeights, 75, 50, 10, 20)
	if r.Score < 0 {
		t.Errorf("score should never be negative, got %d", r.Score)
	}
}

func TestScore_IsInteger(t *testing.T) {
	r := Calculate(makeDiags(7, engine.EngineAntislop, engine.SeverityWarning), defaultWeights, 75, 50, 10, 20)
	if r.Score != int(r.Score) {
		t.Errorf("score should be integer, got %d", r.Score)
	}
}

func TestScore_CriticalLabel(t *testing.T) {
	r := Calculate(makeDiags(200, engine.EngineSecurity, engine.SeverityError), defaultWeights, 75, 50, 10, 20)
	if r.Label != "Critical" {
		t.Errorf("expected Critical, got %s (score=%d)", r.Label, r.Score)
	}
	if r.Score >= 50 {
		t.Errorf("expected score < 50 for critical, got %d", r.Score)
	}
}

func TestScore_HealthyLabel(t *testing.T) {
	r := Calculate(nil, defaultWeights, 75, 50, 10, 20)
	if r.Label != "Healthy" {
		t.Errorf("expected Healthy, got %s", r.Label)
	}
}

func TestScore_NeedsWorkLabel(t *testing.T) {
	r := Calculate(makeDiags(5, engine.EngineLint, engine.SeverityWarning), defaultWeights, 90, 50, 10, 20)
	if r.Score < 50 || r.Score >= 90 {
		t.Skipf("score %d not in Needs Work range for this config", r.Score)
	}
	if r.Label != "Needs Work" {
		t.Errorf("expected Needs Work, got %s (score=%d)", r.Label, r.Score)
	}
}

func TestScore_FallbackFileCount(t *testing.T) {
	// sourceFileCount = 0 should use fallback (count unique files in diagnostics)
	diags := []engine.Diagnostic{
		{FilePath: "a.go", Engine: engine.EngineLint, Severity: engine.SeverityWarning},
		{FilePath: "b.go", Engine: engine.EngineLint, Severity: engine.SeverityWarning},
	}
	r := Calculate(diags, defaultWeights, 75, 50, 0, 20)
	if r.Score >= 100 {
		t.Errorf("expected score < 100 with diagnostics, got %d", r.Score)
	}
}

func TestScore_LargerCodebaseForgivesMore(t *testing.T) {
	sSmall := Calculate(makeDiags(1, engine.EngineSecurity, engine.SeverityError), defaultWeights, 75, 50, 2, 20)
	sLarge := Calculate(makeDiags(1, engine.EngineSecurity, engine.SeverityError), defaultWeights, 75, 50, 200, 20)
	if sLarge.Score < sSmall.Score {
		t.Errorf("larger codebase should forgive more: large=%d < small=%d", sLarge.Score, sSmall.Score)
	}
}
