package spectrogram

import (
	"math"
	"testing"

	"github.com/faiface/beep"
)

func TestAnalyzeStreamMetadataAndPeak(t *testing.T) {
	config := Config{FFTSize: 256, HopSize: 128, Channel: "left", MinDB: -100, MaxDB: 0}
	result, err := AnalyzeStream(sineStreamer(512, 1000, 25600), 25600, 2, config)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Data) != 3 || result.Metadata.FrequencyStepHz != 100 {
		t.Fatalf("unexpected result metadata: %+v", result.Metadata)
	}
	peak := 0
	for i := range result.Data[0] {
		if result.Data[0][i] > result.Data[0][peak] {
			peak = i
		}
	}
	if peak != 10 || math.Abs(result.Data[0][peak]) > .1 {
		t.Fatalf("unexpected peak: bin=%d db=%f", peak, result.Data[0][peak])
	}
}

func TestConfigValidation(t *testing.T) {
	if err := (Config{FFTSize: 1000, HopSize: 10, Channel: "mono", MinDB: -100, MaxDB: 0}).Validate(); err == nil {
		t.Fatal("expected FFT validation error")
	}
}

func sineStreamer(count int, frequency, rate float64) beep.Streamer {
	i := 0
	return beep.StreamerFunc(func(samples [][2]float64) (int, bool) {
		if i >= count {
			return 0, false
		}
		n := 0
		for n < len(samples) && i < count {
			v := math.Sin(2 * math.Pi * frequency * float64(i) / rate)
			samples[n] = [2]float64{v, v}
			n++
			i++
		}
		return n, true
	})
}
