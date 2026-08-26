# Spectrogram with Interactive Plotly Viewer

[![CI](https://github.com/landscape82/spectogram/actions/workflows/ci.yml/badge.svg)](https://github.com/landscape82/spectogram/actions/workflows/ci.yml)

This project allows you to generate a detailed **Spectrogram** using `Go` from an audio file (compressed `MP3` or simple `WAV`), visualize it as a **color PNG**, and interact with it using an **HTML + Plotly heatmap**.

> ⚠️ **Project status:** this is still under active development and is **not a single interactive app**. There's no server or UI tying the pieces together — using it is a manual, two-step workflow: (1) run the Go CLI to generate the spectrogram data, then (2) run a local web server to view it in the browser. See [Usage](#-usage) below for the exact steps. A more user-friendly, integrated workflow is tracked as a future improvement.

I've called this module `go-spectrogram-plotly`. Hope to improve it in the near future.

## ✨ Features

- 🎧 Supports `MP3` and `WAV` audio files
- 🔊 Uses `FFT` + `Hann` window for accurate frequency representation
- 🎨 Generates color spectrograms using a `Viridis-style` heatmap
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

The workflow is two separate, manual steps: first generate the data with the Go CLI (Step 1 below), then serve and view it with a local Python web server (Step 2, in the next section). There is currently no single command or app that does both.

### Step 1. Generate the spectrogram data

Run the spectrogram generator with:

```bash
go run cmd/spectogram/main.go -in audio.mp3 -out spectrogram.png -json data/spectrogram.json
```

- `-in` – path to your `WAV` or `MP3` file
- `-out` – name of the `PNG` image to be generated (default `spectrogram.png`)
- `-json` – path where the spectrogram matrix will be exported as `JSON` (default `data/spectrogram.json`)

This generates:
- image `spectrogram.png`
- output for Plotly in `data/spectrogram.json`, whose parent directory is created automatically if it doesn't exist

## 🌐 Step 2. View the Spectrogram (HTML + Plotly)

To view the spectrogram you generated in Step 1:

### Run local server

```bash
python3 -m http.server
```

### Open your browser

```
http://localhost:8000/web/index.html
or
http://localhost:8000/web
```

You should now see an interactive, zoomable Plotly heatmap.

## 🐳 Running with Docker

A `Dockerfile` is included so you can run both steps without installing Go or Python locally. Build the image once:

```bash
docker build -t spectogram .
```

### Step 1. Generate the spectrogram data

Mount a local directory containing your audio file to `/app/data`, and generate the output into it:

```bash
docker run --rm -v "$(pwd)":/app/data spectogram generate -in /app/data/audio.mp3 -out /app/data/spectrogram.png -json /app/data/spectrogram.json
```

### Step 2. View the Spectrogram

Using the same mount, so the web viewer can find `spectrogram.json`:

```bash
docker run --rm -p 8000:8000 -v "$(pwd)":/app/data spectogram serve
```

Then open `http://localhost:8000/web` in your browser, same as the local workflow above.

## 🧠 How It Works?

The `Go` script:
- Loads the audio and converts to mono (unfortunately)
- Splits into FFT windows (1024 samples, 50% overlap)
- Applies Hann window
- Computes dB scale magnitudes
- Normalizes intensities
- Renders heatmap with tick marks and Viridis-style gradient (tick positions are by index, not real seconds/Hz — sample rate isn't currently exported)
- Exports as `PNG` and `JSON`
- In this version you won't see Aphex Twin's face in "formula" track (mono analysis)

The `HTML` uses `Plotly.js` to render that `JSON` into an interactive spectrogram.

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

Unit and integration tests live alongside the code in `cmd/spectogram/main_test.go`, covering the FFT/Hann-window math, image and JSON output, and a full end-to-end run against a synthesized WAV file.

### Releases

Pushing a tag matching `vX.Y.Z` triggers [`.github/workflows/release.yml`](.github/workflows/release.yml), which builds binaries for Linux/macOS (amd64/arm64), generates release notes from the commit log since the previous tag, and publishes a GitHub Release. See [CHANGELOG.md](CHANGELOG.md).

## 📁 Folder Structure

```
.github/workflows/  - CI pipeline (build, vet, test, Docker image build)
cmd/spectogram/     - main Go application and tests (main.go, main_test.go)
web/                - HTML viewer with Plotly
data/               - generated spectrogram.json output (created automatically, not tracked in git)
Dockerfile          - containerized build (Go binary + Python static file server)
docker-entrypoint.sh - dispatches `generate` and `serve` container commands
README.md           - this file
go.mod              - Go module info
```

## 💡 Tips & Improvements

- You can increase resolution by changing `windowSize` and `step` in `main.go`
- Edit the Plotly colorscale or layout in `web/index.html` as you like
- In future will add support for selecting multiple `*.json` spectrograms
- Also add support for custom gradient style (with selecting palette)
- Will implement support for stereo spectrogram analysis!

## 📄 License

MIT – feel free to use, modify, and share.
