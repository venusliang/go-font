# go-font

[中文文档](README_zh.md)

A Go library for parsing, editing, and serializing TrueType (.ttf) and OpenType/CFF (.otf) font files, with support for WOFF, WOFF2, EOT, and TTC formats. It also implements `golang.org/x/image/font.Face` for rendering text directly onto images.

## Installation

```bash
go get github.com/venusliang/go-font
```

## Quick Start

```go
package main

import (
    "fmt"
    "os"

    gofont "github.com/venusliang/go-font"
)

func main() {
    // Read a TTF font file
    data, _ := os.ReadFile("myfont.ttf")

    // Parse (also supports .otf files, auto-detected)
    ttf, err := gofont.Parse(data)
    if err != nil {
        panic(err)
    }

    // Inspect font info
    fmt.Printf("Glyph count: %d\n", ttf.NumGlyphs())

    // Serialize back to TTF
    out, _ := ttf.Serialize()
    os.WriteFile("output.ttf", out, 0644)
}
```

## API Overview

### Parsing & Serialization

| Method | Description |
|--------|-------------|
| `Parse(data []byte) (TrueTypeFont, error)` | Parse TTF/OTF binary data (auto-detects TrueType vs OpenType/CFF) |
| `ParseWOFF(data []byte) (TrueTypeFont, error)` | Parse WOFF binary data |
| `ParseWOFF2(data []byte) (TrueTypeFont, error)` | Parse WOFF2 binary data |
| `ParseEOT(data []byte) (TrueTypeFont, error)` | Parse EOT binary data |
| `ParseTTC(data []byte) ([]TrueTypeFont, error)` | Parse TTC binary data, returns a list of font objects |
| `ttf.Serialize() ([]byte, error)` | Serialize to TTF binary data |
| `ttf.SerializeWOFF() ([]byte, error)` | Serialize to WOFF format |
| `ttf.SerializeWOFF2() ([]byte, error)` | Serialize to WOFF2 format |
| `ttf.SerializeEOT() ([]byte, error)` | Serialize to EOT format |
| `SerializeTTC(fonts []TrueTypeFont) ([]byte, error)` | Serialize multiple fonts into TTC format |

### Unicode Mapping

| Method | Description |
|--------|-------------|
| `RuneToGlyphID(r rune) uint16` | Map a Unicode code point to a glyph ID; returns 0 if unmapped |
| `GlyphForRune(r rune) *Glyph` | Get glyph data for a Unicode code point; returns nil if unmapped |
| `SetRuneMapping(r rune, glyphID uint16) error` | Set a mapping from code point to glyph ID |
| `RemoveRuneMapping(r rune)` | Remove the mapping for a code point |
| `SetRuneMappings(m map[rune]uint16) error` | Set multiple mappings at once |
| `RuneMappings() []struct{Rune; GlyphID}` | Return all mappings, sorted by code point |
| `MappedRunes() []rune` | Return all mapped code points |

### Glyph Operations

| Method | Description |
|--------|-------------|
| `NumGlyphs() int` | Total number of glyphs |
| `GlyphAt(index int) *Glyph` | Get glyph by index; returns nil if out of range |
| `SetGlyphAt(index int, g *Glyph) error` | Replace glyph data at index |
| `AppendGlyph(g *Glyph) (int, error)` | Append a new glyph, returns its index |
| `CopyGlyph(src, dst int) error` | Copy glyph data |
| `RemoveGlyphs(indices []int) (remap, error)` | Remove glyphs at given indices and compact related tables |
| `TranslateGlyph(index, dx, dy int16) error` | Translate glyph coordinates |
| `ScaleGlyph(index int, sx, sy float64) error` | Scale glyph coordinates |

### Glyph Properties

| Method | Description |
|--------|-------------|
| `IsSimpleGlyph(index int) bool` | Whether the glyph is a simple glyph |
| `IsCompositeGlyph(index int) bool` | Whether the glyph is a composite glyph |
| `GlyphBBox(index) (xMin, yMin, xMax, yMax, ok)` | Glyph bounding box |
| `PointCount(index int) int` | Number of points in the glyph |
| `ContourCount(index int) int` | Number of contours in the glyph |

### Font Metrics

