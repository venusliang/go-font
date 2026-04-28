# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A Go library (`github.com/venusliang/go-font`, package `gofont`) for parsing, editing, and writing TrueType font (.ttf) files. It reads binary font data, deserializes all standard TrueType table structures, and can serialize modified fonts back to valid .ttf files.

## Commands

```bash
# Run all tests
go test ./...

# Run a specific test
go test -run TestParse ./...

# Run with verbose output
go test -v ./...
```

There is no linting, CI configuration, or Makefile in this project.

## Architecture

The entry point is `Parse(data []byte)` in `ttf.go`, which returns a `TrueTypeFont` struct containing all parsed font tables. `Serialize() ([]byte, error)` in `serialize.go` writes a complete .ttf file from the struct.

### Binary I/O Layer

- `binary.go` defines a `Binary` interface with `BigEndian` and `LittleEndian` implementations for sequential byte-level reading/writing. All TTF data is big-endian: `BinaryFrom(data, false)`.
- `fixed.go` defines fixed-point types (`Fixed16_16`, `Fixed2_14`) used in font metrics.

### Table Parsers

Each TrueType table has its own file with a `parseXxx(data []byte)` function and a corresponding `writeXxx(xxx *Xxx) []byte` function:

| File | Table | Notes |
|------|-------|-------|
| `head.go` | `head` | Font header, bounding box, indexToLocFormat |
| `name.go` | `name` | Font name strings, format 0 and 1 |
| `maxp.go` | `maxp` | Maximum profile, glyph count (version is Fixed 16.16, read as U32) |
| `os_2..go` | `OS/2` | OS/2 metrics, versioned parsing/writing (v0-v5) |
| `cmap.go` | `cmap` | Character-to-glyph mapping, subtable formats 0, 4, 6, 12 with `Map(rune) uint16` |
| `table.go` | directory | `TableDirectory` struct and checksum calculation |
| `hhea.go` | `hhea` | Horizontal header, numberOfHMetrics |
| `hmtx.go` | `hmtx` | Horizontal metrics (advance width + LSB per glyph) |
| `loca.go` | `loca` | Glyph index to offset mapping (short/long format) |
| `glyf.go` | `glyf` | Glyph outlines (simple + composite), flag RLE encoding |
| `post.go` | `post` | PostScript name mapping, format 2.0 |

### Parse Order

Independent tables (head, OS/2, cmap, maxp, name, hhea, post) are parsed in the table directory loop. Dependent tables are parsed after the loop:
- `hmtx` needs `hhea.numberOfHMetrics` + `maxp.numGlyphs`
- `loca` needs `head.indexToLocFormat` + `maxp.numGlyphs`
- `glyf` needs `loca` for glyph boundaries

### Test Pattern

Each table has a `_test.go` file with `TestParseXxx` (value assertions) and `TestRoundTripXxx` (write then re-parse and compare). Test data is loaded from `fonts/fonteditor.ttf` via `loadFont(t)` in `testing_test.go` (reads from disk, cached across tests).

### Glyph Editing API

`edit.go` provides all manipulation methods on `TrueTypeFont`:
- **Rune mapping**: `RuneToGlyphID`, `GlyphForRune`, `SetRuneMapping`, `RemoveRuneMapping`, `SetRuneMappings`, `RuneMappings`, `MappedRunes`
- **Glyph access**: `NumGlyphs`, `GlyphAt`, `SetGlyphAt`, `AppendGlyph`, `CopyGlyph`
- **Glyph removal**: `RemoveGlyphs` compacts glyphs, hmtx, updates maxp/hhea, remaps composite references, updates rune mappings
- **Glyph transform**: `TranslateGlyph`, `ScaleGlyph`
- **Glyph query**: `IsSimpleGlyph`, `IsCompositeGlyph`, `GlyphBBox`, `PointCount`, `ContourCount`
- **Font metrics**: `UnitsPerEm`, `FontBBox`, `Ascent`, `Descent`, `AdvanceWidth`, `AdvanceWidthForRune`, `LeftSideBearing`, `SetAdvanceWidth`, `SetLeftSideBearing`
- **Font names**: `FontFamily`, `FontFullName`, `SetFontFamily`, `SetFontFullName`
- **Subset**: `Subset(keepRunes)` keeps only glyphs needed for specified characters

The abstract cmap layer uses `map[rune]uint16` (lazily initialized from parsed cmap via `Enumerate`). When `Serialize()` is called and the map is non-nil, `rebuildCmap()` regenerates the binary cmap from the abstract map.

### Key Implementation Details

- `writeGlyf` and `writeLoca` are coupled: `writeGlyf` returns both the glyf data and the loca offsets, since glyph sizes affect offsets. Glyph data is padded to even boundaries for short loca format compatibility.
- `calcTableChecksum` in `table.go` pads data to 4-byte boundaries. The `head` table's `checksumAdjustment` field must be zeroed before checksum calculation.
- `writeCmap` handles duplicate subtable offsets (multiple encoding records pointing to the same subtable data).
- `TrueTypeFont.Serialize()` sorts tables alphabetically by tag, pads to 4-byte alignment, and patches `head.checksumAdjustment = 0xB1B0AFBA - wholeFileChecksum`.
