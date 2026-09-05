package scan

import (
	"image"
	"math"
)

// OrderPoints takes 4 points of a detected document quadrilateral in
// arbitrary order and returns them as [top-left, top-right, bottom-right,
// bottom-left]. It assumes the quad is roughly upright (not rotated more
// than ~45 degrees), which holds for photos of a page taken right-side up.
//
// The classic trick: top-left/bottom-right are found by the smallest and
// largest (x+y); top-right/bottom-left are found by the smallest and
// largest (y-x).
func OrderPoints(pts [4]image.Point) [4]image.Point {
	var sums, diffs [4]int
	for i, p := range pts {
		sums[i] = p.X + p.Y
		diffs[i] = p.Y - p.X
	}

	minSum, maxSum, minDiff, maxDiff := 0, 0, 0, 0
	for i := 1; i < 4; i++ {
		if sums[i] < sums[minSum] {
			minSum = i
		}
		if sums[i] > sums[maxSum] {
			maxSum = i
		}
		if diffs[i] < diffs[minDiff] {
			minDiff = i
		}
		if diffs[i] > diffs[maxDiff] {
			maxDiff = i
		}
	}

	return [4]image.Point{pts[minSum], pts[minDiff], pts[maxSum], pts[maxDiff]}
}

// OutputSize computes the width and height of the flattened rectangle that
// perspective-correcting an ordered quad [top-left, top-right, bottom-right,
// bottom-left] should produce: the larger of the top/bottom edge lengths,
// and the larger of the left/right edge lengths.
func OutputSize(ordered [4]image.Point) (width, height int) {
	tl, tr, br, bl := ordered[0], ordered[1], ordered[2], ordered[3]

	maxWidth := math.Max(dist(tl, tr), dist(bl, br))
	maxHeight := math.Max(dist(tl, bl), dist(tr, br))

	return int(math.Round(maxWidth)), int(math.Round(maxHeight))
}

func dist(a, b image.Point) float64 {
	dx := float64(a.X - b.X)
	dy := float64(a.Y - b.Y)
	return math.Sqrt(dx*dx + dy*dy)
}
