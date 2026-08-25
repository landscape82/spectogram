package main

import (
	"encoding/json"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/faiface/beep"
	"github.com/faiface/beep/wav"
)

func TestApplyHannWindow(t *testing.T) {
	samples := make([]float64, 8)
	for i := range samples {
		samples[i] = 1
	}

	windowed := applyHannWindow(samples)

	if len(windowed) != len(samples) {
		t.Fatalf("expected length %d, got %d", len(samples), len(windowed))
	}

	// Hann window tapers to (near) zero at both edges and peaks at the center.
	if math.Abs(windowed[0]) > 1e-9 {
		t.Errorf("expected first sample near 0, got %v", windowed[0])
	}
	if math.Abs(windowed[len(windowed)-1]) > 1e-9 {
		t.Errorf("expected last sample near 0, got %v", windowed[len(windowed)-1])
	}

	mid := len(windowed) / 2
	if windowed[mid] <= windowed[0] || windowed[mid] <= windowed[1] {
		t.Errorf("expected center sample to be larger than edge samples, got center=%v", windowed[mid])
	}

	// Window should be symmetric.
	for i := 0; i < len(windowed)/2; i++ {
		j := len(windowed) - 1 - i
		if math.Abs(windowed[i]-windowed[j]) > 1e-9 {
			t.Errorf("expected symmetric window, windowed[%d]=%v != windowed[%d]=%v", i, windowed[i], j, windowed[j])
		}
	}
}

func TestApplyHannWindowSingleSample(t *testing.T) {
	// Documents current behavior: a single-sample window divides by zero
	// (len-1 == 0), producing NaN rather than panicking.
	samples := []float64{1}
	windowed := applyHannWindow(samples)

	if len(windowed) != 1 {
		t.Fatalf("expected length 1, got %d", len(windowed))
	}
	if !math.IsNaN(windowed[0]) {
		t.Errorf("expected NaN for single-sample window, got %v", windowed[0])
	}
}

func TestViridisColor(t *testing.T) {
	tests := []struct {
		name         string
		value, max   float64
		wantR, wantB uint8
	}{
		{"at minimum maps to norm 0", -100, 0, 0, 255},
		{"at maximum maps to norm 1", 0, 0, 255, 0},
		{"below minimum clamps to norm 0", -1000, 0, 0, 255},
		{"above maximum clamps to norm 1", 1000, 0, 255, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := viridisColor(tt.value, tt.max)
			r, _, b, a := c.RGBA()
			gotR := uint8(r >> 8)
			gotB := uint8(b >> 8)
			if gotR != tt.wantR {
				t.Errorf("R = %d, want %d", gotR, tt.wantR)
			}
			if gotB != tt.wantB {
				t.Errorf("B = %d, want %d", gotB, tt.wantB)
			}
			if a>>8 != 255 {
				t.Errorf("expected fully opaque color, got alpha %d", a>>8)
			}
		})
	}
}

func TestComputeSpectrogramDimensions(t *testing.T) {
	const (
		windowSize = 64
		step       = 32
		numSamples = 256
	)

	samples := make([]float64, numSamples)
	for i := range samples {
		samples[i] = math.Sin(2 * math.Pi * float64(i) / 8)
	}

	spectrogram, maxVal := computeSpectrogram(samples, windowSize, step)

	wantWindows := 0
	for i := 0; i+windowSize < numSamples; i += step {
		wantWindows++
	}

	if len(spectrogram) != wantWindows {
		t.Fatalf("expected %d windows, got %d", wantWindows, len(spectrogram))
	}

	for i, row := range spectrogram {
		if len(row) != windowSize/2 {
			t.Fatalf("row %d: expected %d magnitude bins, got %d", i, windowSize/2, len(row))
		}
	}

	if maxVal == -math.MaxFloat64 {
		t.Errorf("expected maxVal to be updated from initial sentinel")
	}
}

func TestComputeSpectrogramTooShortInput(t *testing.T) {
	samples := make([]float64, 10)
	spectrogram, _ := computeSpectrogram(samples, 1024, 512)

	if len(spectrogram) != 0 {
		t.Fatalf("expected no windows for input shorter than windowSize, got %d", len(spectrogram))
	}
}

func TestComputeSpectrogramEmptyInput(t *testing.T) {
	spectrogram, maxVal := computeSpectrogram(nil, 64, 32)

	if len(spectrogram) != 0 {
		t.Fatalf("expected no windows for empty input, got %d", len(spectrogram))
	}
	if maxVal != -math.MaxFloat64 {
		t.Errorf("expected maxVal to remain at sentinel for empty input, got %v", maxVal)
	}
}

