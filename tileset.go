package dualgrid

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// SheetTiles is the number of tiles per row and per column of a sheet.
const SheetTiles = 4

// Tileset holds the 16 tiles of one terrain, indexed by [Mask].
type Tileset struct {
	Size  int // width and height of one tile, in pixels
	tiles [16]*ebiten.Image
}

// NewTileset slices a sheet of 4×4 tiles of size×size pixels. Tile i is at
// column i%4, row i/4, so the sheet reads 0..15 left to right, top to bottom.
func NewTileset(sheet *ebiten.Image, size int) (*Tileset, error) {
	b := sheet.Bounds()
	if err := checkSheet(b.Dx(), b.Dy(), size); err != nil {
		return nil, err
	}
	ts := &Tileset{Size: size}
	for i := range ts.tiles {
		r := tileRect(Mask(i), size).Add(b.Min)
		ts.tiles[i] = sheet.SubImage(r).(*ebiten.Image)
	}
	return ts, nil
}

// Tile returns the tile for a mask.
func (t *Tileset) Tile(m Mask) *ebiten.Image {
	return t.tiles[m&Full]
}

// Placeholder describes generated tiles: rounded blobs of one color with a
// darker rim, so the package works before any art is drawn.
type Placeholder struct {
	Fill, Rim color.RGBA

	// Inset pulls the edge into the filled cells. At 0 the edge lies exactly
	// on the cell border; 0.1 moves it about a tenth of a tile inward. An
	// upper layer with a larger inset than the layer below leaves a strip of
	// the lower layer visible along its edge (a beach between grass and water).
	Inset float64

	// Highlight and Shadow give a raised look, lit from above: a light band
	// inside edges that face up, and a wide dark band inside edges that face
	// down (the side of a boulder). Leave them zero for flat tiles.
	Highlight, Shadow color.RGBA
}

// Grass is the default placeholder look.
var Grass = Placeholder{
	Fill: color.RGBA{0x5a, 0xa8, 0x3c, 0xff},
	Rim:  color.RGBA{0x2f, 0x6b, 0x2a, 0xff},
}

// Rock is a raised placeholder: grey boulders, lit from above. A negative
// inset lets it spread a little past its cells, so a lone boulder fills its
// cell instead of shrinking to a pebble.
var Rock = Placeholder{
	Inset:     -0.15,
	Fill:      color.RGBA{0x8c, 0x8a, 0x92, 0xff},
	Rim:       color.RGBA{0x34, 0x32, 0x3a, 0xff},
	Highlight: color.RGBA{0xbc, 0xba, 0xc4, 0xff},
	Shadow:    color.RGBA{0x5c, 0x5a, 0x64, 0xff},
}

// Band widths in fill-amount units (see Sheet), and how steeply an edge must
// face up or down to be shaded.
const (
	rimWidth       = 0.12
	shadowWidth    = 0.3
	highlightWidth = 0.14
	facing         = 0.3
)

// Tileset returns the generated tiles as a Tileset.
func (p Placeholder) Tileset(size int) *Tileset {
	ts, err := NewTileset(ebiten.NewImageFromImage(p.Sheet(size)), size)
	if err != nil {
		panic(err)
	}
	return ts
}

// Sheet draws a 4×4 sheet of generated tiles. Write it to a PNG to get a
// template to paint over.
//
// The centers of a display tile's four world cells lie on the tile's
// corners, so each pixel gets a "fill amount": the four corner values blended
// by distance (bilinear interpolation). Pixels above the edge threshold are
// filled, and those just above it form the rim. The outermost pixel rows and
// columns sit exactly on the tile's edge, where the amount depends only on
// the two cells that neighbouring tiles share, so tiles always line up.
//
// Shading looks at which way the edge faces: dv, how fast the fill amount
// changes going down. dv is shared along left and right tile edges too, but
// not along top and bottom ones, so the top and bottom pixel rows are never
// shaded. Up- and down-facing edges run through the middle of a tile, so
// their bands don't reach those rows anyway.
func (p Placeholder) Sheet(size int) *image.RGBA {
	raised := p.Highlight != (color.RGBA{}) || p.Shadow != (color.RGBA{})
	img := image.NewRGBA(image.Rect(0, 0, SheetTiles*size, SheetTiles*size))
	edge := 0.5 + p.Inset
	pos := func(i int) float64 {
		if size == 1 {
			return 0.5
		}
		return float64(i) / float64(size-1)
	}
	for i := range 16 {
		m := Mask(i)
		r := tileRect(m, size)
		corner := func(bit Mask) float64 {
			if m&bit != 0 {
				return 1
			}
			return 0
		}
		tl, tr, bl, br := corner(TL), corner(TR), corner(BL), corner(BR)
		for py := range size {
			v := pos(py)
			for px := range size {
				u := pos(px)
				f := tl*(1-u)*(1-v) + tr*u*(1-v) + bl*(1-u)*v + br*u*v
				dv := (bl-tl)*(1-u) + (br-tr)*u
				shade := raised && py > 0 && py < size-1
				var c color.RGBA
				switch {
				case f < edge:
					continue
				case f < edge+rimWidth:
					c = p.Rim
				case shade && dv < -facing && f < edge+rimWidth+shadowWidth:
					c = p.Shadow
				case shade && dv > facing && f < edge+rimWidth+highlightWidth:
					c = p.Highlight
				default:
					c = p.Fill
				}
				img.SetRGBA(r.Min.X+px, r.Min.Y+py, c)
			}
		}
	}
	return img
}

// tileRect is where a mask's tile sits in a sheet whose top-left is (0,0).
func tileRect(m Mask, size int) image.Rectangle {
	x := int(m) % SheetTiles * size
	y := int(m) / SheetTiles * size
	return image.Rect(x, y, x+size, y+size)
}

func checkSheet(w, h, size int) error {
	if size <= 0 {
		return fmt.Errorf("dualgrid: tile size must be positive, got %d", size)
	}
	if w != SheetTiles*size || h != SheetTiles*size {
		return fmt.Errorf("dualgrid: sheet is %dx%d, want %dx%d (4x4 tiles of %dpx)",
			w, h, SheetTiles*size, SheetTiles*size, size)
	}
	return nil
}
