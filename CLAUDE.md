# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A Go library (`github.com/venusliang/go-font`, package `gofont`) for parsing, editing, and writing TrueType font (.ttf) and OpenType/CFF font (.otf) files. Supports WOFF, WOFF2, EOT, and TTC format parsing and serialization. It reads binary font data, deserializes all standard TrueType table structures, and can serialize modified fonts back to valid .ttf, .woff, .woff2, .eot, or .ttc files.

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

The entry point is `Parse(data []byte)` in `ttf.go`, which returns a `TrueTypeFont` struct containing all parsed font tables. `Parse()` delegates to `parseFromOffset(data, offset)` which supports parsing a TTF at an arbitrary offset within a larger file (used by TTC). `Serialize() ([]byte, error)` in `serialize.go` writes a complete .ttf file from the struct.

The `TrueTypeFont.version` field distinguishes font types:
- `0x00010000` or `0x74727565` ("true"): TrueType outlines (glyf/loca tables)
- `0x4F54544F` ("OTTO"): CFF outlines (CFF table, no glyf/loca)

### Binary I/O Layer

- `binary.go` defines a `Binary` interface with `BigEndian` and `LittleEndian` implementations for sequential byte-level reading/writing. All TTF data is big-endian: `BinaryFrom(data, false)`.
- `fixed.go` defines fixed-point types (`Fixed16_16`, `Fixed2_14`) used in font metrics.

### Table Parsers

Each TrueType table has its own file with a `parseXxx(data []byte)` function and a corresponding `writeXxx(xxx *Xxx) []byte` function:

| File | Table | Notes |
|------|-------|-------|
| `head.go` | `head` | Font header, bounding box, indexToLocFormat |
| `name.go` | `name` | Font name strings, format 0 and 1 |
| `maxp.go` | `maxp` | Maximum profile, glyph count (TrueType: 32 bytes v1; CFF: 6 bytes v0.5) |
| `os_2..go` | `OS/2` | OS/2 metrics, versioned parsing/writing (v0-v5) |
| `cmap.go` | `cmap` | Character-to-glyph mapping, subtable formats 0, 4, 6, 12 with `Map(rune) uint16` |
| `table.go` | directory | `TableDirectory` struct and checksum calculation |
| `hhea.go` | `hhea` | Horizontal header, numberOfHMetrics |
| `hmtx.go` | `hmtx` | Horizontal metrics (advance width + LSB per glyph) |
| `loca.go` | `loca` | Glyph index to offset mapping (short/long format) |
| `glyf.go` | `glyf` | Glyph outlines (simple + composite), flag RLE encoding |
| `post.go` | `post` | PostScript name mapping, format 2.0 |
| `kern.go` | `kern` | Kerning table, format 0 |
| `gpos.go` | `GPOS` | Glyph positioning, single substitution / pair positioning |
| `gsub.go` | `GSUB` | Glyph substitution, single substitution |
| `cff.go` | `CFF ` | CFF (Compact Font Format) structure: Header, INDEX, Top/Private DICT, Charset, CharStrings |
| `cff_charstring.go` | — | Type 2 CharString bytecode interpreter, outline extraction (moveto/lineto/curveto) |
| `face.go` | — | `font.Face` implementation for text rendering via `golang.org/x/image/font` |

### Parse Order

Independent tables (head, OS/2, cmap, maxp, name, hhea, post) are parsed in the table directory loop. Dependent tables are parsed after the loop:
- `hmtx` needs `hhea.numberOfHMetrics` + `maxp.numGlyphs`
- `loca` needs `head.indexToLocFormat` + `maxp.numGlyphs`
- `glyf` needs `loca` for glyph boundaries

For CFF/OpenType fonts (version "OTTO"), the CFF table is parsed from `rawTables["CFF "]` after the directory loop. CFF parsing in `cff.go` follows the CFF spec structure: Header → Name INDEX → Top DICT INDEX → String INDEX → Global Subr INDEX → Charset → CharStrings INDEX → Private DICT → Local Subr INDEX.

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
- **CFF**: `IsCFF`, `CFFFontName`, `CFFGlyphName`, `CFFOutlineAt`, `CFFOutlineForRune`
- **Rendering**: `NewFace` on `TrueTypeFont` returns a `*Face` implementing `golang.org/x/image/font.Face` for text rendering with `font.Drawer`