| Method | Description |
|--------|-------------|
| `UnitsPerEm() uint16` | Design units per em |
| `FontBBox() (xMin, yMin, xMax, yMax)` | Global font bounding box |
| `Ascent() int16` | Typographic ascent |
| `Descent() int16` | Typographic descent |
| `AdvanceWidth(glyphID uint16) uint16` | Advance width for a glyph |
| `AdvanceWidthForRune(r rune) uint16` | Advance width for a Unicode code point |
| `LeftSideBearing(glyphID uint16) int16` | Left side bearing |
| `SetAdvanceWidth(glyphID, width uint16) error` | Set advance width |
| `SetLeftSideBearing(glyphID uint16, lsb int16) error` | Set left side bearing |

### Font Names

| Method | Description |
|--------|-------------|
| `FontFamily() string` | Get font family name |
| `FontFullName() string` | Get full font name |
| `SetFontFamily(name)` | Set font family name |
| `SetFontFullName(name)` | Set full font name |

### CFF (OpenType/CFF) Support

These methods apply to CFF outline fonts (`.otf` files). For TrueType outline fonts they return zero values or nil.

| Method | Description |
|--------|-------------|
| `IsCFF() bool` | Whether the font uses CFF outlines (OpenType/CFF) |
| `CFFFontName() string` | CFF font name; empty string for non-CFF fonts |
| `CFFGlyphName(glyphID int) string` | CFF glyph name; empty string for non-CFF fonts |
| `CFFOutlineAt(index int) *CFFOutline` | CFF outline by glyph index; nil if out of range or non-CFF |
| `CFFOutlineForRune(r rune) *CFFOutline` | CFF outline by Unicode code point |

### Advanced Operations

| Method | Description |
|--------|-------------|
| `Subset(keepRunes []rune) error` | Subset the font, keeping only glyphs needed for the specified characters |

### Text Rendering (font.Face)

Text rendering is provided by the `draw` subpackage:

```go
import fontdraw "github.com/venusliang/go-font/draw"
```

| Method | Description |
|--------|-------------|
| `fontdraw.NewFace(ttf, opts) *Face` | Create a rendering Face at the specified size |
| `face.Close() error` | Release resources (currently a no-op) |
| `face.Metrics() font.Metrics` | Return font metrics (Ascent, Descent, Height, etc.) |
| `face.GlyphAdvance(r rune) (fixed.Int26_6, bool)` | Return glyph advance width |
| `face.GlyphBounds(r rune) (fixed.Rectangle26_6, fixed.Int26_6, bool)` | Return glyph bounding box |
| `face.Kern(r0, r1 rune) fixed.Int26_6` | Return kerning value between two runes |
| `face.Glyph(dot, r) (dr, mask, maskp, advance, ok)` | Rasterize a glyph, returning an alpha mask |

Supports both TrueType outlines (quadratic Bezier curves) and CFF/OpenType outlines (cubic Bezier curves), using `golang.org/x/image/vector.Rasterizer` for anti-aliased rendering.

See the [draw subpackage README](draw/README.md) for detailed API documentation and usage examples.

### SVG Export

Glyph outline export as standalone SVG documents is provided by the `svg` subpackage:

```go
import svgexp "github.com/venusliang/go-font/svg"
```

| Function | Description |
|----------|-------------|
| `svgexp.Glyph(ttf, glyphIndex, opts) (string, error)` | Export a glyph by index as a standalone SVG document |
| `svgexp.GlyphForRune(ttf, r, opts) (string, error)` | Export a glyph by Unicode code point as an SVG document |

Supports both TrueType outlines (quadratic Bezier curves) and CFF/OpenType outlines (cubic Bezier curves), using the unified `GlyphPath` API.

See the [svg subpackage README](svg/README.md) for detailed API documentation and usage examples.

## Multi-Format Support

All formats parse to the same `TrueTypeFont` struct and all editing APIs work regardless of source format.

### WOFF (Web Open Font Format)

```go
data, _ := os.ReadFile("myfont.woff")
ttf, _ := gofont.ParseWOFF(data)

// Edit...
ttf.SetFontFamily("NewName")

// Serialize back to WOFF
woffOut, _ := ttf.SerializeWOFF()

// Or convert to TTF / WOFF2
ttfOut, _ := ttf.Serialize()
woff2Out, _ := ttf.SerializeWOFF2()
```

- zlib per-table compression
- Parsed result is equivalent to standard TTF; all editing APIs work
- Each table is compressed independently during serialization

### WOFF2 (Web Open Font Format 2)

