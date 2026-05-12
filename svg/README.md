# Package svg

```go
import "github.com/venusliang/go-font/svg"
```

Package svg provides standalone SVG document export for font glyph outlines. It uses the unified [GlyphPath](https://pkg.go.dev/github.com/venusliang/go-font#TrueTypeFont.GlyphPath) API from the `gofont` package and works with both TrueType outlines (quadratic Bezier) and CFF/OpenType outlines (cubic Bezier).

## Types

### Options

```go
type Options struct {
    // Padding around the glyph bounding box in SVG units. Defaults to 20.
    Padding float64
    // Scale maps font design units to SVG units. Defaults to 1.0.
    Scale float64
    // Fill is the SVG fill color (e.g. "black", "#ff0000"). Defaults to "black".
    Fill string
    // Stroke is the SVG stroke color. If empty, no stroke is rendered.
    Stroke string
    // StrokeWidth sets the SVG stroke-width attribute. Non-zero only when Stroke is set.
    StrokeWidth float64
}
```

Pass `nil` to use all defaults: padding 20, scale 1.0, fill "black", no stroke.

## Functions

### Glyph

```go
func Glyph(f *gofont.TrueTypeFont, glyphIndex int, opts *Options) (string, error)
```

Glyph returns a complete, standalone SVG document for the glyph at `glyphIndex`. The viewBox is computed from the glyph's bounding box with padding applied. Coordinates are converted from font design units (Y-axis up) to SVG (Y-axis down).

Returns an error if the index is out of range or the glyph has no outline.

### GlyphForRune

```go
func GlyphForRune(f *gofont.TrueTypeFont, r rune, opts *Options) (string, error)
```

GlyphForRune returns an SVG document for the glyph mapped to rune `r`. Returns an error if the rune is not mapped or the glyph has no outline.

## Usage

### Basic export

```go
package main

import (
    "fmt"
    "os"

    gofont "github.com/venusliang/go-font"
    svgexp "github.com/venusliang/go-font/svg"
)

func main() {
    data, _ := os.ReadFile("myfont.ttf")
    ttf, _ := gofont.Parse(data)

    // Export glyph for 'A' with default options
    svg, err := svgexp.GlyphForRune(&ttf, 'A', nil)
    if err != nil {
        panic(err)
    }
    fmt.Println(svg)
}
```

Output:

```xml
<svg xmlns="http://www.w3.org/2000/svg" viewBox="3.00 -1569.00 1434.00 1589.00">
<path d="M1417.00,-0.00L1196.00,-0.00L1038.00,-424.00..." fill="black"/>
</svg>
```

### Custom styling

```go
// Red glyph with blue outline, scaled
opts := &svgexp.Options{
    Fill:        "red",
    Stroke:      "blue",
    StrokeWidth: 2.5,
    Scale:       0.5,
}
svg, _ := svgexp.GlyphForRune(&ttf, 'A', opts)
```

### Export by glyph index

```go
// Export glyph 0 (.notdef) at 2x scale
svg, err := svgexp.Glyph(&ttf, 0, &svgexp.Options{Scale: 2.0})
```

### Batch export multiple glyphs

```go
chars := []rune{'A', 'B', 'C'}
for _, ch := range chars {
    svg, err := svgexp.GlyphForRune(&ttf, ch, nil)
    if err != nil {
        continue // skip glyphs with no outline
    }
    filename := fmt.Sprintf("glyph_%04X.svg", ch)
    os.WriteFile(filename, []byte(svg), 0644)
}
```
