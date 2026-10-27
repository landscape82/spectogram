package main

import (
	"encoding/json"
	"image/color"
	"image/png"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/faiface/beep"
	"github.com/faiface/beep/wav"
)

func TestApplyHannWindow(t *testing.T) {
	samples := make([]float64, 8)
	for i := range samples {
		samples[i] = 1
	}
	got := applyHannWindow(samples)
	if math.Abs(got[0]) > 1e-9 || math.Abs(got[len(got)-1]) > 1e-9 {
		t.Fatal("Hann window must taper at both ends")
	}
	for i := 0; i < len(got)/2; i++ {
		if math.Abs(got[i]-got[len(got)-1-i]) > 1e-9 {
			t.Fatal("Hann window must be symmetric")
		}
	}
}

func TestApplyHannWindowSingleSampleIsFinite(t *testing.T) {
	got := applyHannWindow([]float64{1})
	if len(got) != 1 || math.IsNaN(got[0]) {
		t.Fatalf("single sample must remain finite: %v", got)
	}
}

func TestConfigValidation(t *testing.T) {
	for _, config := range []Config{
		{FFTSize: 1000, HopSize: 10, Channel: "mono", MinDB: -100, MaxDB: 0},
		{FFTSize: 64, HopSize: 65, Channel: "mono", MinDB: -100, MaxDB: 0},
		{FFTSize: 64, HopSize: 32, Channel: "banana", MinDB: -100, MaxDB: 0},
		{FFTSize: 64, HopSize: 32, Channel: "mono", MinDB: 0, MaxDB: 0},
	} {
		if err := config.validate(); err == nil {
			t.Fatalf("expected invalid config: %+v", config)
		}
	}
}

func TestStreamSpectrogramIncludesExactFinalWindowAndMetadata(t *testing.T) {
	config := Config{FFTSize: 64, HopSize: 32, Channel: "mono", MinDB: -100, MaxDB: 0}
	stream := stereoSineStreamer(128, 8, 8000, 1, 1)
	got, err := streamSpectrogram(stream, 8000, 2, config)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Data) != 3 {
		t.Fatalf("got %d frames, want 3 including exact final frame", len(got.Data))
	}
	if got.Metadata.SchemaVersion != 1 || got.Metadata.SampleRate != 8000 || got.Metadata.Channels != 2 {
		t.Fatalf("unexpected metadata: %+v", got.Metadata)
	}
	if got.Metadata.FrequencyStepHz != 125 || got.Metadata.TimeStepSeconds != 0.004 {
		t.Fatalf("unexpected physical axis metadata: %+v", got.Metadata)
	}
}

func TestStreamSpectrogramSinePeakAndDBFS(t *testing.T) {
	config := Config{FFTSize: 256, HopSize: 128, Channel: "left", MinDB: -100, MaxDB: 0}
	got, err := streamSpectrogram(stereoSineStreamer(512, 1000, 25600, 1, 0), 25600, 2, config)
	if err != nil {
		t.Fatal(err)
	}
	peak := 0
	for i := range got.Data[0] {
		if got.Data[0][i] > got.Data[0][peak] {
			peak = i
		}
	}
	if peak != 10 {
		t.Fatalf("peak bin = %d, want 10", peak)
	}
	if math.Abs(got.Data[0][peak]) > 0.1 {
		t.Fatalf("full-scale bin should be near 0 dBFS, got %f", got.Data[0][peak])
	}
}

func TestChannelSelection(t *testing.T) {
	sample := [2]float64{0.8, -0.2}
	if selectChannel(sample, "left") != 0.8 || selectChannel(sample, "right") != -0.2 {
		t.Fatal("left/right selection failed")
	}
	if math.Abs(selectChannel(sample, "mono")-0.3) > 1e-9 || math.Abs(selectChannel(sample, "side")-0.5) > 1e-9 {
		t.Fatal("mid/side selection failed")
	}
}

