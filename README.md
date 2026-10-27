# Spectrogram viewer

Generate a physically labelled spectrogram from `WAV` or `MP3` audio. The Go
CLI streams decoded samples through a Hann-windowed FFT, writes a PNG and a
compact, versioned JSON document, and its `serve` mode provides an interactive
Plotly view in seconds, Hz, and dBFS.

## Requirements

- Go `1.24+`
- Docker is optional

## Generate and view

```bash
go run ./cmd/spectogram -in audio.wav \
  -out output/spectrogram.png \
  -json data/spectrogram.json

go run ./cmd/spectogram serve
```

Open `http://localhost:8000/web/`. The viewer loads
`../data/spectrogram.json` by default and lists JSON documents in `data/` in
its input. To load another document directly, use
`http://localhost:8000/web/?file=../data/other.json`.

The output JSON has this stable shape:

```json
{
  "metadata": {
    "schemaVersion": 1,
    "sampleRate": 44100,
    "fftSize": 1024,
    "hopSize": 512,
    "frequencyStepHz": 43.06640625,
    "timeStepSeconds": 0.011609977324263039
  },
  "data": [[-100.0, -82.1]]
}
```

`data[timeFrame][frequencyBin]` is a dBFS magnitude. Metadata also records
source channel count, selected channel, duration, display range, and any
output-frame downsampling stride.

## Analysis options

```bash
go run ./cmd/spectogram -in audio.mp3 \
  -fft-size 2048 -hop-size 512 \
  -channel left -min-db -90 -max-db 0 \
  -max-frames 2000
```

- `-fft-size`: power-of-two FFT length; larger values improve frequency
  resolution while reducing time resolution.
- `-hop-size`: samples between windows. It must be from `1` to `-fft-size`.
- `-channel`: `mono`/`mid` (average), `left`, `right`, or `side`.
- `-min-db` and `-max-db`: dBFS values exported and used for the PNG/Plotly
  color range.
- `-max-frames`: bounds JSON, image width, and matrix memory. Once reached,
  the tool progressively averages adjacent time frames and records the result
  in `metadata.frameStride`. Use `0` (the default) to retain every frame.

The program streams audio through a single FFT-sized working window rather
than retaining the decoded waveform. The exported matrix necessarily remains
in memory until JSON/PNG generation, so use `-max-frames` for long recordings.
Incomplete final windows are intentionally not padded; an exact final window
is included.

`serve` accepts `-address`, `-web-dir`, and `-data-dir` if the default
directories or port `8000` do not suit your setup. It exposes data at
`/data/` and the generated-file list at `/api/spectrograms`.

## Docker

```bash
docker build -t spectogram .
docker run --rm -v "$(pwd)":/app/data spectogram generate \
  -in /app/data/audio.mp3 \
  -out /app/data/spectrogram.png \
  -json /app/data/spectrogram.json
docker run --rm -p 8000:8000 -v "$(pwd)":/app/data spectogram serve
```

Then open `http://localhost:8000/web/`.

## Development

```bash
go build ./cmd/...
go vet ./...
gofmt -l .
go test ./... -race -cover
```

Tests cover FFT/window behavior, exact final-window handling, dBFS peak
calibration, metadata, channel selection, bounded frame collection, output
directories, and a WAV end-to-end run.
