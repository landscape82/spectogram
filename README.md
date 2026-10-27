# Spectrogram viewer

Generate a physically labelled spectrogram from `WAV` or `MP3` audio. The Go
CLI streams decoded samples through a Hann-windowed FFT, writes a PNG and a
compact versioned JSON document, and its `serve` mode provides an interactive
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

The stable JSON document has a `metadata` object and
`data[timeFrame][frequencyBin]` dBFS magnitudes. Metadata records the schema
version, sample rate, source and selected channels, FFT/hop sizes,
time/frequency resolution, duration, display range, and output-frame stride.

## Analysis options

```bash
go run ./cmd/spectogram -in audio.mp3 \
  -fft-size 2048 -hop-size 512 \
  -channel left -min-db -90 -max-db 0 \
  -max-frames 2000
```

- `-fft-size`: power-of-two FFT length; larger values improve frequency
  resolution while reducing time resolution.
- `-hop-size`: samples between windows, from `1` to `-fft-size`.
- `-channel`: `mono`/`mid` (average), `left`, `right`, or `side`.
- `-min-db` and `-max-db`: dBFS values exported and used for the PNG/Plotly
  color range.
- `-max-frames`: bounds JSON, image width, and matrix memory. Once reached,
  the tool progressively averages time frames and records
  `metadata.frameStride`. `0` retains every frame.

Audio is processed through one FFT-sized working window rather than retaining
the decoded waveform. The exported matrix remains in memory until output is
written, so use `-max-frames` for long recordings. Incomplete final windows
are not padded; an exact final window is included.

`serve` accepts `-address`, `-web-dir`, and `-data-dir`. It exposes data at
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

## Running on Kubernetes

A Helm chart at [`charts/spectogram`](charts/spectogram) deploys the
containerized viewer, with optional persistent storage and a generator Job.
See [`charts/spectogram/README.md`](charts/spectogram/README.md) for a `kind`
quickstart and cloud/on-prem configuration.

## Development

```bash
go build ./cmd/...
go vet ./...
gofmt -l .
go test ./... -race -cover
```

Tests cover FFT/window behavior, final-window handling, dBFS calibration,
metadata, channel selection, bounded frame collection, output directories,
the server endpoints, and a WAV end-to-end run.

## Releases

Pushing a `vX.Y.Z` tag triggers [`.github/workflows/release.yml`](.github/workflows/release.yml), which builds Linux/macOS binaries and publishes a GitHub Release. See [CHANGELOG.md](CHANGELOG.md).

## License

MIT – feel free to use, modify, and share.
