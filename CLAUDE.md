# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```sh
go test ./...                                     # all tests
go test -run TestMaskAtEveryMask .                # single test
go vet ./...
go run ./example                                  # interactive demo with generated placeholder tiles
go run ./example -dirt assets/dirt.png -grass assets/grass.png -rock assets/rock.png  # demo with real art
go run ./example -template <dir>                  # write dirt, grass, rock + ripples placeholder sheets and exit
```

Demo controls: **1/2/3/4** pick water/dirt/grass/rock, left mouse paints, right mouse paints water, **G** toggles the overlay (white = world grid, yellow = display grid, numbers = tile masks).

Module: `github.com/dwethmar/dualgrid`, Go 1.27, single dependency Ebitengine v2.

## Architecture

A small library package (root) plus a demo (`example/main.go`). The README is the design doc and drawing guide; keep it in sync when behavior changes.

**Dual-grid core.** Data lives on the *world grid*; tiles are drawn on a *display grid* shifted half a tile up-left, so display tile `(x, y)` sits on the corner shared by world cells `(x-1,y-1)`, `(x,y-1)`, `(x-1,y)`, `(x,y)`. Those four cells form a 4-bit `Mask` (`TL=8, TR=4, BL=2, BR=1`), which is also the tile's index in the sheet. 16 tiles cover every case.

- `mask.go` — `Source` interface (`Filled(x, y) bool`; out-of-bounds must return false), `Mask`, `MaskAt`.
- `grid.go` — `Grid`, a simple bool-grid `Source`.
- `tileset.go` — `Tileset` slices a 4×4 sheet (tile `i` at column `i%4`, row `i/4`; sheet must be exactly `4*size` square). `Placeholder` generates sheets procedurally via bilinear interpolation of the four corner values; `Inset` pulls the edge inward (negative pushes it out); optional `Highlight`/`Shadow` shade up- and down-facing edges for a raised look (`Rock`). Shading uses only the fill amount and its vertical slope, and skips a tile's top and bottom pixel rows, because those are the only quantities neighbouring tiles share. Tile outer edges depend only on cells shared with neighbours, which keeps tiles seamless — preserve this property when changing generation.
- `ripples.go` — `Ripples` generates an animated sheet of shore ripples from the same fill-amount field (`fillAt`), so ripple bands match across tile edges in every frame. Sheets may hold several frames side by side (`Tileset.Frames`/`TileAt`, `DrawOptions.Frame`).
- `draw.go` — `Draw` renders one layer in *world pixels* (one cell = `Tileset.Size` px), iterating display tiles `MinX..MaxX+1` / `MinY..MaxY+1` so half tiles at the edges are covered; `opts.GeoM` is the camera. `Empty` tiles are skipped so layers stack.

**Multiple terrains (demo, not library).** Each cell holds one `Terrain` ordered `Water < Dirt < Grass < Rock`. Water is a flat fill; each higher terrain is a separate dual-grid layer drawn lowest first, with source `AtLeast{W, T}` (`At(x,y) >= T`), not equality — otherwise layers would both retreat from a shared border and leave a gap. Grass uses a larger `Inset` than dirt so a dirt "beach" shows where grass meets water.

## Assets

`assets/dirt.png`, `assets/grass.png`: hand-drawn 64×64 sheets (16px tiles) in mask order. `assets/rock.png`, `assets/ripples.png`: the generated `Rock` and `DefaultRipples` placeholders, to be repainted. Apostle embeds copies of all three. `sheet.png` at the root is a sheet image not referenced by code.
