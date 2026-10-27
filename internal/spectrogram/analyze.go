// Package spectrogram provides reusable audio analysis for the CLI and desktop app.
package spectrogram

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/wav"
	"gonum.org/v1/gonum/dsp/fourier"
)

const (
	DefaultFFTSize = 1024
	DefaultHopSize = 512
)

type Config struct {
	FFTSize   int
	HopSize   int
	Channel   string
	MinDB     float64
	MaxDB     float64
	MaxFrames int
}

func DefaultConfig() Config {
	return Config{FFTSize: DefaultFFTSize, HopSize: DefaultHopSize, Channel: "mono", MinDB: -100, MaxDB: 0}
}

func (c Config) Validate() error {
	if c.FFTSize < 2 || c.FFTSize&(c.FFTSize-1) != 0 {
		return errors.New("fft size must be a power of two and at least 2")
	}
	if c.HopSize < 1 || c.HopSize > c.FFTSize {
		return errors.New("hop size must be between 1 and fft size")
	}
	if c.MinDB >= c.MaxDB {
		return errors.New("min-db must be smaller than max-db")
	}
	if c.MaxFrames < 0 {
		return errors.New("max-frames cannot be negative")
	}
	switch c.Channel {
	case "mono", "left", "right", "mid", "side":
		return nil
	}
	return fmt.Errorf("unsupported channel %q (use mono, left, right, mid, or side)", c.Channel)
}

type Metadata struct {
	SchemaVersion   int     `json:"schemaVersion"`
	SampleRate      int     `json:"sampleRate"`
	Channels        int     `json:"channels"`
	Channel         string  `json:"channel"`
	FFTSize         int     `json:"fftSize"`
	HopSize         int     `json:"hopSize"`
	FrameStride     int     `json:"frameStride"`
	FrequencyStepHz float64 `json:"frequencyStepHz"`
	TimeStepSeconds float64 `json:"timeStepSeconds"`
	MinDB           float64 `json:"minDb"`
	MaxDB           float64 `json:"maxDb"`
	DurationSeconds float64 `json:"durationSeconds"`
}

type Result struct {
	Metadata Metadata    `json:"metadata"`
	Data     [][]float64 `json:"data"`
}

func AnalyzeFile(path string, config Config) (Result, error) {
	if err := config.Validate(); err != nil {
		return Result{}, err
	}
	f, err := os.Open(path)
	if err != nil {
		return Result{}, err
	}
	defer f.Close()
	var stream beep.StreamSeekCloser
	var format beep.Format
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp3":
		stream, format, err = mp3.Decode(f)
	case ".wav":
		stream, format, err = wav.Decode(f)
	default:
		return Result{}, fmt.Errorf("unsupported file format: %s", filepath.Ext(path))
	}
	if err != nil {
		return Result{}, err
	}
	defer stream.Close()
	return AnalyzeStream(stream, int(format.SampleRate), format.NumChannels, config)
}

func AnalyzeStream(stream beep.Streamer, sampleRate, channels int, config Config) (Result, error) {
	if err := config.Validate(); err != nil {
		return Result{}, err
	}
	if sampleRate <= 0 {
		return Result{}, errors.New("audio has an invalid sample rate")
	}
	fft := fourier.NewFFT(config.FFTSize)
	collector := &collector{limit: config.MaxFrames, stride: 1}
	window := make([]float64, 0, config.FFTSize)
	buffer := make([][2]float64, 4096)
	count := 0
	for {
		n, ok := stream.Stream(buffer)
		for i := 0; i < n; i++ {
			window = append(window, channel(buffer[i], config.Channel))
			count++
			if len(window) == config.FFTSize {
				collector.add(frame(window, fft, config))
				copy(window, window[config.HopSize:])
				window = window[:config.FFTSize-config.HopSize]
			}
		}
		if !ok {
			break
		}
	}
	return Result{Metadata: Metadata{SchemaVersion: 1, SampleRate: sampleRate, Channels: channels, Channel: config.Channel, FFTSize: config.FFTSize, HopSize: config.HopSize, FrameStride: collector.stride, FrequencyStepHz: float64(sampleRate) / float64(config.FFTSize), TimeStepSeconds: float64(config.HopSize*collector.stride) / float64(sampleRate), MinDB: config.MinDB, MaxDB: config.MaxDB, DurationSeconds: float64(count) / float64(sampleRate)}, Data: collector.frames}, nil
}

func channel(s [2]float64, mode string) float64 {
	switch mode {
	case "left":
		return s[0]
	case "right":
		return s[1]
	case "side":
		return (s[0] - s[1]) / 2
	}
	return (s[0] + s[1]) / 2
}
func frame(samples []float64, fft *fourier.FFT, c Config) []float64 {
	w := append([]float64(nil), samples...)
	for i := range w {
		w[i] *= .5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(len(w)-1)))
	}
	s := fft.Coefficients(nil, w)
	out := make([]float64, c.FFTSize/2)
	for i := range out {
		db := 20 * math.Log10(math.Max(math.Hypot(real(s[i]), imag(s[i]))/(float64(c.FFTSize)/4), 1e-12))
		out[i] = math.Max(c.MinDB, math.Min(c.MaxDB, db))
	}
	return out
}

type collector struct {
	limit        int
	frames       [][]float64
	stride, seen int
}

func (c *collector) add(f []float64) {
	if c.limit == 0 {
		c.frames = append(c.frames, f)
		c.seen++
		return
	}
	if len(c.frames) >= c.limit {
		c.frames = average(c.frames)
		c.stride *= 2
	}
	if c.seen%c.stride == 0 {
		c.frames = append(c.frames, f)
	}
	c.seen++
}
func average(rows [][]float64) [][]float64 {
	out := make([][]float64, 0, (len(rows)+1)/2)
	for i := 0; i < len(rows); i += 2 {
		if i+1 == len(rows) {
			out = append(out, rows[i])
			continue
		}
		r := make([]float64, len(rows[i]))
		for j := range r {
			r[j] = (rows[i][j] + rows[i+1][j]) / 2
		}
		out = append(out, r)
	}
	return out
}
