package draw

import (
	"image"

	gofont "github.com/venusliang/go-font"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
)

// FaceOptions holds configuration for creating a new Face.
type FaceOptions struct {
	Size    float64      // Font size in points (default 12)
	DPI     float64      // Dots per inch (default 72)
	Hinting font.Hinting // Hinting mode (only HintingNone supported)
}

// Face implements font.Face for a parsed TrueTypeFont at a specific size.
// A Face is not safe for concurrent use.
type Face struct {
	f     *gofont.TrueTypeFont
	scale fixed.Int26_6 // pixels-per-em in 26.6 fixed-point

	metrics    font.Metrics
	metricsSet bool

	// Reusable buffers
	buf  []faceSegment
	rast vector.Rasterizer
	mask image.Alpha
}

// faceSegment represents a path segment for rasterization.
type faceSegment struct {
	op   uint8 // 0=moveTo, 1=lineTo, 2=quadTo, 3=cubeTo
	args [6]float32
}

const (
	segMoveTo uint8 = iota
	segLineTo
	segQuadTo
	segCubeTo
)

// NewFace creates a new Face for rendering the given TrueTypeFont at the
// specified size and DPI.
func NewFace(f *gofont.TrueTypeFont, opts *FaceOptions) *Face {
	return newFace(f, opts)
}

// newFace is the shared implementation.
func newFace(f *gofont.TrueTypeFont, opts *FaceOptions) *Face {
	if opts == nil {
		opts = &FaceOptions{}
	}
	size := opts.Size
	if size <= 0 {
		size = 12
	}
	dpi := opts.DPI
	if dpi <= 0 {
		dpi = 72
	}

	scale := fixed.Int26_6(0.5 + size*dpi*64/72)

	face := &Face{
		f:     f,
		scale: scale,
	}

	return face
}

// Close implements font.Face. It is a no-op; the TrueTypeFont is owned by the caller.
func (face *Face) Close() error {
	return nil
}

// scaleInt16 converts a font design-unit value to 26.6 fixed-point pixel coordinates.
func (face *Face) scaleInt16(v int16) fixed.Int26_6 {
	upem := fixed.Int26_6(face.f.UnitsPerEm())
	if upem == 0 {
		return 0
	}
	return fixed.Int26_6(v) * face.scale / upem
}

// Metrics implements font.Face. It returns the font metrics in pixel units (26.6 fixed-point).
func (face *Face) Metrics() font.Metrics {
	if face.metricsSet {
		return face.metrics
	}

	ascent := face.f.Ascent()
	descent := face.f.Descent()
	lineGap := face.f.LineGap()
	caretRun, caretRise := face.f.CaretSlope()

	face.metrics.Height = face.scaleInt16(ascent - descent + lineGap)
	face.metrics.Ascent = face.scaleInt16(ascent)
	// descent is negative; font.Metrics.Descent is positive (distance below baseline)
	face.metrics.Descent = face.scaleInt16(-descent)
	face.metrics.CaretSlope = image.Pt(int(caretRun), int(caretRise))
	face.metrics.XHeight = face.scaleInt16(face.f.XHeight())
	face.metrics.CapHeight = face.scaleInt16(face.f.CapHeight())

	face.metricsSet = true
	return face.metrics
}

// GlyphAdvance implements font.Face.
func (face *Face) GlyphAdvance(r rune) (advance fixed.Int26_6, ok bool) {
	glyphID := face.f.RuneToGlyphID(r)
	if glyphID == 0 {
		return 0, false
	}
	aw := face.f.AdvanceWidth(glyphID)
	return face.scaleInt16(int16(aw)), true
}

// GlyphBounds implements font.Face.
func (face *Face) GlyphBounds(r rune) (bounds fixed.Rectangle26_6, advance fixed.Int26_6, ok bool) {
	glyphID := face.f.RuneToGlyphID(r)
	if glyphID == 0 {
		return fixed.Rectangle26_6{}, 0, false
	}

	aw := face.f.AdvanceWidth(glyphID)
	advance = face.scaleInt16(int16(aw))

	path, err := face.f.GlyphPath(int(glyphID))
	if err != nil {
		return fixed.Rectangle26_6{}, advance, true
	}

	upem := float32(face.f.UnitsPerEm())
	if upem == 0 {
		return fixed.Rectangle26_6{}, advance, true
	}
	sf := float32(face.scale) / upem

	bounds.Min.X = fixed.Int26_6(path.XMin * sf)
	bounds.Min.Y = fixed.Int26_6(-path.YMax * sf) // Y-flip: max -> min
	bounds.Max.X = fixed.Int26_6(path.XMax * sf)
	bounds.Max.Y = fixed.Int26_6(-path.YMin * sf) // Y-flip: min -> max

	return bounds, advance, true
}

// Kern implements font.Face.
func (face *Face) Kern(r0, r1 rune) fixed.Int26_6 {
	g0 := face.f.RuneToGlyphID(r0)
	g1 := face.f.RuneToGlyphID(r1)
	if g0 == 0 || g1 == 0 {
		return 0
	}
	return face.scaleInt16(face.f.KernPair(g0, g1))
}