```go
data, _ := os.ReadFile("myfont.woff2")
ttf, _ := gofont.ParseWOFF2(data)

// Serialize to WOFF2
woff2Out, _ := ttf.SerializeWOFF2()
```

- Brotli single-stream compression (all tables combined)
- Automatically handles glyf/loca and hmtx transform reversal
- No table transforms on serialization (transform version 3) for broad compatibility
- Depends on `github.com/andybalholm/brotli` (pure Go, no CGO)

### EOT (Embedded OpenType)

```go
data, _ := os.ReadFile("myfont.eot")
ttf, _ := gofont.ParseEOT(data)

// Serialize to EOT
eotOut, _ := ttf.SerializeEOT()
```

- Microsoft font embedding format, primarily for legacy IE
- Supports versions 0x00010000 / 0x00020001 / 0x00020002
- Auto XOR 0x50 decryption
- EOT metadata (PANOSE, Weight, UnicodeRange, etc.) is auto-populated from OS/2, head, and name tables
- Name fields use UTF-16LE encoding

### TTC (TrueType Collection)

```go
data, _ := os.ReadFile("myfont.ttc")
fonts, _ := gofont.ParseTTC(data)

// Edit the first font
fonts[0].SetFontFamily("NewName")

// Serialize a single font to TTF
ttfOut, _ := fonts[0].Serialize()

// Or re-pack all fonts as TTC
ttcOut, _ := gofont.SerializeTTC(fonts)
```

- Multi-font container format (e.g. system CJK fonts like NotoSansCJK.ttc)
- `ParseTTC()` returns `[]TrueTypeFont`; each font can be edited and serialized independently
- Supports TTC versions 1.0 and 2.0
- Table offsets are automatically adjusted for TTC-relative positioning during serialization

### OTF (OpenType/CFF)

```go
data, _ := os.ReadFile("myfont.otf")
ttf, _ := gofont.Parse(data)

if ttf.IsCFF() {
    fmt.Printf("CFF font name: %s\n", ttf.CFFFontName())

    // Get glyph outline
    outline := ttf.CFFOutlineAt(0)
    if outline != nil {
        fmt.Printf("Segments: %d\n", outline.NumSegments())
        for _, seg := range outline.Segments() {
            fmt.Printf("  %v: %v\n", seg.Op, seg.Args)
        }
    }

    // Look up outline by Unicode code point
    outline = ttf.CFFOutlineForRune('A')
}

// Serialize (CFF raw table data is preserved)
out, _ := ttf.Serialize()
os.WriteFile("output.otf", out, 0644)
```

- `Parse()` auto-detects OTF files
- Parses CFF Header, Name INDEX, Top DICT, String INDEX, Charset, CharStrings INDEX, Private DICT
- Supports Type 2 CharString outline decoding (moveto/lineto/curveto opcodes, with subroutine calls)
- CFF raw table data is preserved during serialization for lossless round-trip

### Format Limitations

| Format | Limitation |
|--------|-----------|
| OTF | CFF outlines are read-only and round-trip preserved; modifying CFF CharString data and re-encoding is not supported |
| WOFF | None |
| WOFF2 | No glyf/loca/hmtx table transforms during serialization; compression ratio is slightly lower than official tools |
| EOT | MTX (MicroType Express) compression is not supported; returns an error when encountered |
| EOT | Only TrueType outlines (glyf table) are supported; CFF outlines are not supported |
| TTC | Table sharing is not supported (each font is packed independently, shared tables are not merged) |

### Format Conversion

```go
// OTF -> TTF (CFF table preserved as-is)
ttf, _ := gofont.Parse(otfData)
ttfOut, _ := ttf.Serialize()

// WOFF2 -> TTF
ttf, _ := gofont.ParseWOFF2(woff2Data)
ttfOut, _ := ttf.Serialize()

// TTF -> EOT
ttf, _ := gofont.Parse(ttfData)
eotOut, _ := ttf.SerializeEOT()

// EOT -> WOFF
ttf, _ := gofont.ParseEOT(eotData)
woffOut, _ := ttf.SerializeWOFF()

// Extract first font from TTC -> WOFF2
fonts, _ := gofont.ParseTTC(ttcData)
woff2Out, _ := fonts[0].SerializeWOFF2()

// Pack multiple fonts into TTC
ttcOut, _ := gofont.SerializeTTC(fonts)
```

