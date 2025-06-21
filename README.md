# 🎵 Go Spectogram with Interactive Plotly Viewer

This project allows you to generate a detailed **Spectogram** from an audio file (compressed `MP3` or simple `WAV`), visualize it as a **color PNG**, and interact with it using an **HTML + Plotly heatmap**.

---

## ✨ Features

- 🎧 Supports `MP3` and `WAV` audio files
- 🔊 Uses `FFT` + `Hann` window for accurate frequency representation
- 🎨 Generates color spectograms using a `Viridis-style` heatmap
- 🖼️ Saves as static `PNG` and `JSON` frequency matrix
- 📊 Interactive Plotly viewer with zoom and pan
- 📁 Clean folder structure and command-line usability

---

## 🔧 Requirements

- `Go` 1.20`
- `Python` 3 (for local `HTML` server)
- `libmpg123` (for MP3 decoding, if needed)

---

## 📦 Installation

Clone the repository or unzip the downloaded archive.

```bash
cd go-spectogram-plotly
go mod tidy
```

This will fetch necessary Go modules (especially `beep`, `gonum`).

---

## 🚀 Usage

Run the spectogram generator with:

```bash
go run cmd/main.go -in audio.mp3 -out spectogram.png -json data/spectogram.json
```

- `-in` – path to your `WAV` or `MP3` file
- `-out` – name of the `PNG` image to be generated
- `-json` – path where the spectogram matrix will be exported as `JSON`

✅ Example:
```bash
go run cmd/main.go -in examples/drumloop.wav -out spectogram.png -json data/spectogram.json
```

This generates:
- image `spectogram.png`
- output for Plotly in `data/spectogram.json`

---

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

---

## 🧠 How It Works?

The `Go` script:
- Loads the audio and converts to mono (unfortunetly)
- Splits into FFT windows (1024 samples, 50% overlap)
- Applies Hann window
- Computes dB scale magnitudes
- Normalizes intensities
- Renders heatmap with axes and Viridis-style gradient
- Exports as `PNG` and `JSON`
- In this version you won't see Aphex Twin's face in "formula" track (mono analysis)

The `HTML` uses `Plotly.js` to render that `JSON` into an interactive spectogram.

---

## 📁 Folder Structure

```
cmd/           - main Go application
web/           - HTML viewer with Plotly
data/          - spectogram.json output goes here
README.md      - this file
go.mod         - Go module info
```

---

## 💡 Tips & Improvments

- MP3 support requires `libmpg123` (Linux: `sudo apt install libmpg123-dev`)
- You can increase resolution by changing `windowSize` and `step` in `main.go`
- Edit the Plotly colorscale or layout in `web/index.html` as you like
- In future will add support for selecting multiple `*.json` spectograms
- Also add support for custom gradient style (with selecting pallete)
- Will implement support for stereo spectogram analysis!

---

## 📄 License

MIT – feel free to use, modify, and share.