// Glyph implements font.Face. It rasterizes the glyph for rune r at position dot
// and returns the destination rectangle, alpha mask, and advance width.
func (face *Face) Glyph(dot fixed.Point26_6, r rune) (
	dr image.Rectangle, mask image.Image, maskp image.Point, advance fixed.Int26_6, ok bool,
) {
	glyphID := face.f.RuneToGlyphID(r)
	if glyphID == 0 {
		return image.Rectangle{}, nil, image.Point{}, 0, false
	}

	aw := face.f.AdvanceWidth(glyphID)
	advance = face.scaleInt16(int16(aw))

	// Extract outline segments
	face.buf = face.buf[:0]
	face.buf = face.loadSegments(glyphID)
	if len(face.buf) == 0 {
		return image.Rectangle{}, nil, image.Point{}, advance, true
	}

	// Compute bounding rectangle from segments
	var minX, minY, maxX, maxY float32
	first := true
	for _, seg := range face.buf {
		pts := segPoints(seg)
		for _, p := range pts {
			if first {
				minX, minY, maxX, maxY = p[0], p[1], p[0], p[1]
				first = false
			}
			if p[0] < minX {
				minX = p[0]
			}
			if p[1] < minY {
				minY = p[1]
			}
			if p[0] > maxX {
				maxX = p[0]
			}
			if p[1] > maxY {
				maxY = p[1]
			}
		}
	}
	if first {
		return image.Rectangle{}, nil, image.Point{}, advance, true
	}

	// Translate to dot position (26.6 -> float32 pixels)
	originX := float32(dot.X) / 64
	originY := float32(dot.Y) / 64

	dr = image.Rect(
		int(floorFloat32(minX+originX)),
		int(floorFloat32(minY+originY)),
		int(ceilFloat32(maxX+originX)),
		int(ceilFloat32(maxY+originY)),
	)
	if dr.Empty() {
		return dr, nil, image.Point{}, advance, true
	}

	// Sub-pixel bias: compensate for the integer quantization of dr
	biasX := originX - float32(dr.Min.X)
	biasY := originY - float32(dr.Min.Y)

	w := dr.Dx()
	h := dr.Dy()

	// Grow mask if needed
	if face.mask.Rect.Dx() < w || face.mask.Rect.Dy() < h {
		face.mask = *image.NewAlpha(image.Rect(0, 0, w, h))
	} else {
		face.mask.Rect = image.Rect(0, 0, w, h)
		for i := range face.mask.Pix {
			face.mask.Pix[i] = 0
		}
	}

	// Rasterize
	face.rast.Reset(w, h)
	for _, seg := range face.buf {
		switch seg.op {
		case segMoveTo:
			face.rast.MoveTo(seg.args[0]+biasX, seg.args[1]+biasY)
		case segLineTo:
			face.rast.LineTo(seg.args[0]+biasX, seg.args[1]+biasY)
		case segQuadTo:
			face.rast.QuadTo(
				seg.args[0]+biasX, seg.args[1]+biasY,
				seg.args[2]+biasX, seg.args[3]+biasY,
			)
		case segCubeTo:
			face.rast.CubeTo(
				seg.args[0]+biasX, seg.args[1]+biasY,
				seg.args[2]+biasX, seg.args[3]+biasY,
				seg.args[4]+biasX, seg.args[5]+biasY,
			)
		}
	}
	face.rast.Draw(&face.mask, face.mask.Rect, image.Opaque, image.Point{})

	return dr, &face.mask, image.Point{}, advance, true
}

// loadSegments extracts scaled, Y-flipped outline segments for a glyph.
func (face *Face) loadSegments(glyphID uint16) []faceSegment {
	path, err := face.f.GlyphPath(int(glyphID))
	if err != nil || len(path.Segments) == 0 {
		return nil
	}

	upem := float32(face.f.UnitsPerEm())
	if upem == 0 {
		return nil
	}
	sf := float32(face.scale) / upem / 64

	segs := face.buf[:0]
	for _, seg := range path.Segments {
		switch seg.Op {
		case gofont.OpMoveTo:
			segs = append(segs, faceSegment{
				op:   segMoveTo,
				args: [6]float32{seg.Args[0] * sf, -seg.Args[1] * sf},
			})
		case gofont.OpLineTo:
			segs = append(segs, faceSegment{
				op:   segLineTo,
				args: [6]float32{seg.Args[0] * sf, -seg.Args[1] * sf},
			})
		case gofont.OpQuadTo:
			segs = append(segs, faceSegment{
				op: segQuadTo,
				args: [6]float32{
					seg.Args[0] * sf, -seg.Args[1] * sf,
					seg.Args[2] * sf, -seg.Args[3] * sf,
				},
			})
		case gofont.OpCubeTo:
			segs = append(segs, faceSegment{
				op: segCubeTo,
				args: [6]float32{
					seg.Args[0] * sf, -seg.Args[1] * sf,
					seg.Args[2] * sf, -seg.Args[3] * sf,
					seg.Args[4] * sf, -seg.Args[5] * sf,
				},
			})
		}
	}
	return segs
}

// segPoints returns the coordinate pairs that define a segment's shape.
func segPoints(seg faceSegment) [][2]float32 {
	switch seg.op {
	case segMoveTo:
		return [][2]float32{{seg.args[0], seg.args[1]}}
	case segLineTo:
		return [][2]float32{{seg.args[0], seg.args[1]}}
	case segQuadTo:
		return [][2]float32{
			{seg.args[0], seg.args[1]}, // control
			{seg.args[2], seg.args[3]}, // end
		}
	case segCubeTo:
		return [][2]float32{
			{seg.args[0], seg.args[1]}, // control 1
			{seg.args[2], seg.args[3]}, // control 2
			{seg.args[4], seg.args[5]}, // end
		}
	}
	return nil
}

func floorFloat32(v float32) float32 {
	f := float32(int(v))
	if f > v {
		f--
	}
	return f
}

func ceilFloat32(v float32) float32 {
	f := float32(int(v))
	if f < v {
		f++
	}
	return f
}