## Examples

### Remap Unicode

Remap the glyph at code point 0x91 to code point 0xFB:

```go
ttf, _ := gofont.Parse(data)

gid := ttf.RuneToGlyphID(0x91)   // get glyph ID
ttf.RemoveRuneMapping(0x91)       // remove old mapping
ttf.SetRuneMapping(0xFB, gid)     // create new mapping

out, _ := ttf.Serialize()
```

### Query Glyph Info

```go
ttf, _ := gofont.Parse(data)

// Look up glyph by code point
g := ttf.GlyphForRune('A')
if g != nil {
    fmt.Printf("xMin=%d, yMin=%d, xMax=%d, yMax=%d\n",
        g.header.xMin, g.header.yMin,
        g.header.xMax, g.header.yMax)
}

// Enumerate all mappings
for _, m := range ttf.RuneMappings() {
    fmt.Printf("U+%04X -> glyph %d\n", m.Rune, m.GlyphID)
}
```

### Trim Glyphs

Remove unneeded glyphs to reduce font file size:

```go
ttf, _ := gofont.Parse(data)

// Remove glyphs at indices 3, 7, 10
remap, err := ttf.RemoveGlyphs([]int{3, 7, 10})
if err != nil {
    panic(err)
}

// remap tracks old-index -> new-index
// e.g. remap[4] == 3 means old index 4 became new index 3
// Deleted indices do not appear in remap

fmt.Printf("Glyphs after removal: %d\n", ttf.NumGlyphs())

out, _ := ttf.Serialize()
```

`RemoveGlyphs` automatically updates:

- `glyf` -- removes corresponding glyphs, packs remaining
- `loca` -- recalculates offsets
- `hmtx` -- removes corresponding horizontal metrics
- `maxp` -- updates glyph count and statistics
- `hhea` -- updates numberOfHMetrics
- Composite glyphs -- remaps component references
- cmap -- updates Unicode-to-glyph-ID mappings

> Note: Glyph 0 (.notdef) cannot be removed; it is a required default glyph.

### Replace Glyph Data

```go
ttf, _ := gofont.Parse(data)

// Replace glyph at index 1 with a copy of glyph at index 0
src := ttf.GlyphAt(0)
if src != nil {
    newGlyph := &gofont.Glyph{
        header: src.header,
    }
    if src.simpleGlyph != nil {
        sg := *src.simpleGlyph
        newGlyph.simpleGlyph = &sg
    }

    ttf.SetGlyphAt(1, newGlyph)
}

out, _ := ttf.Serialize()
```

### Add Unicode Mappings

```go
ttf, _ := gofont.Parse(data)

// Map 'A' (U+0041) to existing glyph 1
err := ttf.SetRuneMapping('A', 1)
if err != nil {
    panic(err)
}

// Map multiple characters
chars := []rune{'A', 'B', 'C'}
for i, ch := range chars {
    ttf.SetRuneMapping(ch, uint16(i+1))
}

out, _ := ttf.Serialize()
```

### Read Font Info

```go
ttf, _ := gofont.Parse(data)

fmt.Printf("Family: %s\n", ttf.FontFamily())
fmt.Printf("Full name: %s\n", ttf.FontFullName())
fmt.Printf("Units/Em: %d\n", ttf.UnitsPerEm())
fmt.Printf("Ascent: %d, Descent: %d\n", ttf.Ascent(), ttf.Descent())

// Query glyph metrics and properties
w := ttf.AdvanceWidth(1)
lsb := ttf.LeftSideBearing(1)
xMin, yMin, xMax, yMax, _ := ttf.GlyphBBox(1)
pts := ttf.PointCount(1)
fmt.Printf("Glyph 1: width=%d lsb=%d bbox=(%d,%d,%d,%d) points=%d\n",
    w, lsb, xMin, yMin, xMax, yMax, pts)

// Query advance width by Unicode
aw := ttf.AdvanceWidthForRune(0xE001)
fmt.Printf("U+E001 advance width: %d\n", aw)
```

### Font Subsetting

```go
ttf, _ := gofont.Parse(data)

// Keep only the glyphs needed for these characters
err := ttf.Subset([]rune{'A', 'B', 'C', 'D', 'E'})
if err != nil {
    panic(err)
}

fmt.Printf("Glyphs after subsetting: %d\n", ttf.NumGlyphs())

out, _ := ttf.Serialize()
```

