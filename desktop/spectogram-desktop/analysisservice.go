package main

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
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

// SavePNG exports a compact Viridis-like image of the analyzed result.
func (s *AnalysisService) SavePNG(path string, result spectrogram.Result) error {
	if len(result.Data) == 0 {
		return os.ErrInvalid
	}
	height := result.Metadata.FFTSize / 2
	img := image.NewRGBA(image.Rect(0, 0, len(result.Data), height))
	for x, row := range result.Data {
		for y, value := range row {
			t := (value - result.Metadata.MinDB) / (result.Metadata.MaxDB - result.Metadata.MinDB)
			if t < 0 {
				t = 0
			}
			if t > 1 {
				t = 1
			}
			img.Set(x, height-y-1, color.RGBA{uint8(68 + 185*t), uint8(1 + 230*t), uint8(84 - 47*t), 255})
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil && filepath.Dir(path) != "." {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
