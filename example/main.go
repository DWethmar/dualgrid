// Command example is a paint demo for the dualgrid package with four
// terrains: water, dirt, grass and rock.
//
//	go run ./example                                  # generated placeholder tiles
//	go run ./example -dirt dirt.png -grass grass.png  # your own 64x64 sheets of 16px tiles (also -rock, -ripples)
//	go run ./example -template DIR                    # write placeholder sheets to DIR and exit
//
// 1/2/3/4 pick water/dirt/grass/rock, left mouse paints, right mouse paints water,
// G toggles the grid overlay.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/dwethmar/dualgrid"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	tileSize = 16 // pixels per tile in the art
	scale    = 2  // drawn at 2x: one world cell is 32 screen pixels
	cell     = tileSize * scale
	worldW   = 24
	worldH   = 16
	screenW  = 960
	screenH  = 640
)

// Terrain is what a world cell is. The order is the drawing order: each
// terrain is drawn on top of the ones before it.
type Terrain uint8

const (
	Water Terrain = iota
	Dirt
	Grass
	Rock
)

var terrainNames = [...]string{"water", "dirt", "grass", "rock"}

func (t Terrain) String() string { return terrainNames[t] }

// World holds one Terrain per cell. Cells outside it are water.
type World struct {
	cells [worldW * worldH]Terrain
}

func (w *World) In(x, y int) bool { return x >= 0 && y >= 0 && x < worldW && y < worldH }

func (w *World) At(x, y int) Terrain {
	if !w.In(x, y) {
		return Water
	}
	return w.cells[y*worldW+x]
}

func (w *World) Set(x, y int, t Terrain) {
	if w.In(x, y) {
		w.cells[y*worldW+x] = t
	}
}

// AtLeast is one layer's view of the world: a cell is filled when its
// terrain is this one *or any terrain drawn above it*. So grass cells count
// as dirt too, and the dirt layer runs on under the grass instead of leaving
// a gap where grass meets water.
type AtLeast struct {
	W *World
	T Terrain
}

func (a AtLeast) Filled(x, y int) bool { return a.W.At(x, y) >= a.T }

// layer is one dual-grid terrain layer.
type layer struct {
	terrain Terrain
	tiles   *dualgrid.Tileset
}

var (
	bgColor        = color.RGBA{0x20, 0x20, 0x28, 0xff}
	waterColor     = color.RGBA{0x3a, 0x78, 0xb8, 0xff}
	waveColor      = color.RGBA{0x6a, 0xa4, 0xd8, 0xff}
	worldLineColor = color.RGBA{0xff, 0xff, 0xff, 0x50}
	dualLineColor  = color.RGBA{0xff, 0xd0, 0x30, 0xa0}
	hoverColor     = color.RGBA{0xff, 0xff, 0xff, 0xc0}

	dirtLook = dualgrid.Placeholder{
		Fill: color.RGBA{0xb0, 0x8a, 0x5a, 0xff},
		Rim:  color.RGBA{0x7a, 0x5a, 0x38, 0xff},
	}
	// Grass sits further in than dirt, so a strip of dirt (a beach) shows
	// wherever grass meets water.
	grassLook = func() dualgrid.Placeholder {
		p := dualgrid.Grass
		p.Inset = 0.12
		return p
	}()
	rockLook = dualgrid.Rock
)

// rippleTicks is how long each ripple frame shows, in ticks (60 per second).
const rippleTicks = 8

type game struct {
	world    World
	layers   []layer           // drawn in order, after the water
	ripples  *dualgrid.Tileset // animated, between the water and the land
	brush    Terrain
	showGrid bool
	ticks    int

	// Where world pixel (0,0) lands on screen: the world is centered.
	ox, oy int
}

