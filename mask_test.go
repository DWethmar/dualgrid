package dualgrid

import "testing"

func TestMaskAtEveryMask(t *testing.T) {
	for i := range 16 {
		want := Mask(i)
		g := NewGrid(2, 2)
		g.Set(0, 0, want&TL != 0)
		g.Set(1, 0, want&TR != 0)
		g.Set(0, 1, want&BL != 0)
		g.Set(1, 1, want&BR != 0)
		if got := MaskAt(g, 1, 1); got != want {
			t.Errorf("mask %d: got %d", want, got)
		}
	}
}

// A single filled cell is seen by the four display tiles on its corners,
// each from a different side.
func TestMaskAtSingleCell(t *testing.T) {
	g := NewGrid(1, 1)
	g.Set(0, 0, true)
	cases := []struct {
		x, y int
		want Mask
	}{
		{0, 0, BR}, // the cell's top-left corner
		{1, 0, BL}, // top-right
		{0, 1, TR}, // bottom-left
		{1, 1, TL}, // bottom-right
		{2, 2, Empty},
		{-1, -1, Empty},
	}
	for _, c := range cases {
		if got := MaskAt(g, c.x, c.y); got != c.want {
			t.Errorf("MaskAt(%d,%d) = %d, want %d", c.x, c.y, got, c.want)
		}
	}
}

func TestGridOutOfBounds(t *testing.T) {
	g := NewGrid(2, 2)
	g.Set(5, 5, true) // ignored
	for _, p := range [][2]int{{-1, 0}, {0, -1}, {2, 0}, {0, 2}, {5, 5}} {
		if g.Filled(p[0], p[1]) {
			t.Errorf("Filled(%d,%d) outside the grid", p[0], p[1])
		}
	}
}