The abstract cmap layer uses `map[rune]uint16` (lazily initialized from parsed cmap via `Enumerate`). When `Serialize()` is called and the map is non-nil, `rebuildCmap()` regenerates the binary cmap from the abstract map.

### Font Format Support

Each font format has its own file with `ParseXxx()` and `SerializeXxx()` methods:

| File | Format | Parse | Serialize | Notes |
|------|--------|-------|-----------|-------|
| `woff.go` | WOFF | `ParseWOFF()` | `SerializeWOFF()` | zlib per-table compression, uses shared `rebuildTTF()` |
| `woff2.go` | WOFF2 | `ParseWOFF2()` | `SerializeWOFF2()` | Brotli single-stream compression, glyf/loca & hmtx transform support, uses shared `rebuildTTF()` |
| `eot.go` | EOT | `ParseEOT()` | `SerializeEOT()` | Little-endian header, XOR 0x50 decryption, no compression (MTX not supported) |
| `ttc.go` | TTC | `ParseTTC()` | `SerializeTTC()` | TrueType Collection, multiple fonts in one file, uses `parseFromOffset()` |

All formats parse to the same `TrueTypeFont` struct and all editing APIs work regardless of source format. `rebuildTTF()` in `woff2.go` is shared by WOFF and WOFF2 for reconstructing a TTF byte stream from decompressed table entries.

TTC (TrueType Collection) is a container format that bundles multiple fonts in one file. `ParseTTC()` returns `[]TrueTypeFont`. Each font's table offsets are relative to the TTC file start. `SerializeTTC(fonts)` serializes each font independently then adjusts table directory offsets to be TTC-relative. `TestFormatCrossRoundTrip` in `ttc_test.go` verifies the full chain: TTC → TTF → WOFF → WOFF2 → EOT → TTF.

### Key Implementation Details

- `writeGlyf` and `writeLoca` are coupled: `writeGlyf` returns both the glyf data and the loca offsets, since glyph sizes affect offsets. Glyph data is padded to even boundaries for short loca format compatibility.
- `calcTableChecksum` in `table.go` pads data to 4-byte boundaries. The `head` table's `checksumAdjustment` field must be zeroed before checksum calculation.
- `writeCmap` handles duplicate subtable offsets (multiple encoding records pointing to the same subtable data).
- `TrueTypeFont.Serialize()` sorts tables alphabetically by tag, pads to 4-byte alignment, and patches `head.checksumAdjustment = 0xB1B0AFBA - wholeFileChecksum`.
- `rawTables` in `TrueTypeFont` stores raw bytes for tables not natively parsed (e.g. `CFF `). These are written back as-is during serialization for lossless round-trip.
- `NumGlyphs()` uses `len(glyf)` for TrueType fonts and `maxp.numGlyphs` for CFF fonts.
- CFF charstring outlines are decoded lazily via `DecodeOutlines()` using a Type 2 VM with a 48-entry operand stack, subroutine support (local + global), and 10-level call depth limit.

### Font Rendering (face.go)

`Face` implements `golang.org/x/image/font.Face` for rendering parsed fonts as images. `NewFace(f *TrueTypeFont, opts *FaceOptions)` (package-level function or method on `TrueTypeFont`) creates a `Face` at a specific size/DPI. The Face uses `golang.org/x/image/vector.Rasterizer` for anti-aliased rasterization.

- **Scale**: `scale = fixed.Int26_6(0.5 + size * dpi * 64 / 72)` converts design units to 26.6 fixed-point pixels.
- **Y-flip**: OpenType Y increases upward; image Y increases downward. Coordinates are negated during outline extraction.
- **TrueType outlines**: Quadratic Bézier curves. Consecutive off-curve points produce an implicit on-curve midpoint. Composite glyphs recurse (max depth 8) with translation + optional 2x2 transform.
- **CFF outlines**: Cubic Bézier curves. `DecodeOutlines()` is called once in `NewFace` and cached in `Face.cffOutlines`.
- **Kerning**: Binary search in kern table format-0 subtables.
- **Reusable buffers**: `Face` reuses `[]faceSegment` slice, `vector.Rasterizer`, and `image.Alpha` mask across Glyph calls. Not safe for concurrent use.

## Git Rules

- Do NOT use `Co-Authored-By: Claude` or any default Claude user in commit messages. Only use the configured git account for all commits and pushes.
