package engine

import (
	"sync"
	"time"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

type EngineName string

const (
	EngineFormat      EngineName = "format"
	EngineLint        EngineName = "lint"
	EngineCodeQuality EngineName = "code-quality"
	EngineAntislop    EngineName = "antislop"
	EngineSecurity    EngineName = "security"
)

type Diagnostic struct {
	FilePath string     `json:"filePath"`
	Engine   EngineName `json:"engine"`
	Rule     string     `json:"rule"`
	Severity Severity   `json:"severity"`
	Message  string     `json:"message"`
	Help     string     `json:"help"`
	Line     int        `json:"line"`
	Column   int        `json:"column"`
	Category string     `json:"category"`
	Fixable  bool       `json:"fixable"`
	Detail   string     `json:"detail,omitempty"`
}

type EngineResult struct {
	Engine      EngineName    `json:"engine"`
	Diagnostics []Diagnostic  `json:"diagnostics"`
	Elapsed     time.Duration `json:"elapsed"`
	Skipped     bool          `json:"skipped"`
	SkipReason  string        `json:"skipReason,omitempty"`
}

type EngineContext struct {
	RootDir        string
	Languages      []string
	Files          []string
	InstalledTools map[string]bool
	Config         EngineConfig
}

type EngineConfig struct {
	Quality  QualityConfig
	Security SecurityConfig
}

type QualityConfig struct {
	MaxFunctionLoc int
	MaxFileLoc     int
	MaxNesting     int
	MaxParams      int
}

type SecurityConfig struct {
	Audit        bool
	AuditTimeout time.Duration
}

type Engine interface {
	Name() EngineName
	Run(ctx EngineContext) EngineResult
}

func RunEngines(ctx EngineContext, enabled map[EngineName]bool, engines []Engine) []EngineResult {
	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results = make([]EngineResult, 0, len(engines))
	)

	for _, e := range engines {
		if !enabled[e.Name()] {
			continue
		}
		wg.Add(1)
		go func(eng Engine) {
			defer wg.Done()
			start := time.Now()
			result := eng.Run(ctx)
			result.Elapsed = time.Since(start)
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(e)
	}

	wg.Wait()
	return results
}
