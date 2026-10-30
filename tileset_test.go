package dualgrid

import (
	"image"
	"testing"
)

func TestCheckSheet(t *testing.T) {
	if err := checkSheet(64, 64, 16); err != nil {
		t.Errorf("64x64 at 16px: %v", err)
	}
	for _, c := range [][3]int{{64, 48, 16}, {60, 60, 16}, {64, 64, 0}, {64, 64, -16}} {
		if checkSheet(c[0], c[1], c[2]) == nil {
			t.Errorf("checkSheet(%d,%d,%d) accepted", c[0], c[1], c[2])
		}
	}
}

func TestTileRect(t *testing.T) {
	cases := map[Mask]image.Rectangle{
		0:  image.Rect(0, 0, 16, 16),
		3:  image.Rect(48, 0, 64, 16),
		4:  image.Rect(0, 16, 16, 32),
		15: image.Rect(48, 48, 64, 64),
	}
	for m, want := range cases {
		if got := tileRect(m, 16); got != want {
			t.Errorf("tileRect(%d) = %v, want %v", m, got, want)
		}
	}
}

func opaque(img *image.RGBA, x, y int) bool {
	return img.RGBAAt(x, y).A != 0
}

func TestPlaceholderSheet(t *testing.T) {
	const s = 16
	for _, tc := range []struct {
		base  Placeholder
		inset float64
	}{{Grass, 0}, {Grass, 0.1}, {Grass, 0.25}, {Rock, -0.15}, {Rock, 0}, {Rock, 0.1}} {
		p, inset := tc.base, tc.inset
		p.Inset = inset
		img := p.Sheet(s)

		full, empty := tileRect(Full, s), tileRect(Empty, s)
		for y := range s {
			for x := range s {
				if !opaque(img, full.Min.X+x, full.Min.Y+y) {
					t.Fatalf("inset %v: full tile has a hole at (%d,%d)", inset, x, y)
				}
				if opaque(img, empty.Min.X+x, empty.Min.Y+y) {
					t.Fatalf("inset %v: empty tile is painted at (%d,%d)", inset, x, y)
				}
			}
		}

		// Neighbouring tiles share the two cells on their common edge, so
		// the pixels along that edge must match exactly.
		for a := range Mask(16) {
			for b := range Mask(16) {
				ra, rb := tileRect(a, s), tileRect(b, s)
				// a left of b: a's TR/BR are b's TL/BL.
				if (a&TR != 0) == (b&TL != 0) && (a&BR != 0) == (b&BL != 0) {
					for i := range s {
						if img.RGBAAt(ra.Max.X-1, ra.Min.Y+i) != img.RGBAAt(rb.Min.X, rb.Min.Y+i) {
							t.Errorf("inset %v: %d|%d differ on their seam at row %d", inset, a, b, i)
						}
					}
				}
				// a above b: a's BL/BR are b's TL/TR.
				if (a&BL != 0) == (b&TL != 0) && (a&BR != 0) == (b&TR != 0) {
					for i := range s {
						if img.RGBAAt(ra.Min.X+i, ra.Max.Y-1) != img.RGBAAt(rb.Min.X+i, rb.Min.Y) {
							t.Errorf("inset %v: %d over %d differ on their seam at column %d", inset, a, b, i)
						}
					}
				}
			}
		}
	}
}