### Render Text with font.Drawer

```go
package main

import (
    "image"
    "image/color"
    "image/draw"
    "image/png"
    "os"

    gofont "github.com/venusliang/go-font"
    fontdraw "github.com/venusliang/go-font/draw"

    "golang.org/x/image/font"
    "golang.org/x/image/math/fixed"
)

func main() {
    // Load and parse the font
    data, _ := os.ReadFile("myfont.ttf")
    ttf, _ := gofont.Parse(data)

    // Create a rendering Face at 24pt / 72 DPI
    face := fontdraw.NewFace(&ttf, &fontdraw.FaceOptions{Size: 24, DPI: 72})
    defer face.Close()

    // Create a destination image
    img := image.NewRGBA(image.Rect(0, 0, 400, 60))
    draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)

    // Draw text using font.Drawer
    d := &font.Drawer{
        Dst:  img,
        Src:  image.Black,
        Face: face,
        Dot:  fixed.P(10, 40),
    }
    d.DrawString("Hello, World")

    // Save as PNG
    f, _ := os.Create("output.png")
    png.Encode(f, img)
    f.Close()
}
```

Colored text:

```go
// Blue text
blue := image.NewUniform(color.RGBA{0, 0, 255, 255})
d := &font.Drawer{
    Dst:  img,
    Src:  blue,
    Face: face,
    Dot:  fixed.P(10, 40),
}
d.DrawString("Blue text")
```

Multi-line text:

```go
m := face.Metrics()
lineHeight := m.Height >> 6 // convert to integer pixels

lines := []string{"Line one", "Line two", "Line three"}
for i, line := range lines {
    d.Dot = fixed.P(10, int(m.Ascent>>6)+i*int(lineHeight))
    d.DrawString(line)
}
```

### Modify Font Metrics

```go
ttf, _ := gofont.Parse(data)

// Change advance width of glyph 1
ttf.SetAdvanceWidth(1, 600)
ttf.SetLeftSideBearing(1, 20)

// Change font names
ttf.SetFontFamily("MyFont")
ttf.SetFontFullName("MyFont Regular")

out, _ := ttf.Serialize()
```

### Parse OpenType/CFF Fonts

```go
data, _ := os.ReadFile("myfont.otf")
ttf, _ := gofont.Parse(data)

if ttf.IsCFF() {
    fmt.Printf("CFF font name: %s\n", ttf.CFFFontName())
    fmt.Printf("Glyph count: %d\n", ttf.NumGlyphs())

    // Get glyph name
    name := ttf.CFFGlyphName(1)
    fmt.Printf("Glyph 1 name: %s\n", name)

    // Decode CFF outline
    outline := ttf.CFFOutlineForRune('A')
    if outline != nil {
        fmt.Printf("Segments: %d\n", outline.NumSegments())
        for _, seg := range outline.Segments() {
            switch seg.Op {
            case gofont.CFFOpMoveTo:
                fmt.Printf("  moveto %v\n", seg.Args[:2])
            case gofont.CFFOpLineTo:
                fmt.Printf("  lineto %v\n", seg.Args[:2])
            case gofont.CFFOpCurveTo:
                fmt.Printf("  curveto %v\n", seg.Args[:6])
            }
        }
    }
}

// Serialize (CFF raw table data is fully preserved)
out, _ := ttf.Serialize()
os.WriteFile("output.otf", out, 0644)
```

### Glyph Geometric Transforms

```go
ttf, _ := gofont.Parse(data)

// Translate glyph (right 100 units, down 50 units)
ttf.TranslateGlyph(1, 100, -50)

// Scale glyph (2x in both directions)
ttf.ScaleGlyph(1, 2.0, 2.0)

out, _ := ttf.Serialize()
```

### Append a New Glyph

```go
ttf, _ := gofont.Parse(data)

// Create a new glyph
newGlyph := &gofont.Glyph{
    header: gofont.GlyphHeader{
        numberOfContours: 1,
        xMin: 0, yMin: 0, xMax: 500, yMax: 700,
    },
    simpleGlyph: &gofont.SimpleGlyph{
        endPtsOfContours: []uint16{3},
        xCoordinates:     []int16{0, 500, 500, 0},
        yCoordinates:     []int16{0, 0, 700, 700},
    },
}

idx, _ := ttf.AppendGlyph(newGlyph)
ttf.SetRuneMapping('Z', uint16(idx))

out, _ := ttf.Serialize()
```

