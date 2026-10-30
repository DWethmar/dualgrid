// Package dualgrid renders tile maps with the dual-grid technique.
//
// The world grid says what each cell is (for one terrain: filled or not).
// Tiles are not drawn on that grid. They are drawn on a second, "display"
// grid shifted half a tile up and left, so every display tile sits on a
// corner of the world grid and overlaps the four world cells around it.
// Those four cells form a 4-bit [Mask], which picks one of 16 tiles.
//
// Because the edge between two world cells runs through the middle of a
// display tile, 16 tiles cover every possible arrangement.
package dualgrid

// Source reports whether a world cell is filled. Cells outside the map should
// report false.
type Source interface {
	Filled(x, y int) bool
}

// Mask says which of a display tile's four world cells are filled. It is
// also the tile's index in a tileset sheet.
type Mask uint8

// The corner bits. A display tile at (x, y) overlaps the world cells
// (x-1, y-1), (x, y-1), (x-1, y) and (x, y).
const (
	BR Mask = 1 << iota // bottom-right: world cell (x, y)
	BL                  // bottom-left:  world cell (x-1, y)
	TR                  // top-right:    world cell (x, y-1)
	TL                  // top-left:     world cell (x-1, y-1)

	Empty Mask = 0
	Full  Mask = TL | TR | BL | BR
)

// MaskAt returns the mask of display tile (x, y), which sits on the corner
// shared by world cells (x-1, y-1), (x, y-1), (x-1, y) and (x, y).
func MaskAt(src Source, x, y int) Mask {
	var m Mask
	if src.Filled(x-1, y-1) {
		m |= TL
	}
	if src.Filled(x, y-1) {
		m |= TR
	}
	if src.Filled(x-1, y) {
		m |= BL
	}
	if src.Filled(x, y) {
		m |= BR
	}
	return m
}
