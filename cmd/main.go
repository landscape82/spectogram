package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/wav"
	"gonum.org/v1/gonum/dsp/fourier"
)

const (
	windowSize = 1024
	step       = 512
)

func main() {
	input := flag.String("in", "", "Input audio file (WAV or MP3)")
	output := flag.String("out", "spectrogram.png", "Output PNG file")
	jsonOut := flag.String("json", "data/spectrogram.json", "Output JSON for Plotly")
	flag.Parse()

	if *input == "" {
		log.Fatal("Please specify input file using -in flag")
	}

	if err := run(*input, *output, *jsonOut); err != nil {
		log.Fatal(err)
	}
}

// run performs the full pipeline: decode input, compute the spectrogram,
// and write the JSON and PNG outputs.
func run(input, output, jsonOut string) error {
	f, err := os.Open(input)
	if err != nil {
		return err
	}
	defer f.Close()

	var streamer beep.StreamSeekCloser

	switch ext := filepath.Ext(input); ext {
	case ".mp3":
		streamer, _, err = mp3.Decode(f)
	case ".wav":
		streamer, _, err = wav.Decode(f)
	default:
		return fmt.Errorf("unsupported file format: %s", ext)
	}

	if err != nil {
		return err
	}
	defer streamer.Close()

	samples := decodeToMonoSamples(streamer)
	spectrogram, maxVal := computeSpectrogram(samples, windowSize, step)

	if err := writeJSON(jsonOut, spectrogram); err != nil {
		return err
	}

	if err := writeImage(output, spectrogram, windowSize, maxVal); err != nil {
		return err
	}

	fmt.Println("Spectrogram saved to", output)
	fmt.Println("JSON data saved to", jsonOut)
	return nil
}

// decodeToMonoSamples reads all samples from streamer, downmixing stereo to mono.
func decodeToMonoSamples(streamer beep.Streamer) []float64 {
	samples := make([]float64, 0)
	buf := make([][2]float64, 1024)

	for {
		n, ok := streamer.Stream(buf)
		if !ok {
			break
		}
		for i := 0; i < n; i++ {
			sample := (buf[i][0] + buf[i][1]) / 2
			samples = append(samples, sample)
		}
	}

	return samples
}

// computeSpectrogram splits samples into overlapping windows, applies a Hann
// window, and computes the dB-scale FFT magnitude for each window. It also
// returns the maximum magnitude value found across the whole spectrogram.
func computeSpectrogram(samples []float64, windowSize, step int) ([][]float64, float64) {
	fft := fourier.NewFFT(windowSize)

	var spectrogram [][]float64
	maxVal := -math.MaxFloat64

	for i := 0; i+windowSize < len(samples); i += step {
		window := applyHannWindow(append([]float64(nil), samples[i:i+windowSize]...))
		spectrum := fft.Coefficients(nil, window)
		magnitude := make([]float64, windowSize/2)
		for j := 0; j < len(magnitude); j++ {
			re := real(spectrum[j])
			im := imag(spectrum[j])
			v := 10 * math.Log10(re*re+im*im+1e-9)
			if v > maxVal {
				maxVal = v
			}
			magnitude[j] = v
		}
		spectrogram = append(spectrogram, magnitude)
	}

	return spectrogram, maxVal
}

// writeJSON marshals the spectrogram and writes it to path, creating any
// missing parent directories.
func writeJSON(path string, spectrogram [][]float64) error {
	jsonData, err := json.MarshalIndent(spectrogram, "", "  ")
	if err != nil {
		return err
	}

	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return os.WriteFile(path, jsonData, 0644)
}

// writeImage renders the spectrogram as a Viridis-style heatmap PNG with
// simple axis tick marks, and writes it to path.
func writeImage(path string, spectrogram [][]float64, windowSize int, maxVal float64) error {
	width := len(spectrogram)
	height := windowSize / 2
	img := image.NewRGBA(image.Rect(0, 0, width+60, height+40))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			v := spectrogram[x][y]
			c := viridisColor(v, maxVal)
			img.Set(x+60, height-y, c)
		}
	}

	// Simple axis labels
	for i := 0; i <= 10; i++ {
		y := height - i*height/10
		for dx := 0; dx < 50; dx++ {
			img.Set(dx+5, y, color.Black)
		}
	}

	for i := 0; i <= 10; i++ {
		x := i * width / 10
		for dy := 0; dy < 10; dy++ {
			img.Set(x+60, height+dy, color.Black)
		}
	}

	outFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer outFile.Close()

	return png.Encode(outFile, img)
}

func applyHannWindow(samples []float64) []float64 {
	for i := range samples {
		samples[i] *= 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(len(samples)-1)))
	}
	return samples
}

func viridisColor(value, max float64) color.Color {
	norm := (value + 100) / (max + 100)
	if norm < 0 {
		norm = 0
	}
	if norm > 1 {
		norm = 1
	}
	r := uint8(255 * norm)
	g := uint8(255 * (1 - norm*0.7))
	b := uint8(255 * (1 - norm))
	return color.RGBA{r, g, b, 255}
}
