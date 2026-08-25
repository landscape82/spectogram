# Spectogram with Interactive Plotly Viewer

[![CI](https://github.com/landscape82/spectogram/actions/workflows/ci.yml/badge.svg)](https://github.com/landscape82/spectogram/actions/workflows/ci.yml)

This project allows you to generate a detailed **Spectogram** using `Go` from an audio file (compressed `MP3` or simple `WAV`), visualize it as a **color PNG**, and interact with it using an **HTML + Plotly heatmap**.

I've call this module `go-spectrogram-plotly`. Hope to improve it in near future.

## ✨ Features

- 🎧 Supports `MP3` and `WAV` audio files
- 🔊 Uses `FFT` + `Hann` window for accurate frequency representation
- 🎨 Generates color spectograms using a `Viridis-style` heatmap
- 🖼️ Saves as static `PNG` and `JSON` frequency matrix
- 📊 Interactive Plotly viewer with zoom and pan
- 📁 Clean folder structure and command-line usability

## 🔧 Requirements

- `Go` version `1.20+` (CI runs the test suite on Go `1.21`, `1.22`, and `1.23`, on Ubuntu and macOS)
- `Python3` (for local `HTML` server, build and tested with `3.9.21`)

## 📦 Installation

Clone the repository or unzip the downloaded archive.

```bash
cd spectogram
go mod tidy
```

This will fetch necessary Go modules (especially `beep`, `gonum`).

## 🚀 Usage

Run the spectogram generator with:

```bash
go run cmd/main.go -in audio.mp3 -out spectrogram.png -json data/spectrogram.json
```

- `-in` – path to your `WAV` or `MP3` file
- `-out` – name of the `PNG` image to be generated (default `spectrogram.png`)
- `-json` – path where the spectogram matrix will be exported as `JSON` (default `data/spectrogram.json`)

This generates:
- image `spectrogram.png`
- output for Plotly in `data/spectrogram.json`, whose parent directory is created automatically if it doesn't exist

## 🌐 View Interactive Spectogram (HTML + Plotly)

To view your Spectogram interactively:

### Step 1. Run local server

```bash
python3 -m http.server
```

### Step 2. Open your browser:

```
http://localhost:8000/web/index.html
or
http://localhost:8000/web
```

You should now see an interactive, zoomable Plotly heatmap.

## 🧠 How It Works?

The `Go` script:
- Loads the audio and converts to mono (unfortunetly)
- Splits into FFT windows (1024 samples, 50% overlap)
- Applies Hann window
- Computes dB scale magnitudes
- Normalizes intensities
- Renders heatmap with tick marks and Viridis-style gradient (tick positions are by index, not real seconds/Hz — sample rate isn't currently exported)
- Exports as `PNG` and `JSON`
- In this version you won't see Aphex Twin's face in "formula" track (mono analysis)

The `HTML` uses `Plotly.js` to render that `JSON` into an interactive spectogram.

## 🧪 Development & Testing

Continuous integration runs on every push and pull request to `main` via [GitHub Actions](.github/workflows/ci.yml). It builds the project, checks formatting, runs `go vet`, verifies `go.mod`/`go.sum` are tidy, and runs the test suite (with the race detector and coverage) across a matrix of Go versions and operating systems.

To run the same checks locally before pushing:

```bash
go build ./cmd/...
go vet ./...
gofmt -l .
go mod tidy && git diff --exit-code go.mod go.sum
go test ./... -race -cover
```

Unit and integration tests live alongside the code in `cmd/main_test.go`, covering the FFT/Hann-window math, image and JSON output, and a full end-to-end run against a synthesized WAV file.

## 📁 Folder Structure

```
.github/workflows/ - CI pipeline (build, vet, test)
cmd/                - main Go application and tests (main.go, main_test.go)
web/                - HTML viewer with Plotly
data/               - generated spectrogram.json output (created automatically, not tracked in git)
README.md           - this file
go.mod              - Go module info
```

## 💡 Tips & Improvments

- You can increase resolution by changing `windowSize` and `step` in `main.go`
- Edit the Plotly colorscale or layout in `web/index.html` as you like
- In future will add support for selecting multiple `*.json` spectograms
- Also add support for custom gradient style (with selecting pallete)
- Will implement support for stereo spectogram analysis!

## 📄 License

MIT – feel free to use, modify, and share.