func newGame(dirt, grass, rock, ripples *dualgrid.Tileset) *game {
	g := &game{
		layers:   []layer{{Dirt, dirt}, {Grass, grass}, {Rock, rock}},
		ripples:  ripples,
		brush:    Grass,
		showGrid: true,
		ox:       (screenW - worldW*cell) / 2,
		oy:       (screenH - worldH*cell) / 2,
	}
	// An island of dirt with a lake, grass fields, and a pond in the grass.
	for y := 1; y < worldH-1; y++ {
		for x := 1; x < worldW-1; x++ {
			g.world.Set(x, y, Dirt)
		}
	}
	for y := 2; y < 10; y++ {
		for x := 2; x < 12; x++ {
			g.world.Set(x, y, Grass)
		}
	}
	for y := 7; y < 13; y++ {
		for x := 14; x < 21; x++ {
			if (x-17)*(x-17)+(y-10)*(y-10) <= 7 {
				g.world.Set(x, y, Water)
			}
		}
	}
	for _, p := range [][2]int{{5, 4}, {6, 4}, {5, 5}, {6, 5}, {7, 5}} {
		g.world.Set(p[0], p[1], Water)
	}
	for y := 9; y < 14; y++ {
		g.world.Set(12, y, Grass)
		g.world.Set(13, y, Grass)
	}
	g.world.Set(14, 10, Grass) // grass straight against the lake
	// Boulders: a lone one, a pair, and a rocky ridge.
	for _, p := range [][2]int{{9, 7}, {3, 11}, {4, 11}, {18, 2}, {19, 2}, {20, 2}, {19, 3}, {20, 3}, {20, 4}} {
		g.world.Set(p[0], p[1], Rock)
	}
	return g
}

// cellAt converts a screen position to a world cell.
func (g *game) cellAt(sx, sy int) (int, int) {
	return floorDiv(sx-g.ox, cell), floorDiv(sy-g.oy, cell)
}

func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

func (g *game) Update() error {
	g.ticks++
	if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		g.showGrid = !g.showGrid
	}
	for i, k := range []ebiten.Key{ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4} {
		if inpututil.IsKeyJustPressed(k) {
			g.brush = Terrain(i)
		}
	}
	x, y := g.cellAt(ebiten.CursorPosition())
	switch {
	case ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft):
		g.world.Set(x, y, g.brush)
	case ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight):
		g.world.Set(x, y, Water)
	}
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	screen.Fill(bgColor)

	// Water is the bottom layer, so it has no edges of its own: it is a fill
	// under the whole world (plus a half cell, where the edge tiles reach).
	half := cell / 2
	fillRect(screen, g.ox-half, g.oy-half, worldW*cell+cell, worldH*cell+cell, waterColor)
	g.drawWaves(screen)

	opts := &dualgrid.DrawOptions{MaxX: worldW - 1, MaxY: worldH - 1}
	opts.GeoM.Scale(scale, scale)
	opts.GeoM.Translate(float64(g.ox), float64(g.oy))

	// Shore ripples: the same "is it land?" mask as the dirt layer, so they
	// follow the shoreline, but drawn under it and animated.
	opts.Frame = g.ticks / rippleTicks
	dualgrid.Draw(screen, AtLeast{&g.world, Dirt}, g.ripples, opts)
	opts.Frame = 0

	// Every other terrain is a dual-grid layer, lowest first.
	for _, l := range g.layers {
		dualgrid.Draw(screen, AtLeast{&g.world, l.terrain}, l.tiles, opts)
	}

	if g.showGrid {
		g.drawOverlay(screen)
	}

	mx, my := g.cellAt(ebiten.CursorPosition())
	if g.world.In(mx, my) {
		strokeRect(screen, g.ox+mx*cell, g.oy+my*cell, cell, cell, hoverColor)
	}

	ebitenutil.DebugPrint(screen, fmt.Sprintf(
		"1 water  2 dirt  3 grass  4 rock  [brush: %s]   LMB paint  RMB water  G overlay\n"+
			"white = world grid   yellow = display grid   numbers = masks of the %s layer",
		g.brush, g.overlayLayer().terrain))
}

// drawWaves draws a few moving highlights so the water reads as water. Real
// art would animate a water tile instead.
func (g *game) drawWaves(screen *ebiten.Image) {
	for y := 0; y < worldH; y++ {
		for x := 0; x < worldW; x++ {
			if g.world.At(x, y) != Water || (x*7+y*3)%4 != 0 {
				continue
			}
			shift := (g.ticks/20 + x + y) % 3 * 2
			fillRect(screen, g.ox+x*cell+8+shift, g.oy+y*cell+14, 10, 2, waveColor)
		}
	}
}

// overlayLayer is the layer whose masks the overlay shows: the one the brush
// paints, or dirt when painting water (water changes dirt's edges).
func (g *game) overlayLayer() layer {
	for _, l := range g.layers {
		if l.terrain == g.brush {
			return l
		}
	}
	return g.layers[0]
}

