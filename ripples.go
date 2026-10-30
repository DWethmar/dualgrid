package dualgrid

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Ripples describes a generated, animated sheet of shore ripples: thin lines
// in the water that follow the shoreline, start at the land's edge, drift
// outward and fade.
//
// Use it as its own layer with the same Source as the land layer, drawn
// after the water and before the land. Its tiles are transparent on the
// land side, and tiles 0 (open water) and 15 (all land) are empty.
type Ripples struct {
	Color  color.NRGBA
	Frames int     // frames per cycle
	Count  int     // ripples in view at once, evenly spaced in time
	Reach  float64 // how far a ripple travels, in fill-amount units (0.5 is half a cell)
}

// DefaultRipples is a soft white ripple, two at a time, over 8 frames.
var DefaultRipples = Ripples{
	Color:  color.NRGBA{0xe8, 0xf6, 0xff, 0xd0},
	Frames: 8,
	Count:  2,
	Reach:  0.35,
}

// Band widths in fill-amount units: the gap between the shore and a fresh
// ripple, and a ripple's thickness (about one pixel at 16px).
const (
	rippleGap   = 0.04
	rippleWidth = 0.06
)

// Tileset returns the generated ripples as an animated Tileset.
func (r Ripples) Tileset(size int) *Tileset {
	ts, err := NewTileset(ebiten.NewImageFromImage(r.Sheet(size)), size)
	if err != nil {
		panic(err)
	}
	return ts
}

// Sheet draws the ripples: Frames blocks of 4×4 tiles, side by side. Write
// it to a PNG to get a template to paint over.
//
// Ripple i is at phase p = frame/Frames + i/Count (wrapping at 1). It is the
// band where the fill amount is a little below the shore's 0.5, moved out by
// p·Reach, and it fades in quickly and out slowly as p grows. The fill amount
// is exact on tile edges, so ripples line up across tiles like the shore.
func (r Ripples) Sheet(size int) *image.NRGBA {
	frames := max(r.Frames, 1)
	block := SheetTiles * size
	img := image.NewNRGBA(image.Rect(0, 0, frames*block, block))
	for fr := range frames {
		for i := 1; i < 15; i++ { // 0 and 15 have no shore
			m := Mask(i)
			t := tileRect(m, size).Add(image.Pt(fr*block, 0))
			for py := range size {
				for px := range size {
					f, _ := fillAt(m, px, py, size)
					if a := r.alpha(f, fr, frames); a > 0 {
						c := r.Color
						c.A = uint8(math.Round(float64(c.A) * a))
						img.SetNRGBA(t.Min.X+px, t.Min.Y+py, c)
					}
				}
			}
		}
	}
	return img
}

// alpha is how strongly any ripple covers a pixel with fill amount f in a
// frame, from 0 to 1.
func (r Ripples) alpha(f float64, frame, frames int) float64 {
	if f >= 0.5 {
		return 0 // land
	}
	count := max(r.Count, 1)
	best := 0.0
	for i := range count {
		p := math.Mod(float64(frame)/float64(frames)+float64(i)/float64(count), 1)
		outer := 0.5 - rippleGap - p*r.Reach // the ripple's far side
		if f > outer+rippleWidth || f <= outer {
			continue
		}
		// Fade in over the first fifth of the trip, out over the rest.
		a := math.Min(1, p*5) * (1 - p)
		best = math.Max(best, a)
	}
	return best
}
