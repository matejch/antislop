package security

import (
	"github.com/matej/antislop/engine"
)

type Security struct{}

func (s Security) Name() engine.EngineName { return engine.EngineSecurity }

func (s Security) Run(ctx engine.EngineContext) engine.EngineResult {
	var diags []engine.Diagnostic
	diags = append(diags, ScanSecrets(ctx)...)
	diags = append(diags, DetectRiskyConstructs(ctx)...)
	return engine.EngineResult{
		Engine:      engine.EngineSecurity,
		Diagnostics: diags,
	}
}