func TestComputeSpectrogramSineWavePeakBin(t *testing.T) {
	const (
		windowSize = 256
		step       = 128
		cycles     = 10 // exact number of sine cycles within one window
	)

	samples := make([]float64, windowSize*3)
	for i := range samples {
		samples[i] = math.Sin(2 * math.Pi * float64(cycles) * float64(i) / float64(windowSize))
	}

	spectrogram, _ := computeSpectrogram(samples, windowSize, step)
	if len(spectrogram) == 0 {
		t.Fatal("expected at least one window")
	}

	row := spectrogram[0]
	peakBin := 0
	for j := 1; j < len(row); j++ {
		if row[j] > row[peakBin] {
			peakBin = j
		}
	}

	if peakBin != cycles {
		t.Errorf("expected FFT peak at bin %d, got bin %d", cycles, peakBin)
	}
}

func TestComputeSpectrogramDoesNotDoubleWindowOverlap(t *testing.T) {
	// Regression test: overlapping windows (step < windowSize) must not
	// mutate the shared underlying samples slice, or the overlap region
	// would be Hann-windowed twice.
	const windowSize = 64
	const step = 32

	samples := make([]float64, windowSize+step)
	for i := range samples {
		samples[i] = 1
	}
	original := append([]float64(nil), samples...)

	computeSpectrogram(samples, windowSize, step)

	for i, v := range samples {
		if v != original[i] {
			t.Fatalf("computeSpectrogram mutated input samples at index %d: got %v, want %v", i, v, original[i])
		}
	}
}

func TestWriteJSONCreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "spectrogram.json")

	spectrogram := [][]float64{{1, 2}, {3, 4}}
	if err := writeJSON(path, spectrogram); err != nil {
		t.Fatalf("writeJSON returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read written JSON: %v", err)
	}

	var got [][]float64
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to unmarshal written JSON: %v", err)
	}

	if len(got) != len(spectrogram) {
		t.Fatalf("expected %d rows, got %d", len(spectrogram), len(got))
	}
}

func TestWriteImageProducesValidPNG(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spectrogram.png")

	spectrogram := [][]float64{{-10, -5}, {-8, -1}}
	if err := writeImage(path, spectrogram, 4, -1); err != nil {
		t.Fatalf("writeImage returned error: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open written PNG: %v", err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("failed to decode written PNG: %v", err)
	}

	wantWidth := len(spectrogram) + 60
	wantHeight := 4/2 + 40
	if img.Bounds().Dx() != wantWidth || img.Bounds().Dy() != wantHeight {
		t.Errorf("unexpected image size: got %dx%d, want %dx%d", img.Bounds().Dx(), img.Bounds().Dy(), wantWidth, wantHeight)
	}
}

// sineStreamer is a minimal beep.Streamer producing a mono sine wave,
// used to synthesize a WAV file for end-to-end testing of run().
func sineStreamer(numSamples int, freq, sampleRate float64) beep.Streamer {
	i := 0
	return beep.StreamerFunc(func(samples [][2]float64) (n int, ok bool) {
		if i >= numSamples {
			return 0, false
		}
		for n = 0; n < len(samples) && i < numSamples; n, i = n+1, i+1 {
			v := math.Sin(2 * math.Pi * freq * float64(i) / sampleRate)
			samples[n][0] = v
			samples[n][1] = v
		}
		return n, true
	})
}

func TestRunEndToEnd(t *testing.T) {
	dir := t.TempDir()
	wavPath := filepath.Join(dir, "input.wav")
	pngPath := filepath.Join(dir, "out.png")
	jsonPath := filepath.Join(dir, "nested", "out.json")

	format := beep.Format{SampleRate: 44100, NumChannels: 2, Precision: 2}

	wf, err := os.Create(wavPath)
	if err != nil {
		t.Fatalf("failed to create temp wav file: %v", err)
	}
	if err := wav.Encode(wf, sineStreamer(44100, 440, 44100), format); err != nil {
		wf.Close()
		t.Fatalf("failed to encode test wav file: %v", err)
	}
	if err := wf.Close(); err != nil {
		t.Fatalf("failed to close test wav file: %v", err)
	}

	if err := run(wavPath, pngPath, jsonPath); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	if _, err := os.Stat(pngPath); err != nil {
		t.Errorf("expected PNG output to exist: %v", err)
	}
	if _, err := os.Stat(jsonPath); err != nil {
		t.Errorf("expected JSON output (with created parent dir) to exist: %v", err)
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("failed to read JSON output: %v", err)
	}
	var spectrogram [][]float64
	if err := json.Unmarshal(data, &spectrogram); err != nil {
		t.Fatalf("failed to unmarshal JSON output: %v", err)
	}
	if len(spectrogram) == 0 {
		t.Error("expected non-empty spectrogram from 1 second of audio")
	}
}

func TestRunUnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "input.txt")
	if err := os.WriteFile(path, []byte("not audio"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	err := run(path, filepath.Join(dir, "out.png"), filepath.Join(dir, "out.json"))
	if err == nil {
		t.Fatal("expected error for unsupported file format, got nil")
	}
}

func TestRunMissingInputFile(t *testing.T) {
	dir := t.TempDir()
	err := run(filepath.Join(dir, "does-not-exist.wav"), filepath.Join(dir, "out.png"), filepath.Join(dir, "out.json"))
	if err == nil {
		t.Fatal("expected error for missing input file, got nil")
	}
}
