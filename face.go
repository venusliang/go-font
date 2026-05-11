package gofont

import (
	"image"

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
	f     *TrueTypeFont
	scale fixed.Int26_6 // pixels-per-em in 26.6 fixed-point

	metrics    font.Metrics
	metricsSet bool

	// CFF outline cache (decoded once for CFF fonts)
	cffOutlines []*CFFOutline

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

// NewFace creates a new Face for rendering the font at the specified size and DPI.
// This is a convenience method equivalent to calling the package-level NewFace function.
func (ttf *TrueTypeFont) NewFace(opts *FaceOptions) *Face {
	return newFace(ttf, opts)
}

// NewFace creates a new Face for rendering the given TrueTypeFont at the
// specified size and DPI.
func NewFace(f *TrueTypeFont, opts *FaceOptions) *Face {
	return newFace(f, opts)
}

// newFace is the shared implementation.
func newFace(f *TrueTypeFont, opts *FaceOptions) *Face {
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

	// Pre-decode CFF outlines if applicable
	if f.IsCFF() && f.cff != nil {
		outlines, err := f.cff.DecodeOutlines()
		if err == nil {
			face.cffOutlines = outlines
		}
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

	hhea := face.f.hhea
	os2 := face.f.os2

	if hhea != nil {
		face.metrics.Height = face.scaleInt16(hhea.ascent - hhea.descent + hhea.lineGap)
		face.metrics.Ascent = face.scaleInt16(hhea.ascent)
		// hhea.descent is negative; font.Metrics.Descent is positive (distance below baseline)
		face.metrics.Descent = face.scaleInt16(-hhea.descent)
		face.metrics.CaretSlope = image.Pt(int(hhea.caretSlopeRun), int(hhea.caretSlopeRise))
	}

	if os2 != nil {
		face.metrics.XHeight = face.scaleInt16(os2.sxHeight)
		face.metrics.CapHeight = face.scaleInt16(os2.sCapHeight)
	}

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

	var xMin, yMin, xMax, yMax int16
	if face.f.IsCFF() {
		if outline := face.getCFFOutline(int(glyphID)); outline != nil {
			xMin, yMin, xMax, yMax = outline.BBox()
		}
	} else {
		var bboxOk bool
		xMin, yMin, xMax, yMax, bboxOk = face.f.GlyphBBox(int(glyphID))
		if !bboxOk {
			return fixed.Rectangle26_6{}, advance, true
		}
	}

	// Y-flip: OpenType Y up -> image Y down
	bounds.Min.X = face.scaleInt16(xMin)
	bounds.Min.Y = -face.scaleInt16(yMax)
	bounds.Max.X = face.scaleInt16(xMax)
	bounds.Max.Y = -face.scaleInt16(yMin)

	return bounds, advance, true
}

// Kern implements font.Face.
func (face *Face) Kern(r0, r1 rune) fixed.Int26_6 {
	g0 := face.f.RuneToGlyphID(r0)
	g1 := face.f.RuneToGlyphID(r1)
	if g0 == 0 || g1 == 0 {
		return 0
	}
	return face.scaleInt16(face.lookupKern(g0, g1))
}

// lookupKern searches the kern table for a kerning value between two glyph IDs.
func (face *Face) lookupKern(left, right uint16) int16 {
	kern := face.f.kern
	if kern == nil {
		return 0
	}
	for _, sub := range kern.subtables {
		if sub.format != 0 {
			continue
		}
		// Binary search in sorted pairs
		lo, hi := 0, len(sub.pairs)
		for lo < hi {
			mid := (lo + hi) / 2
			p := sub.pairs[mid]
			if p.Left < left || (p.Left == left && p.Right < right) {
				lo = mid + 1
			} else {
				hi = mid
			}
		}
		if lo < len(sub.pairs) && sub.pairs[lo].Left == left && sub.pairs[lo].Right == right {
			return sub.pairs[lo].Value
		}
	}
	return 0
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
	if face.f.IsCFF() {
		outline := face.getCFFOutline(int(glyphID))
		if outline == nil {
			return nil
		}
		return face.extractCFFOutline(outline, face.buf)
	}
	return face.extractTTOutline(int(glyphID), 0)
}

// getCFFOutline returns the cached CFF outline for the given glyph index.
func (face *Face) getCFFOutline(index int) *CFFOutline {
	if face.cffOutlines == nil || index < 0 || index >= len(face.cffOutlines) {
		return nil
	}
	return face.cffOutlines[index]
}

// extractTTOutline extracts TrueType outline segments (quadratic bezier).
func (face *Face) extractTTOutline(glyphIndex int, depth int) []faceSegment {
	if depth > 8 || glyphIndex < 0 || glyphIndex >= len(face.f.glyf) {
		return nil
	}
	g := face.f.glyf[glyphIndex]
	if g == nil {
		return nil
	}

	if g.simpleGlyph != nil {
		return face.extractSimpleGlyph(g.simpleGlyph)
	}
	if g.compositeGlyph != nil {
		return face.extractCompositeGlyph(g.compositeGlyph, depth)
	}
	return nil
}

// extractSimpleGlyph converts a TrueType simple glyph to scaled path segments.
// TrueType contours use quadratic Bézier curves: on-curve points are connected
// directly (LineTo), while off-curve points are control points. Two consecutive
// off-curve points have an implied on-curve point at their midpoint.
func (face *Face) extractSimpleGlyph(sg *SimpleGlyph) []faceSegment {
	if sg == nil || len(sg.flags) == 0 {
		return nil
	}

	segs := make([]faceSegment, 0)
	xCoords := sg.xCoordinates
	yCoords := sg.yCoordinates
	flags := sg.flags
	nPts := len(flags)
	upem := float32(face.f.UnitsPerEm())
	sf := float32(face.scale) / upem / 64

	scaleX := func(v int16) float32 { return float32(v) * sf }
	scaleY := func(v int16) float32 { return -float32(v) * sf } // Y-flip

	contourStart := 0
	for _, endPt := range sg.endPtsOfContours {
		contourEnd := int(endPt)
		if contourEnd >= nPts {
			break
		}
		n := contourEnd - contourStart + 1 // points in this contour

		// Helper: get wrapped index within contour
		idx := func(i int) int { return contourStart + ((i - contourStart + n) % n) }
		onCurve := func(i int) bool { return flags[i]&glyphFlagOnCurve != 0 }

		// Find a starting on-curve point, or create midpoint of two off-curve
		start := -1
		for i := contourStart; i <= contourEnd; i++ {
			if onCurve(i) {
				start = i
				break
			}
		}

		var penX, penY float32
		if start >= 0 {
			penX = scaleX(xCoords[start])
			penY = scaleY(yCoords[start])
			segs = append(segs, faceSegment{op: segMoveTo, args: [6]float32{penX, penY}})
		} else {
			// All off-curve: midpoint of first and last points
			f := contourStart
			l := contourEnd
			mx := scaleX(int16((int32(xCoords[f]) + int32(xCoords[l])) / 2))
			my := scaleY(int16((int32(yCoords[f]) + int32(yCoords[l])) / 2))
			penX, penY = mx, my
			segs = append(segs, faceSegment{op: segMoveTo, args: [6]float32{penX, penY}})
			start = contourStart
		}

		// Walk the contour from start
		i := start
		for step := 0; step < n; step++ {
			j := idx(i + 1)
			if onCurve(j) {
				penX = scaleX(xCoords[j])
				penY = scaleY(yCoords[j])
				segs = append(segs, faceSegment{op: segLineTo, args: [6]float32{penX, penY}})
				i = j
			} else {
				// j is off-curve control point
				cpX := scaleX(xCoords[j])
				cpY := scaleY(yCoords[j])

				k := idx(j + 1)
				if onCurve(k) {
					// off + on -> quad bezier
					penX = scaleX(xCoords[k])
					penY = scaleY(yCoords[k])
					segs = append(segs, faceSegment{op: segQuadTo, args: [6]float32{cpX, cpY, penX, penY}})
					i = k
					step++
				} else {
					// off + off -> implicit midpoint on-curve
					mx := scaleX(int16((int32(xCoords[j]) + int32(xCoords[k])) / 2))
					my := scaleY(int16((int32(yCoords[j]) + int32(yCoords[k])) / 2))
					segs = append(segs, faceSegment{op: segQuadTo, args: [6]float32{cpX, cpY, mx, my}})
					penX, penY = mx, my
					i = j
				}
			}
		}

		contourStart = contourEnd + 1
	}

	return segs
}

// extractCompositeGlyph recursively extracts segments from a composite glyph.
func (face *Face) extractCompositeGlyph(cg *CompositeGlyph, depth int) []faceSegment {
	var segs []faceSegment
	for _, comp := range cg.components {
		subSegs := face.extractTTOutline(int(comp.glyphIndex), depth+1)
		if len(subSegs) == 0 {
			continue
		}

		// Compute transform
		upem := float32(face.f.UnitsPerEm())
		sf := float32(face.scale) / upem / 64
		dx := float32(comp.arg1) * sf
		dy := -float32(comp.arg2) * sf // Y-flip

		var m00, m01, m10, m11 float32 = 1, 0, 0, 1
		if comp.hasScale {
			s := f2dot14ToFloat(comp.transform[0])
			m00, m11 = s, s
		} else if comp.hasXYScale {
			m00 = f2dot14ToFloat(comp.transform[0])
			m11 = f2dot14ToFloat(comp.transform[3])
		} else if comp.has2x2 {
			m00 = f2dot14ToFloat(comp.transform[0])
			m01 = f2dot14ToFloat(comp.transform[1])
			m10 = f2dot14ToFloat(comp.transform[2])
			m11 = f2dot14ToFloat(comp.transform[3])
		}

		transformPt := func(x, y float32) (float32, float32) {
			return m00*x + m01*y + dx, m10*x + m11*y + dy
		}

		for _, seg := range subSegs {
			s := seg
			switch s.op {
			case segMoveTo:
				s.args[0], s.args[1] = transformPt(s.args[0], s.args[1])
			case segLineTo:
				s.args[0], s.args[1] = transformPt(s.args[0], s.args[1])
			case segQuadTo:
				s.args[0], s.args[1] = transformPt(s.args[0], s.args[1])
				s.args[2], s.args[3] = transformPt(s.args[2], s.args[3])
			case segCubeTo:
				s.args[0], s.args[1] = transformPt(s.args[0], s.args[1])
				s.args[2], s.args[3] = transformPt(s.args[2], s.args[3])
				s.args[4], s.args[5] = transformPt(s.args[4], s.args[5])
			}
			segs = append(segs, s)
		}
	}
	return segs
}

// f2dot14ToFloat converts an F2Dot14 fixed-point value to float32.
func f2dot14ToFloat(v int16) float32 {
	return float32(v) / 16384.0
}

// extractCFFOutline converts CFF outline segments (relative coordinates) to
// absolute scaled path segments.
func (face *Face) extractCFFOutline(outline *CFFOutline, segs []faceSegment) []faceSegment {
	if outline == nil {
		return segs
	}
	segments := outline.Segments()
	if len(segments) == 0 {
		return segs
	}

	upem := float32(face.f.UnitsPerEm())
	sf := float32(face.scale) / upem / 64

	scaleF := func(v int32) float32 { return float32(v) * sf }
	flipY := func(v float32) float32 { return -v }

	// CFF segments use relative coordinates
	var penX, penY float32 // absolute pixel position (scaled, Y-flipped)

	for _, seg := range segments {
		switch seg.Op {
		case CFFOpMoveTo:
			penX += scaleF(seg.Args[0])
			penY += flipY(scaleF(seg.Args[1]))
			segs = append(segs, faceSegment{op: segMoveTo, args: [6]float32{penX, penY}})
		case CFFOpLineTo:
			penX += scaleF(seg.Args[0])
			penY += flipY(scaleF(seg.Args[1]))
			segs = append(segs, faceSegment{op: segLineTo, args: [6]float32{penX, penY}})
		case CFFOpCurveTo:
			// Args: (dx1, dy1, dx2, dy2, dx3, dy3) - relative
			// Control point 1
			cp1x := penX + scaleF(seg.Args[0])
			cp1y := penY + flipY(scaleF(seg.Args[1]))
			// Control point 2
			cp2x := penX + scaleF(seg.Args[2])
			cp2y := penY + flipY(scaleF(seg.Args[3]))
			// End point
			penX += scaleF(seg.Args[4])
			penY += flipY(scaleF(seg.Args[5]))

			segs = append(segs, faceSegment{
				op: segCubeTo,
				args: [6]float32{cp1x, cp1y, cp2x, cp2y, penX, penY},
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
