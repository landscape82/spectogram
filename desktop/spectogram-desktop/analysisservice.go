package main

import "go-spectrogram-plotly/internal/spectrogram"

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