// drawOverlay shows both grids: the world grid where cells live, and the
// display grid, offset by half a cell, whose tiles each overlap 4 cells.
func (g *game) drawOverlay(screen *ebiten.Image) {
	w, h := worldW*cell, worldH*cell
	for x := 0; x <= worldW; x++ {
		fillRect(screen, g.ox+x*cell, g.oy, 1, h, worldLineColor)
	}
	for y := 0; y <= worldH; y++ {
		fillRect(screen, g.ox, g.oy+y*cell, w, 1, worldLineColor)
	}
	half := cell / 2
	for x := 0; x <= worldW+1; x++ {
		fillRect(screen, g.ox+x*cell-half, g.oy-half, 1, h+cell, dualLineColor)
	}
	for y := 0; y <= worldH+1; y++ {
		fillRect(screen, g.ox-half, g.oy+y*cell-half, w+cell, 1, dualLineColor)
	}
	// Each display tile's mask, printed at its center (a world-grid corner).
	src := AtLeast{&g.world, g.overlayLayer().terrain}
	for y := 0; y <= worldH; y++ {
		for x := 0; x <= worldW; x++ {
			m := dualgrid.MaskAt(src, x, y)
			if m == dualgrid.Empty || m == dualgrid.Full {
				continue
			}
			s := strconv.Itoa(int(m))
			ebitenutil.DebugPrintAt(screen, s, g.ox+x*cell-3*len(s), g.oy+y*cell-8)
		}
	}
}

func (g *game) Layout(int, int) (int, int) { return screenW, screenH }

var pixel = func() *ebiten.Image {
	img := ebiten.NewImage(1, 1)
	img.Fill(color.White)
	return img
}()

func fillRect(dst *ebiten.Image, x, y, w, h int, c color.Color) {
	var op ebiten.DrawImageOptions
	op.GeoM.Scale(float64(w), float64(h))
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(c)
	dst.DrawImage(pixel, &op)
}

func strokeRect(dst *ebiten.Image, x, y, w, h int, c color.Color) {
	fillRect(dst, x, y, w, 1, c)
	fillRect(dst, x, y+h-1, w, 1, c)
	fillRect(dst, x, y, 1, h, c)
	fillRect(dst, x+w-1, y, 1, h, c)
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// loadTiles loads a sheet from path, or generates tiles with gen when path
// is empty.
func loadTiles(path string, gen func(size int) *dualgrid.Tileset) (*dualgrid.Tileset, error) {
	if path == "" {
		return gen(tileSize), nil
	}
	sheet, _, err := ebitenutil.NewImageFromFile(path)
	if err != nil {
		return nil, err
	}
	return dualgrid.NewTileset(sheet, tileSize)
}

func main() {
	dirtPath := flag.String("dirt", "", "PNG sheet for dirt: 4x4 tiles of 16px (64x64)")
	grassPath := flag.String("grass", "", "PNG sheet for grass: 4x4 tiles of 16px (64x64)")
	rockPath := flag.String("rock", "", "PNG sheet for rock: 4x4 tiles of 16px (64x64)")
	ripplesPath := flag.String("ripples", "", "PNG sheet of animated shore ripples: frames of 4x4 16px tiles side by side (Nx64 wide, 64 high)")
	templateDir := flag.String("template", "", "write dirt.png, grass.png, rock.png and ripples.png placeholder sheets to this directory and exit")
	flag.Parse()

	if *templateDir != "" {
		sheets := map[string]image.Image{
			"dirt.png":    dirtLook.Sheet(tileSize),
			"grass.png":   grassLook.Sheet(tileSize),
			"rock.png":    rockLook.Sheet(tileSize),
			"ripples.png": dualgrid.DefaultRipples.Sheet(tileSize),
		}
		for name, img := range sheets {
			path := filepath.Join(*templateDir, name)
			if err := writePNG(path, img); err != nil {
				log.Fatal(err)
			}
			fmt.Println("wrote", path)
		}
		return
	}

	var tiles [4]*dualgrid.Tileset
	for i, src := range []struct {
		path string
		gen  func(int) *dualgrid.Tileset
	}{
		{*dirtPath, dirtLook.Tileset},
		{*grassPath, grassLook.Tileset},
		{*rockPath, rockLook.Tileset},
		{*ripplesPath, dualgrid.DefaultRipples.Tileset},
	} {
		ts, err := loadTiles(src.path, src.gen)
		if err != nil {
			log.Fatal(err)
		}
		tiles[i] = ts
	}

	ebiten.SetWindowSize(screenW, screenH)
	ebiten.SetWindowTitle("dualgrid example")
	if err := ebiten.RunGame(newGame(tiles[0], tiles[1], tiles[2], tiles[3])); err != nil {
		log.Fatal(err)
	}
}
