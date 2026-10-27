package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"go-spectrogram-plotly/internal/spectrogram"
)

// AnalysisService is the Wails binding for the shared Go DSP engine.
type AnalysisService struct{}

// DefaultConfig supplies UI-friendly defaults without duplicating CLI values.
func (s *AnalysisService) DefaultConfig() spectrogram.Config {
	return spectrogram.DefaultConfig()
}

// Analyze reads a WAV or MP3 file and returns its metadata-rich spectrogram.
func (s *AnalysisService) Analyze(path string, config spectrogram.Config) (spectrogram.Result, error) {
	return spectrogram.AnalyzeFile(path, config)
}

// SaveJSON exports an analysis result without requiring the desktop UI to
// handle filesystem access directly.
func (s *AnalysisService) SaveJSON(path string, result spectrogram.Result) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil && filepath.Dir(path) != "." {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
