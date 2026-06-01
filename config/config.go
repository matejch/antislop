package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/matej/antislop/engine"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Version  int           `yaml:"version"`
	Engines  EnginesConfig `yaml:"engines"`
	Quality  QualityConfig `yaml:"quality"`
	Scoring  ScoringConfig `yaml:"scoring"`
	CI       CIConfig      `yaml:"ci"`
	Exclude  []string      `yaml:"exclude"`
	Include  []string      `yaml:"include"`
	Security SecurityYAML  `yaml:"security"`
}

type EnginesConfig struct {
	Format      bool `yaml:"format"`
	Lint        bool `yaml:"lint"`
	CodeQuality bool `yaml:"code-quality"`
	Antislop    bool `yaml:"antislop"`
	Security    bool `yaml:"security"`
}

type QualityConfig struct {
	MaxFunctionLoc int `yaml:"maxFunctionLoc"`
	MaxFileLoc     int `yaml:"maxFileLoc"`
	MaxNesting     int `yaml:"maxNesting"`
	MaxParams      int `yaml:"maxParams"`
}

type ScoringConfig struct {
	Weights    map[string]float64 `yaml:"weights"`
	Thresholds ThresholdsConfig   `yaml:"thresholds"`
	Smoothing  float64            `yaml:"smoothing"`
}

type ThresholdsConfig struct {
	Good int `yaml:"good"`
	OK   int `yaml:"ok"`
}

type CIConfig struct {
	FailBelow int `yaml:"failBelow"`
}

type SecurityYAML struct {
	Audit        bool `yaml:"audit"`
	AuditTimeout int  `yaml:"auditTimeout"`
}

func Default() Config {
	return Config{
		Version: 1,
		Engines: EnginesConfig{
			Format:      true,
			Lint:        true,
			CodeQuality: true,
			Antislop:    true,
			Security:    true,
		},
		Quality: QualityConfig{
			MaxFunctionLoc: 80,
			MaxFileLoc:     400,
			MaxNesting:     5,
			MaxParams:      6,
		},
		Scoring: ScoringConfig{
			Weights: map[string]float64{
				"format":       0.3,
				"lint":         0.6,
				"code-quality": 0.8,
				"antislop":     2.5,
				"security":     1.5,
			},
			Thresholds: ThresholdsConfig{
				Good: 75,
				OK:   50,
			},
			Smoothing: 20,
		},
		CI: CIConfig{
			FailBelow: 70,
		},
		Exclude: []string{
			"vendor", ".git", "node_modules", "dist", "build",
			"__pycache__", ".mypy_cache", ".pytest_cache",
		},
		Include: []string{},
		Security: SecurityYAML{
			Audit:        true,
			AuditTimeout: 25000,
		},
	}
}

func Load(dir string) (Config, error) {
	cfg := Default()

	candidates := []string{
		filepath.Join(dir, ".antislop", "config.yml"),
		filepath.Join(dir, ".antislop", "config.yaml"),
		filepath.Join(dir, ".antislop.yml"),
		filepath.Join(dir, ".antislop.yaml"),
	}

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return Default(), err
		}
		// Ensure default weights are present for any missing engines
		defaults := Default()
		for k, v := range defaults.Scoring.Weights {
			if _, ok := cfg.Scoring.Weights[k]; !ok {
				cfg.Scoring.Weights[k] = v
			}
		}
		return cfg, nil
	}

	return cfg, nil
}

func (c Config) EnabledEngines() map[engine.EngineName]bool {
	return map[engine.EngineName]bool{
		engine.EngineFormat:      c.Engines.Format,
		engine.EngineLint:        c.Engines.Lint,
		engine.EngineCodeQuality: c.Engines.CodeQuality,
		engine.EngineAntislop:    c.Engines.Antislop,
		engine.EngineSecurity:    c.Engines.Security,
	}
}

func (c Config) ToEngineConfig() engine.EngineConfig {
	return engine.EngineConfig{
		Quality: engine.QualityConfig{
			MaxFunctionLoc: c.Quality.MaxFunctionLoc,
			MaxFileLoc:     c.Quality.MaxFileLoc,
			MaxNesting:     c.Quality.MaxNesting,
			MaxParams:      c.Quality.MaxParams,
		},
		Security: engine.SecurityConfig{
			Audit:        c.Security.Audit,
			AuditTimeout: time.Duration(c.Security.AuditTimeout) * time.Millisecond,
		},
	}
}