## Data Structures

### Glyph

```go
type Glyph struct {
    header         GlyphHeader      // contour count, bounding box
    simpleGlyph    *SimpleGlyph     // non-nil for simple glyphs
    compositeGlyph *CompositeGlyph  // non-nil for composite glyphs
}
```

**GlyphHeader**

```go
type GlyphHeader struct {
    numberOfContours int16  // >= 0 for simple, < 0 for composite
    xMin, yMin       int16  // bounding box minimum
    xMax, yMax       int16  // bounding box maximum
}
```

**SimpleGlyph**

```go
type SimpleGlyph struct {
    endPtsOfContours []uint16  // end point index for each contour
    instructions     []byte    // hinting instructions
    flags            []uint8   // per-point flags
    xCoordinates     []int16   // absolute X coordinates
    yCoordinates     []int16   // absolute Y coordinates
}
```

**CompositeGlyph**

```go
type CompositeGlyph struct {
    components []GlyphComponent
}

type GlyphComponent struct {
    flags      uint16     // component flags
    glyphIndex uint16     // referenced glyph index
    arg1, arg2 int16      // positioning arguments
    transform  [4]int16   // optional 2x2 transform matrix (F2Dot14)
}
```

### CFFOutline

```go
type CFFOutline struct {
    // Decoded outline data from CFF Type 2 CharStrings
}

type CFFPathSegment struct {
    Op   CFFPathOp  // CFFOpMoveTo / CFFOpLineTo / CFFOpCurveTo
    Args [6]int32   // Coordinate parameters (first 2 for MoveTo/LineTo, all 6 for CurveTo)
}

const (
    CFFOpMoveTo  CFFPathOp = iota  // Move to new position
    CFFOpLineTo                     // Straight line segment
    CFFOpCurveTo                    // Cubic Bezier curve
)
```

> CFF outlines use cubic Bezier curves (2 control points per curve segment), while TrueType outlines use quadratic Bezier curves.

## Supported Font Tables

| Table | File | Description |
|-------|------|-------------|
| `head` | `head.go` | Font header, global metrics |
| `hhea` | `hhea.go` | Horizontal layout header |
| `hmtx` | `hmtx.go` | Horizontal metrics (advance width + LSB) |
| `maxp` | `maxp.go` | Maximum profile, glyph count |
| `OS/2` | `os_2..go` | OS/2 metrics |
| `name` | `name.go` | Font name strings |
| `cmap` | `cmap.go` | Character-to-glyph mapping (formats 0/4/6/12) |
| `loca` | `loca.go` | Glyph index to offset mapping |
| `glyf` | `glyf.go` | Glyph outline data |
| `post` | `post.go` | PostScript name mapping |
| `kern` | `kern.go` | Kerning table |
| `GPOS` | `gpos.go` | Glyph positioning table |
| `GSUB` | `gsub.go` | Glyph substitution table |
| `CFF ` | `cff.go` | Compact Font Format table (OpenType/CFF fonts) |
| CharString | `cff_charstring.go` | CFF Type 2 CharString outline decoding |
| Rendering | `draw/face.go` | `font.Face` implementation, anti-aliased rasterization via `golang.org/x/image/vector` |
| SVG Export | `svg/svg.go` | Glyph outline export as standalone SVG documents |

## Supported Font Formats

| Format | File | Parse | Serialize | Description |
|--------|------|-------|-----------|-------------|
| TTF | `ttf.go` / `serialize.go` | `Parse()` | `Serialize()` | TrueType font |
| OTF | `ttf.go` / `cff.go` | `Parse()` | `Serialize()` | OpenType/CFF font |
| WOFF | `woff.go` | `ParseWOFF()` | `SerializeWOFF()` | zlib per-table compression |
| WOFF2 | `woff2.go` | `ParseWOFF2()` | `SerializeWOFF2()` | Brotli single-stream compression |
| EOT | `eot.go` | `ParseEOT()` | `SerializeEOT()` | Microsoft embedded font |
| TTC | `ttc.go` | `ParseTTC()` | `SerializeTTC()` | TrueType Collection |

## Running Tests

```bash
# Run all tests
go test ./...

# Run a specific test
go test -run TestRemoveGlyphs ./...

# Verbose output
go test -v ./...
```

## License

MIT License
