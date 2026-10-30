package dualgrid

// Grid is a fixed-size map of filled cells. It implements [Source]; cells
// outside it are empty.
type Grid struct {
	W, H  int
	cells []bool
}

// NewGrid returns an empty w×h grid.
func NewGrid(w, h int) *Grid {
	return &Grid{W: w, H: h, cells: make([]bool, w*h)}
}

// In reports whether (x, y) lies inside the grid.
func (g *Grid) In(x, y int) bool {
	return x >= 0 && y >= 0 && x < g.W && y < g.H
}

// Set fills or clears a cell. Cells outside the grid are ignored.
func (g *Grid) Set(x, y int, filled bool) {
	if g.In(x, y) {
		g.cells[y*g.W+x] = filled
	}
}

// Filled reports whether a cell is filled.
func (g *Grid) Filled(x, y int) bool {
	return g.In(x, y) && g.cells[y*g.W+x]
}
