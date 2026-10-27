package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/wav"
	"gonum.org/v1/gonum/dsp/fourier"
)

const (
	defaultFFTSize = 1024
	defaultHopSize = 512
)

// Config controls the short-time Fourier transform and the exported view.
type Config struct {
	FFTSize   int
	HopSize   int
	Channel   string
	MinDB     float64
	MaxDB     float64
	MaxFrames int
}

func defaultConfig() Config {
	return Config{FFTSize: defaultFFTSize, HopSize: defaultHopSize, Channel: "mono", MinDB: -100, MaxDB: 0}
}

func (c Config) validate() error {
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
	default:
		return fmt.Errorf("unsupported channel %q (use mono, left, right, mid, or side)", c.Channel)
	}
}

// Metadata makes every data point interpretable without relying on defaults.
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

// Spectrogram is the JSON document consumed by the viewer. Data is indexed as
// data[timeFrame][frequencyBin], with values in dBFS.
type Spectrogram struct {
	Metadata Metadata    `json:"metadata"`
	Data     [][]float64 `json:"data"`
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		serveCommand(os.Args[2:])
		return
	}
	config := defaultConfig()
	input := flag.String("in", "", "input audio file (WAV or MP3)")
	output := flag.String("out", "spectrogram.png", "output PNG file")
	jsonOut := flag.String("json", "data/spectrogram.json", "output JSON for the viewer")
	flag.IntVar(&config.FFTSize, "fft-size", config.FFTSize, "FFT size (power of two)")
	flag.IntVar(&config.HopSize, "hop-size", config.HopSize, "samples between FFT windows")
	flag.StringVar(&config.Channel, "channel", config.Channel, "mono, left, right, mid, or side")
	flag.Float64Var(&config.MinDB, "min-db", config.MinDB, "minimum dBFS exported and displayed")
	flag.Float64Var(&config.MaxDB, "max-db", config.MaxDB, "maximum dBFS exported and displayed")
	flag.IntVar(&config.MaxFrames, "max-frames", 0, "maximum output frames; 0 keeps all frames")
	flag.Parse()

	if *input == "" {
		log.Fatal("please specify input with -in")
	}
	if err := runWithConfig(*input, *output, *jsonOut, config); err != nil {
		log.Fatal(err)
	}
}

func serveCommand(args []string) {
	flags := flag.NewFlagSet("serve", flag.ExitOnError)
	address := flags.String("address", ":8000", "HTTP listen address")
	webDir := flags.String("web-dir", "web", "directory containing the viewer")
	dataDir := flags.String("data-dir", "data", "directory containing spectrogram JSON files")
	_ = flags.Parse(args)
	log.Printf("Serving viewer on http://localhost%s/web/", *address)
	if err := http.ListenAndServe(*address, newServer(*webDir, *dataDir)); err != nil {
		log.Fatal(err)
	}
}

func newServer(webDir, dataDir string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/spectrograms", func(w http.ResponseWriter, r *http.Request) {
		files, err := listSpectrograms(dataDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(files)
	})
	mux.Handle("/data/", http.StripPrefix("/data/", http.FileServer(http.Dir(dataDir))))
	mux.Handle("/web/", http.StripPrefix("/web/", http.FileServer(http.Dir(webDir))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/web/", http.StatusTemporaryRedirect)
	})
	return mux
}