func TestFrameCollectorBoundsOutput(t *testing.T) {
	c := newFrameCollector(4)
	for i := 0; i < 40; i++ {
		c.add([]float64{float64(i)})
	}
	if len(c.frames) > 4 {
		t.Fatalf("stored %d frames, limit is 4", len(c.frames))
	}
	if c.stride <= 1 {
		t.Fatalf("expected downsampling, got stride %d", c.stride)
	}
}

func TestViridisColor(t *testing.T) {
	low := viridisColor(-100, -100, 0).(color.RGBA)
	high := viridisColor(0, -100, 0).(color.RGBA)
	if low != (color.RGBA{68, 1, 84, 255}) || high != (color.RGBA{253, 231, 37, 255}) {
		t.Fatalf("unexpected Viridis endpoints: %#v %#v", low, high)
	}
}

func TestWriteOutputsCreateParentDirs(t *testing.T) {
	dir := t.TempDir()
	s := Spectrogram{Metadata: Metadata{FFTSize: 4, MinDB: -100, MaxDB: 0}, Data: [][]float64{{-10, -5}, {-8, -1}}}
	jsonPath := filepath.Join(dir, "json", "spectrogram.json")
	pngPath := filepath.Join(dir, "images", "spectrogram.png")
	if err := writeJSON(jsonPath, s); err != nil {
		t.Fatal(err)
	}
	if err := writeImage(pngPath, s); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Spectrogram
	if err := json.Unmarshal(data, &decoded); err != nil || decoded.Metadata.FFTSize != 4 {
		t.Fatalf("invalid structured JSON: %v %+v", err, decoded)
	}
	f, err := os.Open(pngPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := png.Decode(f); err != nil {
		t.Fatalf("invalid PNG: %v", err)
	}
}

func TestServerListsAndServesGeneratedData(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "one.json"), []byte(`{"ok":true}`), 0644); err != nil {
		t.Fatal(err)
	}
	server := newServer(t.TempDir(), dir)
	listRequest := httptest.NewRequest(http.MethodGet, "/api/spectrograms", nil)
	listResponse := httptest.NewRecorder()
	server.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK || !strings.Contains(listResponse.Body.String(), "/data/one.json") {
		t.Fatalf("unexpected list response: %d %s", listResponse.Code, listResponse.Body.String())
	}
	dataRequest := httptest.NewRequest(http.MethodGet, "/data/one.json", nil)
	dataResponse := httptest.NewRecorder()
	server.ServeHTTP(dataResponse, dataRequest)
	if dataResponse.Code != http.StatusOK || dataResponse.Body.String() != `{"ok":true}` {
		t.Fatalf("unexpected data response: %d %s", dataResponse.Code, dataResponse.Body.String())
	}
}

func TestRunEndToEnd(t *testing.T) {
	dir := t.TempDir()
	wavPath := filepath.Join(dir, "INPUT.WAV")
	f, err := os.Create(wavPath)
	if err != nil {
		t.Fatal(err)
	}
	format := beep.Format{SampleRate: 44100, NumChannels: 2, Precision: 2}
	if err := wav.Encode(f, stereoSineStreamer(44100, 440, 44100, 1, 1), format); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := run(wavPath, filepath.Join(dir, "png", "out.png"), filepath.Join(dir, "data", "out.json")); err != nil {
		t.Fatal(err)
	}
}

func stereoSineStreamer(numSamples int, frequency, sampleRate, left, right float64) beep.Streamer {
	index := 0
	return beep.StreamerFunc(func(samples [][2]float64) (int, bool) {
		if index >= numSamples {
			return 0, false
		}
		n := 0
		for n < len(samples) && index < numSamples {
			v := math.Sin(2 * math.Pi * frequency * float64(index) / sampleRate)
			samples[n] = [2]float64{left * v, right * v}
			n++
			index++
		}
		return n, true
	})
}
