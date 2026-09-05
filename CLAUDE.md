# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A CLI that takes a folder of phone photos of paper documents and produces
flattened, black-and-white "scanned" versions of each — replicating the
core of iOS Notes.app's document scanner / (pre-"Microsoft Lens") Office
Lens: detect the page's edges, perspective-correct it flat, crop, and
binarize to a clean scanned look. One output image per input photo (not a
combined PDF).

## Required environment: OpenCV 4.x via pkg-config

This project uses `gocv` (cgo bindings to OpenCV), so building, testing, or
vetting `internal/scan` requires a real OpenCV install discoverable via
`pkg-config` — **not just `go build`**.

**Homebrew's plain `opencv` formula is now OpenCV 5.x, which gocv does not
support.** You must install `opencv@4` specifically, and since it's
keg-only, point `PKG_CONFIG_PATH` at it for every build/test/vet command:

```bash
brew install opencv@4 pkg-config
export PKG_CONFIG_PATH="$(brew --prefix opencv@4)/lib/pkgconfig:$PKG_CONFIG_PATH"
```

Without that export, any command touching `internal/scan` (which is most
of them) fails at the cgo/link step, not with a Go-level error.

## Commands

```bash
go build -o go-bw .                 # build the CLI
go test ./...                       # run all tests
go test ./internal/scan/ -run TestOrderPoints_SkewedQuad -v   # single test
go vet ./...

./go-bw -input ./photos -output ./scanned -v   # run it
```

`go.mod`'s only dependency is `gocv.io/x/gocv`.

There is no automated test for the OpenCV pipeline itself (impractical
without a curated golden-image corpus). Verification is manual: build,
point `-input` at a folder of real sample photos (angled, background
visible, include one where the page fills the whole frame to exercise the
fallback path), run with `-v`, and visually inspect `-output`.

## Architecture

Two-package split, deliberately drawn along the "needs OpenCV" line:

- **`internal/scan/geometry.go`** — pure Go, no gocv import. `OrderPoints`
  takes 4 arbitrary-order corners and returns them as
  [top-left, top-right, bottom-right, bottom-left] via the classic
  sum/diff-of-coordinates trick (assumes the quad is roughly upright, which
  holds for a page photographed right-side up). `OutputSize` derives the
  flattened rectangle's width/height from those ordered corners. Because
  this file has no gocv dependency, `internal/scan/geometry_test.go` runs
  anywhere, including without OpenCV installed — this is the only
  meaningfully unit-testable part of the pipeline.

- **`internal/scan/pipeline.go`** — the only file that imports
  `gocv.io/x/gocv`; all OpenCV calls live here. `ProcessImage(inputPath,
  outputPath, opts)` is the entry point and runs:
  1. Load, downscale to ~500px height for detection speed (ratio kept to
     scale points back up later).
  2. Grayscale → GaussianBlur → Canny.
  3. **Dilate the edge map before `FindContours`.** This isn't optional
     polish — without it, JPEG-compression softening on diagonal edges
     fragments the page boundary into several open contours instead of one
     closed loop, and detection silently fails (found this empirically
     while building the pipeline; don't remove it).
  4. Find the largest contour that `ApproxPolyDP` reduces to exactly 4
     points.
  5. **Fallback, not error**: if no 4-point contour is found (e.g. the page
     fills the whole frame), use the full image bounds as the quad instead
     of failing the file — log a warning, same code path continues (the
     perspective transform becomes a no-op crop). A single bad photo must
     never abort the batch.
  6. Scale the ordered quad back to full resolution, perspective-warp
     (`GetPerspectiveTransform` + `WarpPerspective`), then grayscale +
     `AdaptiveThreshold` (Gaussian, binary) for the scanned B&W look.
  7. `gocv.IMWrite` to the output path.

  `adaptiveThresholdBlockSize`/`adaptiveThresholdC` (package-level
  constants) are the scan-quality knobs — blotchy/noisy output means raise
  `blockSize`; too dark/washed-out means adjust `cConstant`. They're
  intentionally not CLI flags.

- **`main.go`** — stdlib `flag` only (no cobra/etc — deliberate, this is a
  single command with 4 flags). Non-recursive `os.ReadDir` over `-input`,
  filtered to `.jpg`/`.jpeg`/`.png` (all gocv/OpenCV reliably reads).
  **HEIC/HEIF (default iPhone format) is not supported by OpenCV's
  `imread`** — users must convert first (`sips -s format jpeg *.heic`).
  Files are processed **sequentially, not concurrently** — OpenCV is
  already internally multi-threaded and a typical scan-session folder is
  small, so a Go worker pool would add complexity for no real wall-clock
  gain, and sequential processing keeps per-file log lines in input order.
  Each file's error is caught, logged, and skipped; one bad file never
  aborts the run.