func listSpectrograms(dataDir string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.WalkDir(dataDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(path), ".json") {
			relative, err := filepath.Rel(dataDir, path)
			if err != nil {
				return err
			}
			files = append(files, "/data/"+filepath.ToSlash(relative))
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

// run retains the original programmatic entry point with default options.
func run(input, output, jsonOut string) error {
	return runWithConfig(input, output, jsonOut, defaultConfig())
}

func runWithConfig(input, output, jsonOut string, config Config) error {
	if err := config.validate(); err != nil {
		return err
	}
	f, err := os.Open(input)
	if err != nil {
		return err
	}
	defer f.Close()

	var streamer beep.StreamSeekCloser
	var format beep.Format
	switch strings.ToLower(filepath.Ext(input)) {
	case ".mp3":
		streamer, format, err = mp3.Decode(f)
	case ".wav":
		streamer, format, err = wav.Decode(f)
	default:
		return fmt.Errorf("unsupported file format: %s", filepath.Ext(input))
	}
	if err != nil {
		return err
	}
	defer streamer.Close()

	spectrogram, err := streamSpectrogram(streamer, int(format.SampleRate), format.NumChannels, config)
	if err != nil {
		return err
	}
	if len(spectrogram.Data) == 0 {
		return fmt.Errorf("audio is shorter than one FFT window (%d samples)", config.FFTSize)
	}
	if err := writeJSON(jsonOut, spectrogram); err != nil {
		return err
	}
	if err := writeImage(output, spectrogram); err != nil {
		return err
	}
	fmt.Println("Spectrogram saved to", output)
	fmt.Println("JSON data saved to", jsonOut)
	return nil
}

// streamSpectrogram performs the STFT while retaining only one working window.
func streamSpectrogram(streamer beep.Streamer, sampleRate, channels int, config Config) (Spectrogram, error) {
	if err := config.validate(); err != nil {
		return Spectrogram{}, err
	}
	if sampleRate <= 0 {
		return Spectrogram{}, errors.New("audio has an invalid sample rate")
	}
	collector := newFrameCollector(config.MaxFrames)
	fft := fourier.NewFFT(config.FFTSize)
	window := make([]float64, 0, config.FFTSize)
	buffer := make([][2]float64, 4096)
	var sampleCount int

	for {
		n, ok := streamer.Stream(buffer)
		for i := 0; i < n; i++ {
			window = append(window, selectChannel(buffer[i], config.Channel))
			sampleCount++
			if len(window) == config.FFTSize {
				collector.add(computeFrame(window, fft, config))
				copy(window, window[config.HopSize:])
				window = window[:config.FFTSize-config.HopSize]
			}
		}
		if !ok {
			break
		}
	}

	stride := collector.stride
	return Spectrogram{
		Metadata: Metadata{
			SchemaVersion: 1, SampleRate: sampleRate, Channels: channels, Channel: config.Channel,
			FFTSize: config.FFTSize, HopSize: config.HopSize, FrameStride: stride,
			FrequencyStepHz: float64(sampleRate) / float64(config.FFTSize),
			TimeStepSeconds: float64(config.HopSize*stride) / float64(sampleRate),
			MinDB:           config.MinDB, MaxDB: config.MaxDB,
			DurationSeconds: float64(sampleCount) / float64(sampleRate),
		},
		Data: collector.frames,
	}, nil
}

func selectChannel(sample [2]float64, channel string) float64 {
	switch channel {
	case "left":
		return sample[0]
	case "right":
		return sample[1]
	case "side":
		return (sample[0] - sample[1]) / 2
	default: // mono and mid
		return (sample[0] + sample[1]) / 2
	}
}

func computeFrame(samples []float64, fft *fourier.FFT, config Config) []float64 {
	windowed := applyHannWindow(append([]float64(nil), samples...))
	spectrum := fft.Coefficients(nil, windowed)
	frame := make([]float64, config.FFTSize/2)
	// The Hann window has a coherent gain of 0.5, so a full-scale sine
	// aligned to an FFT bin has magnitude N/4 after windowing.
	denominator := float64(config.FFTSize) / 4
	for i := range frame {
		magnitude := math.Hypot(real(spectrum[i]), imag(spectrum[i])) / denominator
		db := 20 * math.Log10(math.Max(magnitude, 1e-12))
		frame[i] = clamp(db, config.MinDB, config.MaxDB)
	}
	return frame
}

// frameCollector bounds output width by repeatedly averaging adjacent frames.
// The source stream is still processed continuously, so decoded audio is never
// accumulated in memory.
type frameCollector struct {
	maxFrames int
	frames    [][]float64
	stride    int
	seen      int
}

func newFrameCollector(maxFrames int) *frameCollector {
	return &frameCollector{maxFrames: maxFrames, stride: 1}
}

func (c *frameCollector) add(frame []float64) {
	if c.maxFrames == 0 {
		c.frames = append(c.frames, frame)
		c.seen++
		return
	}
	if len(c.frames) >= c.maxFrames {
		c.frames = averagePairs(c.frames)
		c.stride *= 2
	}
	if c.seen%c.stride == 0 {
		c.frames = append(c.frames, frame)
	}
	c.seen++
}

func averagePairs(frames [][]float64) [][]float64 {
	result := make([][]float64, 0, (len(frames)+1)/2)
	for i := 0; i < len(frames); i += 2 {
		if i+1 == len(frames) {
			result = append(result, frames[i])
			continue
		}
		row := make([]float64, len(frames[i]))
		for j := range row {
			row[j] = (frames[i][j] + frames[i+1][j]) / 2
		}
		result = append(result, row)
	}
	return result
}

func writeJSON(path string, spectrogram Spectrogram) error {
	data, err := json.Marshal(spectrogram)
	if err != nil {
		return err
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0644)
}

func writeImage(path string, spectrogram Spectrogram) error {
	width, height := len(spectrogram.Data), spectrogram.Metadata.FFTSize/2
	if width == 0 || height == 0 {
		return errors.New("cannot render an empty spectrogram")
	}
	img := image.NewRGBA(image.Rect(0, 0, width+60, height+40))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	for x, frame := range spectrogram.Data {
		for y, value := range frame {
			img.Set(x+60, height-y-1, viridisColor(value, spectrogram.Metadata.MinDB, spectrogram.Metadata.MaxDB))
		}
	}
	for i := 0; i <= 10; i++ {
		y := height - 1 - i*(height-1)/10
		for dx := 0; dx < 50; dx++ {
			img.Set(dx+5, y, color.Black)
		}
		x := i * (width - 1) / 10
		for dy := 0; dy < 10; dy++ {
			img.Set(x+60, height+dy, color.Black)
		}
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func applyHannWindow(samples []float64) []float64 {
	if len(samples) < 2 {
		return samples
	}
	for i := range samples {
		samples[i] *= 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(len(samples)-1)))
	}
	return samples
}

// viridisColor is an interpolation of the published Viridis anchor colors.
func viridisColor(value, min, max float64) color.Color {
	t := clamp((value-min)/(max-min), 0, 1)
	anchors := [][3]uint8{{68, 1, 84}, {59, 82, 139}, {33, 145, 140}, {94, 201, 98}, {253, 231, 37}}
	position := t * float64(len(anchors)-1)
	i := int(position)
	if i == len(anchors)-1 {
		return color.RGBA{anchors[i][0], anchors[i][1], anchors[i][2], 255}
	}
	fraction := position - float64(i)
	lerp := func(a, b uint8) uint8 { return uint8(float64(a) + fraction*(float64(b)-float64(a))) }
	return color.RGBA{lerp(anchors[i][0], anchors[i+1][0]), lerp(anchors[i][1], anchors[i+1][1]), lerp(anchors[i][2], anchors[i+1][2]), 255}
}

func clamp(v, min, max float64) float64 { return math.Max(min, math.Min(max, v)) }
