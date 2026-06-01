package antislop

import (
	"sync"

	"github.com/matej/antislop/engine"
)

type AISlop struct{}

func (a AISlop) Name() engine.EngineName { return engine.EngineAntislop }

func (a AISlop) Run(ctx engine.EngineContext) engine.EngineResult {
	type detector func(engine.EngineContext) []engine.Diagnostic

	detectors := []detector{
		DetectTrivialComments,
		DetectNarrativeComments,
		DetectSwallowedExceptions,
		DetectOverAbstraction,
		DetectDeadPatterns,
		DetectGoPatterns,
		DetectPythonPatterns,
	}

	var (
		mu  sync.Mutex
		wg  sync.WaitGroup
		all []engine.Diagnostic
	)

	for _, d := range detectors {
		wg.Add(1)
		go func(fn detector) {
			defer wg.Done()
			diags := fn(ctx)
			mu.Lock()
			all = append(all, diags...)
			mu.Unlock()
		}(d)
	}
	wg.Wait()

	return engine.EngineResult{
		Engine:      engine.EngineAntislop,
		Diagnostics: all,
	}
}
