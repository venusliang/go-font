# Package draw

```go
import "github.com/venusliang/go-font/draw"
```

Package draw implements `golang.org/x/image/font.Face` for fonts parsed by the `gofont` package, enabling anti-aliased text rendering with `font.Drawer` onto `image.Image` targets.

Supports both TrueType outlines (quadratic Bezier curves) and CFF/OpenType outlines (cubic Bezier curves).

## Types

### Face

```go
type Face struct { /* ... */ }
```

Face implements `font.Face`. A Face is created at a specific size and DPI for a parsed `*gofont.TrueTypeFont`.

A Face is **not safe for concurrent use**.

### FaceOptions

```go
type FaceOptions struct {
    Size    float64      // Font size in points (default 12)
    DPI     float64      // Dots per inch (default 72)
    Hinting font.Hinting // Hinting mode (only HintingNone supported)
}
```

## Functions

### NewFace

```go
func NewFace(f *gofont.TrueTypeFont, opts *FaceOptions) *Face
```

NewFace creates a new Face for rendering the given font. If `opts` is nil, defaults are used (size 12pt, 72 DPI).

## Methods

### Close

```go
func (face *Face) Close() error
```

Close implements `font.Face.Close`. The underlying font is owned by the caller and is not freed.

### Metrics

```go
func (face *Face) Metrics() font.Metrics
```

Metrics returns font metrics in 26.6 fixed-point pixel units, scaled to the configured size and DPI.

### GlyphAdvance

```go
func (face *Face) GlyphAdvance(r rune) (advance fixed.Int26_6, ok bool)
```

GlyphAdvance returns the advance width for rune `r`. Returns `ok == false` if the rune is not mapped.

### GlyphBounds

```go
func (face *Face) GlyphBounds(r rune) (bounds fixed.Rectangle26_6, advance fixed.Int26_6, ok bool)
```

GlyphBounds returns the bounding box and advance width for rune `r`.

### Kern

```go
func (face *Face) Kern(r0, r1 rune) fixed.Int26_6
```

Kern returns the kerning adjustment between runes `r0` and `r1`.

### Glyph

```go
func (face *Face) Glyph(dot fixed.Point26_6, r rune) (
    dr image.Rectangle, mask image.Image, maskp image.Point,
    advance fixed.Int26_6, ok bool,
)
```

Glyph rasterizes the glyph for rune `r` at position `dot`, returning the destination rectangle, an anti-aliased alpha mask, and the advance width.

## Usage

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

    // Create a rendering face at 24pt / 72 DPI
    face := fontdraw.NewFace(&ttf, &fontdraw.FaceOptions{Size: 24, DPI: 72})
    defer face.Close()

    // Create a destination image
    img := image.NewRGBA(image.Rect(0, 0, 400, 60))
    draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)

    // Draw text
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

**Colored text:**

```go
blue := image.NewUniform(color.RGBA{0, 0, 255, 255})
d := &font.Drawer{
    Dst:  img,
    Src:  blue,
    Face: face,
    Dot:  fixed.P(10, 40),
}
d.DrawString("Blue text")
```

**Multi-line text:**

```go
m := face.Metrics()
lineHeight := m.Height >> 6 // convert to integer pixels

lines := []string{"Line one", "Line two", "Line three"}
for i, line := range lines {
    d.Dot = fixed.P(10, int(m.Ascent>>6)+i*int(lineHeight))
    d.DrawString(line)
}
```
