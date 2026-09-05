package scan

import (
	"image"
	"testing"
)

func TestOrderPoints_Scrambled(t *testing.T) {
	// br, tl, bl, tr, scrambled.
	scrambled := [4]image.Point{{10, 10}, {0, 0}, {0, 10}, {10, 0}}
	want := [4]image.Point{{0, 0}, {10, 0}, {10, 10}, {0, 10}}

	got := OrderPoints(scrambled)
	if got != want {
		t.Errorf("OrderPoints(%v) = %v, want %v", scrambled, got, want)
	}
}

func TestOrderPoints_AlreadyOrdered(t *testing.T) {
	ordered := [4]image.Point{{0, 0}, {10, 0}, {10, 10}, {0, 10}}

	got := OrderPoints(ordered)
	if got != ordered {
		t.Errorf("OrderPoints(%v) = %v, want unchanged %v", ordered, got, ordered)
	}
}

func TestOrderPoints_SkewedQuad(t *testing.T) {
	// A page photographed at a slight angle: still roughly upright, but
	// each corner is nudged off a perfect rectangle. Passed in scrambled
	// (br, bl, tl, tr) order.
	scrambled := [4]image.Point{{205, 300}, {8, 290}, {12, 10}, {210, 15}}
	want := [4]image.Point{{12, 10}, {210, 15}, {205, 300}, {8, 290}}

	got := OrderPoints(scrambled)
	if got != want {
		t.Errorf("OrderPoints(%v) = %v, want %v", scrambled, got, want)
	}
}

func TestOutputSize_Square(t *testing.T) {
	ordered := [4]image.Point{{0, 0}, {10, 0}, {10, 10}, {0, 10}}

	w, h := OutputSize(ordered)
	if w != 10 || h != 10 {
		t.Errorf("OutputSize(%v) = (%d, %d), want (10, 10)", ordered, w, h)
	}
}

func TestOutputSize_SkewedQuad(t *testing.T) {
	// Top edge (8px) shorter than bottom edge (10px); left edge (10px)
	// shorter than the slightly longer right edge (~10.2px).
	ordered := [4]image.Point{{0, 0}, {8, 0}, {10, 10}, {0, 10}}

	w, h := OutputSize(ordered)
	if w != 10 {
		t.Errorf("OutputSize width = %d, want 10", w)
	}
	if h < 10 || h > 11 {
		t.Errorf("OutputSize height = %d, want ~10", h)
	}
}
