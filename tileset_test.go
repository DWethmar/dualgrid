package dualgrid

import (
	"image"
	"testing"
)

func TestSheetFrames(t *testing.T) {
	for _, c := range []struct{ w, h, size, want int }{
		{64, 64, 16, 1},
		{128, 64, 16, 2},
		{512, 64, 16, 8},
	} {
		if got, err := sheetFrames(c.w, c.h, c.size); err != nil || got != c.want {
			t.Errorf("sheetFrames(%d,%d,%d) = %d, %v; want %d", c.w, c.h, c.size, got, err, c.want)
		}
	}
	for _, c := range [][3]int{{64, 48, 16}, {60, 60, 16}, {96, 64, 16}, {32, 64, 16}, {64, 64, 0}, {64, 64, -16}} {
		if _, err := sheetFrames(c[0], c[1], c[2]); err == nil {
			t.Errorf("sheetFrames(%d,%d,%d) accepted", c[0], c[1], c[2])
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

func opaque(img image.Image, x, y int) bool {
	_, _, _, a := img.At(x, y).RGBA()
	return a != 0
}

// checkSeams fails if two tiles that can sit side by side (or one above the
// other) differ anywhere along their shared edge. dx is the frame's offset.
func checkSeams(t *testing.T, name string, img image.Image, dx, s int) {
	t.Helper()
	for a := range Mask(16) {
		for b := range Mask(16) {
			ra, rb := tileRect(a, s).Add(image.Pt(dx, 0)), tileRect(b, s).Add(image.Pt(dx, 0))
			// a left of b: a's TR/BR are b's TL/BL.
			if (a&TR != 0) == (b&TL != 0) && (a&BR != 0) == (b&BL != 0) {
				for i := range s {
					if img.At(ra.Max.X-1, ra.Min.Y+i) != img.At(rb.Min.X, rb.Min.Y+i) {
						t.Errorf("%s: %d|%d differ on their seam at row %d", name, a, b, i)
					}
				}
			}
			// a above b: a's BL/BR are b's TL/TR.
			if (a&BL != 0) == (b&TL != 0) && (a&BR != 0) == (b&TR != 0) {
				for i := range s {
					if img.At(ra.Min.X+i, ra.Max.Y-1) != img.At(rb.Min.X+i, rb.Min.Y) {
						t.Errorf("%s: %d over %d differ on their seam at column %d", name, a, b, i)
					}
				}
			}
		}
	}
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
		checkSeams(t, "placeholder", img, 0, s)
	}
}

func TestRipplesSheet(t *testing.T) {
	const s = 16
	r := DefaultRipples
	img := r.Sheet(s)
	if got := img.Bounds(); got.Dx() != r.Frames*4*s || got.Dy() != 4*s {
		t.Fatalf("sheet is %v, want %d frames of 64x64", got, r.Frames)
	}
	for fr := range r.Frames {
		dx := fr * 4 * s
		checkSeams(t, "ripples", img, dx, s)

		// Open water and all-land tiles have no shore, so no ripples; land
		// pixels stay clear for the land layer.
		for _, m := range []Mask{Empty, Full} {
			tr := tileRect(m, s).Add(image.Pt(dx, 0))
			for y := range s {
				for x := range s {
					if opaque(img, tr.Min.X+x, tr.Min.Y+y) {
						t.Fatalf("frame %d: tile %d is painted at (%d,%d)", fr, m, x, y)
					}
				}
			}
		}
		edge := tileRect(TL|TR, s).Add(image.Pt(dx, 0)) // land on top
		for y := range s / 2 {
			for x := range s {
				if opaque(img, edge.Min.X+x, edge.Min.Y+y) {
					t.Fatalf("frame %d: ripple on land at (%d,%d)", fr, x, y)
				}
			}
		}
	}

	// Ripples move: frames are not all the same.
	a, b := img.SubImage(image.Rect(0, 0, 64, 64)), img.SubImage(image.Rect(64, 0, 128, 64))
	same := true
	for y := range 64 {
		for x := range 64 {
			if a.At(x, y) != b.At(64+x, y) {
				same = false
			}
		}
	}
	if same {
		t.Error("frame 0 and frame 1 are identical")
	}
}
