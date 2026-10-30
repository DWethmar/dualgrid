package dualgrid

import "github.com/hajimehoshi/ebiten/v2"

// DrawOptions controls [Draw].
type DrawOptions struct {
	// GeoM maps world pixels (one world cell = Tileset.Size pixels, cell
	// (0,0) at the origin) to the destination, e.g. a camera's scale and
	// offset. The zero value is the identity.
	GeoM ebiten.GeoM

	// The world cells to cover, inclusive. Draw covers them completely,
	// including the half tiles that stick out past their edges.
	MinX, MinY, MaxX, MaxY int
}

// Draw draws one terrain layer. Display tile (x, y) is centered on the world
// corner (x*Size, y*Size). Empty tiles are skipped, so several layers can be
// drawn on top of each other, lowest first.
func Draw(dst *ebiten.Image, src Source, ts *Tileset, opts *DrawOptions) {
	s := ts.Size
	for y := opts.MinY; y <= opts.MaxY+1; y++ {
		for x := opts.MinX; x <= opts.MaxX+1; x++ {
			m := MaskAt(src, x, y)
			if m == Empty {
				continue
			}
			var op ebiten.DrawImageOptions
			op.GeoM.Translate(float64(x*s-s/2), float64(y*s-s/2))
			op.GeoM.Concat(opts.GeoM)
			dst.DrawImage(ts.Tile(m), &op)
		}
	}
}
