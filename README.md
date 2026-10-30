# dualgrid

A small [Ebitengine](https://ebitengine.org/) package for drawing tile maps with the **dual-grid** technique: 16 hand-drawn tiles cover every way a terrain can meet its neighbours.

```sh
go run ./example                                  # paint with generated placeholder tiles
go run ./example -dirt dirt.png -grass grass.png  # paint with your own art (any of -dirt -grass -rock -ripples)
go run ./example -template .                      # write dirt, grass, rock + ripples placeholder sheets as a starting point
go test ./...
```

In the example: **1/2/3/4** pick water/dirt/grass/rock, left mouse paints, right mouse paints water, **G** toggles the grid overlay.

## How it works

There are two grids.

- The **world grid** holds the data: every cell is grass or not. This is what you paint.
- The **display grid** is the same grid moved half a cell up and left. Tiles are drawn on *this* grid.

Because of that half-cell shift, every display tile sits on a **corner** of the world grid and covers a quarter of four world cells:

```
  four world cells (solid lines); one display tile (dotted) on their shared corner

 +-----------+-----------+
 |           |           |
 |    TL     |    TR     |
 |     :.....|.....:     |
 +-----:-----+-----:-----+
 |     :.....|.....:     |
 |    BL     |    BR     |
 |           |           |
 +-----------+-----------+
```

Each of the four corners is one bit, and together they make the tile's **mask**:

| corner | world cell  | bit |
|--------|-------------|-----|
| TL     | (x-1, y-1)  | 8   |
| TR     | (x,   y-1)  | 4   |
| BL     | (x-1, y)    | 2   |
| BR     | (x,   y)    | 1   |

`MaskAt(src, x, y)` computes it; the mask (0–15) is also the tile's number in the sheet. Tile 0 is empty and is not drawn; tile 15 is solid grass.

Why this is nice: with the classic approach (one tile per world cell, looking at its 8 neighbours) you need 47 tiles. Here, the boundary between two cells always runs straight through the middle of a display tile, so the tile only has to care about its 4 cells — 16 tiles, and many of them are rotations of each other.

Run the example and press **G**: white lines are the world grid, yellow lines the display grid, and each number is the mask of the tile centred there.

## Using it

```go
grass := dualgrid.NewGrid(24, 16)       // or anything with Filled(x, y int) bool
grass.Set(3, 4, true)

tiles, err := dualgrid.NewTileset(sheet, 16)  // sheet: a 64x64 *ebiten.Image

opts := &dualgrid.DrawOptions{MaxX: 23, MaxY: 15} // the world cells to cover
opts.GeoM.Scale(2, 2)                              // your camera
dualgrid.Draw(screen, grass, tiles, opts)
```

`Draw` works in *world pixels*: one cell is `Tileset.Size` pixels and cell (0,0) starts at the origin. `GeoM` maps that onto the screen.

## Several terrains: water, dirt, grass

Each cell still holds one terrain, but every terrain is drawn as its own **layer**, lowest first. Put them in order:

```
water  <  dirt  <  grass
```

- **Water** is the bottom, so it never has an edge. It's a flat fill (in a real game, an animated tile).
- **Dirt** and **grass** are dual-grid layers, each with its own 16-tile sheet.

Each layer's `Source` asks "is this cell my terrain **or anything above it**?":

```go
type AtLeast struct{ W *World; T Terrain }

func (a AtLeast) Filled(x, y int) bool { return a.W.At(x, y) >= a.T }

fillWater(screen)
dualgrid.Draw(screen, AtLeast{w, Dirt}, dirtTiles, opts)
dualgrid.Draw(screen, AtLeast{w, Grass}, grassTiles, opts)
```

Why "at least" and not "equals": a grass cell then also counts as dirt, so the dirt layer runs on *under* the grass. With "equals", both layers would pull back from a grass–dirt border and leave a gap of water between them.

What you get at each border:

| border         | drawn by                                   |
|----------------|--------------------------------------------|
| grass – dirt   | the grass tiles' edge, over dirt            |
| dirt – water   | the dirt tiles' edge (the shoreline), over water |
| grass – water  | both edges on the same line. Draw grass slightly further in and a thin strip of dirt shows: a beach |
| all three at one corner | nothing special: the layers stack |

So you draw one sheet per terrain that has an edge, with transparent "empty" parts. A new terrain is one more sheet, not a set of tiles for every pair of terrains. The example does exactly this; its placeholder grass uses `Inset: 0.12` to leave the beach.

Rock sits on top of grass as a fourth layer. Its placeholder (`dualgrid.Rock`) is *raised*: `Highlight` and `Shadow` colours add a light band inside edges facing up and a wide dark band inside edges facing down, so blobs read as boulders lit from above. A negative `Inset` lets rock spread a little past its cells, so a lone boulder fills its cell.

The other common approach is a transition sheet for every *pair* of terrains (grass–dirt, grass–water, dirt–water). That gives more control per border, but the number of sheets grows fast and corners where three terrains meet need extra work.

## Animation: shore ripples

A tileset can have **frames**: put several 4×4 blocks side by side, so an 8-frame sheet of 16px tiles is 512×64. `Tileset.Frames()` says how many there are, `Tileset.TileAt(mask, frame)` picks one (the frame wraps, so pass a running counter), and `DrawOptions.Frame` does the same for `Draw`.

Shore ripples use this. They are one more dual-grid layer, drawn **after the water and before the land**, with the *same* source as the land layer (in the example `AtLeast{w, Dirt}`). Each tile holds only the ripple lines on its water side and is transparent on the land side, which the land layer covers anyway. Tiles 0 (open water) and 15 (all land) are empty. Because the shore and the ripples come from the same mask, the ripples bend around every corner exactly like the coastline.

```go
opts.Frame = ticks / 12 // a new frame every 12 ticks (200ms)
dualgrid.Draw(screen, AtLeast{w, Dirt}, ripples, opts)
opts.Frame = 0
// ... then the dirt, grass and rock layers
```

`dualgrid.DefaultRipples` generates a sheet: two faint, blurred light bands at a time that start at the shore, drift out about a third of a cell and fade, over 16 frames. `Blur` sets how soft they are (0 is a crisp line), `Color.A` how strong, and `Frames` how smooth; how *slow* is up to how long you show each frame. `go run ./example -template .` writes it as `ripples.png`. To draw your own, keep the frames' tile edges matching (as for any sheet), leave land and tiles 0 and 15 transparent, and make the last frame lead back into the first so the loop is seamless.

## Drawing guide

### The sheet

One PNG of **64×64** pixels: 4×4 tiles of **16×16**. Tile number = mask, read left to right, top to bottom. In the pictures below `#` is grass and `.` is the ground underneath; each picture is one tile split into its four quarters (TL TR / BL BR).

```
  0        1        2        3
 . .      . .      . .      . .
 . .      . #      # .      # #
 empty    corner   corner   edge (bottom)

  4        5        6        7
 . #      . #      . #      . #
 . .      . #      # .      # #
 corner   edge     diagonal inner corner
          (right)

  8        9        10       11
 # .      # .      # .      # .
 . .      . #      # .      # #
 corner   diagonal edge     inner corner
                   (left)

  12       13       14       15
 # #      # #      # #      # #
 . .      . #      # .      # #
 edge     inner    inner    full
 (top)    corner   corner
```

`go run ./example -template .` writes `dirt.png`, `grass.png` and `rock.png` in exactly this layout — open them in your editor and paint over them.

### Order to draw them

1. **15 (full)** — your plain grass. Make sure it tiles with itself.
2. **0 (empty)** — usually leave it transparent, so the layer below shows.
3. **Edges: 3, 12, 5, 10** — a straight border through the middle of the tile.
4. **Outer corners: 1, 2, 4, 8** — a small blob of grass in one corner.
5. **Inner corners: 7, 11, 13, 14** — grass everywhere except one corner.
6. **Diagonals: 6, 9** — two opposite corners. Decide whether they connect through the middle or stay two blobs. For two blobs, don't draw them at all: copy your corner tiles on top of each other (6 = 4 + 2, 9 = 8 + 1). Then they always match your corners. Avoid filling the two quarters as solid squares: next to your rounded corners that reads as a hard, square pinch.

### Rules for seamless tiles

- **The border lives in the middle.** A world cell's edge lies on the tile's centre line (pixel 8 of 16). The tile's own outer edges are always in the *middle* of a world cell, so they are either fully grass or fully ground — the only exception is where a border crosses them.
- **Borders must cross tile edges at the same spot.** Where a grass border leaves a tile (say at x=8 on its bottom edge), it must enter the tile below at x=8 on its top edge, with the same thickness and colour. Pick fixed crossing points and reuse them in every tile. Edge tiles are the reference: the corners must meet them.
- **Copy, then vary.** Draw one edge, then rotate/flip it for the other three; same for corners. Small variations inside the tile are fine as long as the edges match.
- **Leave empty parts transparent.** The layer below has to show through.
- **Check it live.** Run `go run ./example -grass yours.png`, paint shapes that use every tile (the overlay numbers show which is which), and look for steps at the seams.

Tools: [Aseprite](https://www.aseprite.org/), [LibreSprite](https://libresprite.github.io/) or [Piskel](https://www.piskelapp.com/) (free, in the browser). Set a 16×16 grid so each tile is one grid square.

### Possible next steps

- Since most tiles are rotations of others, a tileset could be built from just 6 drawn tiles (empty, full, edge, outer corner, inner corner, diagonal) by rotating them in code. Not implemented here — drawing all 16 gives you more control (light from the top-left, shadows on south edges).
- Variants of tile 15 picked by a hash of the position, to break up repetition.
