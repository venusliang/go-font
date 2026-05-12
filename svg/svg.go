// Package svg provides SVG document export for font glyph outlines
// using the unified GlyphPath API.
package svg

import (
	"fmt"
	"strings"

	gofont "github.com/venusliang/go-font"
)

// Options controls the SVG output.
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

func defaultOpts(opts *Options) Options {
	o := Options{}
	if opts != nil {
		o = *opts
	}
	if o.Fill == "" {
		o.Fill = "black"
	}
	if o.Padding == 0 {
		o.Padding = 20
	}
	if o.Scale <= 0 {
		o.Scale = 1
	}
	return o
}

// Glyph returns a complete, standalone SVG document for the glyph at glyphIndex.
// The SVG viewBox is computed from the glyph's bounding box with padding applied.
// Coordinates are converted from font design units (Y-up) to SVG (Y-down).
func Glyph(f *gofont.TrueTypeFont, glyphIndex int, opts *Options) (string, error) {
	o := defaultOpts(opts)

	path, err := f.GlyphPath(glyphIndex)
	if err != nil {
		return "", err
	}

	if len(path.Segments) == 0 {
		return "", fmt.Errorf("svg: glyph %d has no outline", glyphIndex)
	}

	return buildSVG(path, o), nil
}

// GlyphForRune returns an SVG document for the glyph that maps to rune r.
func GlyphForRune(f *gofont.TrueTypeFont, r rune, opts *Options) (string, error) {
	gid := f.RuneToGlyphID(r)
	if gid == 0 {
		return "", fmt.Errorf("svg: no glyph for rune U+%04X", r)
	}
	return Glyph(f, int(gid), opts)
}

// buildSVG constructs a complete SVG document from a GlyphPath.
func buildSVG(path *gofont.GlyphPath, o Options) string {
	d := pathData(path.Segments, o.Scale)

	sf := o.Scale
	x := float64(path.XMin)*sf - o.Padding
	y := float64(-path.YMax)*sf - o.Padding
	w := float64(path.XMax-path.XMin)*sf + 2*o.Padding
	h := float64(path.YMax-path.YMin)*sf + 2*o.Padding

	var sb strings.Builder
	sb.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" `)
	fmt.Fprintf(&sb, `viewBox="%.2f %.2f %.2f %.2f">`, x, y, w, h)
	fmt.Fprint(&sb, "\n<path ")
	fmt.Fprintf(&sb, `d="%s"`, d)
	fmt.Fprintf(&sb, ` fill="%s"`, o.Fill)
	if o.Stroke != "" {
		fmt.Fprintf(&sb, ` stroke="%s"`, o.Stroke)
		if o.StrokeWidth != 0 {
			fmt.Fprintf(&sb, ` stroke-width="%.2f"`, o.StrokeWidth)
		}
	}
	sb.WriteString("/>\n</svg>")

	return sb.String()
}

// pathData converts path segments to an SVG "d" attribute string.
// Font design units (Y-up) → SVG coordinates (Y-down).
func pathData(segs []gofont.PathSegment, scale float64) string {
	var b strings.Builder
	firstSeg := true
	for _, seg := range segs {
		switch seg.Op {
		case gofont.OpMoveTo:
			if !firstSeg {
				b.WriteString("Z ")
			}
			firstSeg = false
			fmt.Fprintf(&b, "M%.2f,%.2f",
				float64(seg.Args[0])*scale,
				float64(-seg.Args[1])*scale)
		case gofont.OpLineTo:
			fmt.Fprintf(&b, "L%.2f,%.2f",
				float64(seg.Args[0])*scale,
				float64(-seg.Args[1])*scale)
		case gofont.OpQuadTo:
			fmt.Fprintf(&b, "Q%.2f,%.2f %.2f,%.2f",
				float64(seg.Args[0])*scale, float64(-seg.Args[1])*scale,
				float64(seg.Args[2])*scale, float64(-seg.Args[3])*scale)
		case gofont.OpCubeTo:
			fmt.Fprintf(&b, "C%.2f,%.2f %.2f,%.2f %.2f,%.2f",
				float64(seg.Args[0])*scale, float64(-seg.Args[1])*scale,
				float64(seg.Args[2])*scale, float64(-seg.Args[3])*scale,
				float64(seg.Args[4])*scale, float64(-seg.Args[5])*scale)
		}
	}
	if !firstSeg {
		b.WriteString("Z")
	}
	return b.String()
}
