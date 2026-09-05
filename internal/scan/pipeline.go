package scan

import (
	"fmt"
	"image"
	"log"

	"gocv.io/x/gocv"
)

// Options controls per-run behavior of ProcessImage.
type Options struct {
	Verbose bool
}

const (
	// detectHeight is the height (px) images are downscaled to before
	// running edge/contour detection. Detection doesn't need full
	// resolution and this keeps it fast on large phone photos.
	detectHeight = 500.0

	// approxEpsilonRatio is the epsilon for ApproxPolyDP, as a fraction
	// of the contour's perimeter. Larger values simplify more
	// aggressively; smaller values require closer-to-exact quads.
	approxEpsilonRatio = 0.02

	// adaptiveThresholdBlockSize and adaptiveThresholdC control the
	// "scanned" black-and-white look. If output looks blotchy/noisy from
	// paper texture or shadows, increase blockSize. If it's too dark or
	// too washed out, adjust cConstant. blockSize must be odd.
	adaptiveThresholdBlockSize = 35
	adaptiveThresholdC         = 15
)

// ProcessImage reads the photo at inputPath, detects the document within
// it, perspective-corrects and crops to that document, converts it to a
// clean black-and-white "scanned" look, and writes the result to
// outputPath.
func ProcessImage(inputPath, outputPath string, opts Options) error {
	img := gocv.IMRead(inputPath, gocv.IMReadColor)
	if img.Empty() {
		return fmt.Errorf("could not read image")
	}
	defer img.Close()

	quad, ratio, err := detectDocumentQuad(img)
	if err != nil {
		return err
	}
	if quad == nil {
		if opts.Verbose {
			log.Printf("WARN: no document contour found in %s, using full frame", inputPath)
		}
		w, h := img.Cols(), img.Rows()
		quad = &[4]image.Point{
			{0, 0}, {w, 0},
			{w, h}, {0, h},
		}
		ratio = 1.0
	}

	ordered := OrderPoints(*quad)
	fullRes := scalePoints(ordered, 1.0/ratio)
	width, height := OutputSize(fullRes)
	if width <= 0 || height <= 0 {
		return fmt.Errorf("degenerate document region detected")
	}

	warped := warpPerspective(img, fullRes, width, height)
	defer warped.Close()

	result := toScannedBW(warped)
	defer result.Close()

	if ok := gocv.IMWrite(outputPath, result); !ok {
		return fmt.Errorf("failed to write output image")
	}
	return nil
}

// detectDocumentQuad locates the largest 4-point contour in img, working on
// a downscaled copy for speed. It returns the quad's 4 corners in the
// downscaled image's coordinate space along with the scale ratio used
// (downscaled/original), so callers can map back to full resolution. If no
// suitable quad is found, it returns a nil quad (not an error) so callers
// can apply a fallback.
func detectDocumentQuad(img gocv.Mat) (quad *[4]image.Point, ratio float64, err error) {
	ratio = detectHeight / float64(img.Rows())

	resized := gocv.NewMat()
	defer resized.Close()
	gocv.Resize(img, &resized, image.Point{}, ratio, ratio, gocv.InterpolationArea)

	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(resized, &gray, gocv.ColorBGRToGray)

	blurred := gocv.NewMat()
	defer blurred.Close()
	gocv.GaussianBlur(gray, &blurred, image.Pt(5, 5), 0, 0, gocv.BorderDefault)

	edges := gocv.NewMat()
	defer edges.Close()
	gocv.Canny(blurred, &edges, 75, 200)

	// Dilate to close small gaps in the edge map (e.g. from JPEG
	// compression softening a diagonal edge) so the document's boundary
	// traces as a single closed contour rather than fragmenting into
	// several open ones.
	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
	defer kernel.Close()
	dilated := gocv.NewMat()
	defer dilated.Close()
	gocv.Dilate(edges, &dilated, kernel)

	contours := gocv.FindContours(dilated, gocv.RetrievalExternal, gocv.ChainApproxSimple)
	defer contours.Close()

	bestArea := 0.0
	var best *[4]image.Point
	for i := 0; i < contours.Size(); i++ {
		c := contours.At(i)
		area := gocv.ContourArea(c)
		if area <= bestArea {
			continue
		}

		peri := gocv.ArcLength(c, true)
		approx := gocv.ApproxPolyDP(c, approxEpsilonRatio*peri, true)
		if approx.Size() == 4 {
			pts := approx.ToPoints()
			best = &[4]image.Point{pts[0], pts[1], pts[2], pts[3]}
			bestArea = area
		}
	}

	return best, ratio, nil
}

func scalePoints(pts [4]image.Point, scale float64) [4]image.Point {
	var out [4]image.Point
	for i, p := range pts {
		out[i] = image.Pt(int(float64(p.X)*scale), int(float64(p.Y)*scale))
	}
	return out
}

// warpPerspective perspective-corrects img so the quadrilateral defined by
// ordered corners [tl, tr, br, bl] becomes a flat width x height rectangle.
func warpPerspective(img gocv.Mat, ordered [4]image.Point, width, height int) gocv.Mat {
	srcPts := gocv.NewPointVectorFromPoints([]image.Point{
		ordered[0], ordered[1], ordered[2], ordered[3],
	})
	defer srcPts.Close()

	dstPts := gocv.NewPointVectorFromPoints([]image.Point{
		{0, 0}, {width - 1, 0}, {width - 1, height - 1}, {0, height - 1},
	})
	defer dstPts.Close()

	transform := gocv.GetPerspectiveTransform(srcPts, dstPts)
	defer transform.Close()

	warped := gocv.NewMat()
	gocv.WarpPerspective(img, &warped, transform, image.Pt(width, height))
	return warped
}

// toScannedBW converts a warped color document image into a clean
// black-and-white "scanned" look via adaptive thresholding.
func toScannedBW(warped gocv.Mat) gocv.Mat {
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(warped, &gray, gocv.ColorBGRToGray)

	result := gocv.NewMat()
	gocv.AdaptiveThreshold(gray, &result, 255,
		gocv.AdaptiveThresholdGaussian, gocv.ThresholdBinary,
		adaptiveThresholdBlockSize, adaptiveThresholdC)
	return result
}
