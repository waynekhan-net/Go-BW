# go-BW

A CLI that turns a folder of phone photos of paper documents into flattened,
black-and-white "scanned" images — the core of what iOS Notes.app's document
scanner (and the app formerly known as Microsoft Office Lens) does: detect
the page's edges, correct the perspective so it looks flat, crop to just the
page, and clean it up into a crisp black-and-white scan.

## Prerequisites

- Go 1.23+
- **OpenCV 4.x**, via Homebrew on macOS:

  ```bash
  brew install opencv@4 pkg-config
  ```

  `opencv@4` is required specifically — Homebrew's default `opencv` formula
  is now OpenCV 5.x, which [gocv](https://github.com/hybridgroup/gocv) (the
  Go/OpenCV bindings this tool uses) does not yet support. `opencv@4` is
  keg-only, so it won't be on the default pkg-config search path — export
  this before building or running:

  ```bash
  export PKG_CONFIG_PATH="$(brew --prefix opencv@4)/lib/pkgconfig:$PKG_CONFIG_PATH"
  ```

### Known limitation: HEIC photos

OpenCV's image reader does not support HEIC/HEIF, which is the default photo
format on iPhone. Convert photos to JPEG first, e.g.:

```bash
sips -s format jpeg *.heic
```

(or switch the iPhone Camera setting to "Most Compatible" before shooting).

## Build & run

```bash
go build -o go-bw .
./go-bw -input ./photos -output ./scanned
```

Flags:

| Flag | Default | Purpose |
|---|---|---|
| `-input` | *(required)* | folder of input photos (`.jpg`/`.jpeg`/`.png`) |
| `-output` | *(required)* | folder to write scanned output images to (created if missing) |
| `-format` | `jpg` | output image format: `jpg` or `png` |
| `-v` | `false` | verbose per-file progress/warnings |

Each input photo produces one output image of the same basename in the
output folder.

If no rectangular document can be detected in a photo (e.g. the page fills
the entire frame), the tool falls back to using the full photo instead of
failing that file, and logs a warning under `-v`.

## Tuning scan quality

The black-and-white conversion uses adaptive thresholding, controlled by two
constants in `internal/scan/pipeline.go`:

- `adaptiveThresholdBlockSize` — increase if output looks blotchy/noisy from
  paper texture or shadows.
- `adaptiveThresholdC` — adjust if output looks too dark or too washed out.

## Testing

- `go test ./...` runs unit tests for the pure-Go corner-ordering/geometry
  math (`internal/scan/geometry_test.go`), which don't require OpenCV.
- The OpenCV-based pipeline itself is verified manually, since golden-image
  testing of image-processing output is impractical:
  1. `go build -o go-bw .`
  2. Gather a small folder of real sample photos — angled, with background
     visible around the page — including at least one photo where the page
     fills the whole frame (to exercise the fallback path).
  3. `./go-bw -input ./samples -output ./scanned -v`
  4. Visually inspect each output: background should be cropped out, the
     page should look flat/rectangular, text should be legible and clean.
