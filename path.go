package gofont

import "fmt"

// PathOp represents a path drawing operation.
type PathOp uint8

const (
	OpMoveTo PathOp = iota // move to (Args[0], Args[1])
	OpLineTo                // line to (Args[0], Args[1])
	OpQuadTo                // quadratic Bézier: control at (Args[0], Args[1]), end at (Args[2], Args[3])
	OpCubeTo                // cubic Bézier: ctrl1 (Args[0], Args[1]), ctrl2 (Args[2], Args[3]), end (Args[4], Args[5])
)

// PathSegment represents one path drawing command with absolute coordinates
// in font design units (Y-axis points up, per OpenType convention).
type PathSegment struct {
	Op   PathOp
	Args [6]float32
}

// GlyphPath holds the outline path data for a single glyph.
// Coordinates are absolute, in font design units, Y-up.
type GlyphPath struct {
	Segments               []PathSegment
	XMin, YMin, XMax, YMax float32
}

// GlyphPath returns the outline path for the glyph at the given index.
// Works for both TrueType (quadratic beziers) and CFF/OpenType (cubic beziers) fonts.
// Returns an empty path (no segments) for glyphs with no outline (e.g. space, .notdef).
func (f *TrueTypeFont) GlyphPath(glyphIndex int) (*GlyphPath, error) {
	if f.IsCFF() {
		return f.cffGlyphPath(glyphIndex)
	}
	return f.ttGlyphPath(glyphIndex)
}

func (f *TrueTypeFont) ttGlyphPath(glyphIndex int) (*GlyphPath, error) {
	if glyphIndex < 0 || glyphIndex >= len(f.glyf) {
		return nil, fmt.Errorf("glyph index %d out of range (0-%d)", glyphIndex, len(f.glyf)-1)
	}
	g := f.glyf[glyphIndex]
	if g == nil {
		return &GlyphPath{}, nil
	}
	segs := extractGlyphPath(f, g, 0)
	if len(segs) == 0 {
		return &GlyphPath{}, nil
	}
	xMin, yMin, xMax, yMax := pathBBox(segs)
	return &GlyphPath{Segments: segs, XMin: xMin, YMin: yMin, XMax: xMax, YMax: yMax}, nil
}

func (f *TrueTypeFont) cffGlyphPath(glyphIndex int) (*GlyphPath, error) {
	if f.cff == nil {
		return nil, fmt.Errorf("CFF table not present")
	}
	n := int(f.cff.charStrings.count)
	if glyphIndex < 0 || glyphIndex >= n {
		return nil, fmt.Errorf("glyph index %d out of range (0-%d)", glyphIndex, n-1)
	}

	csData, err := indexElement(&f.cff.charStrings, glyphIndex)
	if err != nil {
		return nil, fmt.Errorf("glyph %d charstring: %w", glyphIndex, err)
	}

	outline, err := decodeCharString(
		csData,
		&f.cff.globalSubrs,
		&f.cff.localSubrs,
		f.cff.privateDict.defaultWidthX,
		f.cff.privateDict.nominalWidthX,
	)
	if err != nil {
		return nil, fmt.Errorf("glyph %d charstring: %w", glyphIndex, err)
	}

	cffSegs := outline.Segments()
	if len(cffSegs) == 0 {
		xMin, yMin, xMax, yMax := outline.BBox()
		return &GlyphPath{
			XMin: float32(xMin), YMin: float32(yMin),
			XMax: float32(xMax), YMax: float32(yMax),
		}, nil
	}

	segs := make([]PathSegment, 0, len(cffSegs))
	var penX, penY float32
	for _, cs := range cffSegs {
		switch cs.Op {
		case CFFOpMoveTo:
			penX += float32(cs.Args[0])
			penY += float32(cs.Args[1])
			segs = append(segs, PathSegment{Op: OpMoveTo, Args: [6]float32{penX, penY}})
		case CFFOpLineTo:
			penX += float32(cs.Args[0])
			penY += float32(cs.Args[1])
			segs = append(segs, PathSegment{Op: OpLineTo, Args: [6]float32{penX, penY}})
		case CFFOpCurveTo:
			cp1x := penX + float32(cs.Args[0])
			cp1y := penY + float32(cs.Args[1])
			cp2x := penX + float32(cs.Args[2])
			cp2y := penY + float32(cs.Args[3])
			penX += float32(cs.Args[4])
			penY += float32(cs.Args[5])
			segs = append(segs, PathSegment{
				Op: OpCubeTo,
				Args: [6]float32{cp1x, cp1y, cp2x, cp2y, penX, penY},
			})
		}
	}

	xMin, yMin, xMax, yMax := pathBBox(segs)
	return &GlyphPath{Segments: segs, XMin: xMin, YMin: yMin, XMax: xMax, YMax: yMax}, nil
}

// extractGlyphPath extracts path segments from a TrueType Glyph.
// depth limits composite glyph recursion.
func extractGlyphPath(f *TrueTypeFont, g *Glyph, depth int) []PathSegment {
	if depth > 8 {
		return nil
	}
	if g.simpleGlyph != nil {
		return extractSimplePath(g.simpleGlyph)
	}
	if g.compositeGlyph != nil {
		return extractCompositePath(f, g.compositeGlyph, depth)
	}
	return nil
}

// extractSimplePath converts a TrueType simple glyph to path segments.
// Contours use the on-curve/off-curve point convention:
//   - On-curve → LineTo
//   - Off-curve + On-curve → QuadTo
//   - Off-curve + Off-curve → QuadTo with implicit midpoint
func extractSimplePath(sg *SimpleGlyph) []PathSegment {
	if sg == nil || len(sg.flags) == 0 {
		return nil
	}

	xs := sg.xCoordinates
	ys := sg.yCoordinates
	flags := sg.flags
	nPts := len(flags)

	var segs []PathSegment
	contourStart := 0
	for _, endPt := range sg.endPtsOfContours {
		contourEnd := int(endPt)
		if contourEnd >= nPts {
			break
		}
		n := contourEnd - contourStart + 1

		idx := func(i int) int { return contourStart + ((i - contourStart + n) % n) }
		onCurve := func(i int) bool { return flags[i]&glyphFlagOnCurve != 0 }

		// Find starting on-curve point, or create midpoint of two off-curve
		start := -1
		for i := contourStart; i <= contourEnd; i++ {
			if onCurve(i) {
				start = i
				break
			}
		}

		var penX, penY float32
		if start >= 0 {
			penX = float32(xs[start])
			penY = float32(ys[start])
			segs = append(segs, PathSegment{Op: OpMoveTo, Args: [6]float32{penX, penY}})
		} else {
			fi := contourStart
			li := contourEnd
			mx := int16((int32(xs[fi]) + int32(xs[li])) / 2)
			my := int16((int32(ys[fi]) + int32(ys[li])) / 2)
			penX, penY = float32(mx), float32(my)
			segs = append(segs, PathSegment{Op: OpMoveTo, Args: [6]float32{penX, penY}})
			start = contourStart
		}

		i := start
		for step := 0; step < n; step++ {
			j := idx(i + 1)
			if onCurve(j) {
				penX = float32(xs[j])
				penY = float32(ys[j])
				segs = append(segs, PathSegment{Op: OpLineTo, Args: [6]float32{penX, penY}})
				i = j
			} else {
				cpX := float32(xs[j])
				cpY := float32(ys[j])
				k := idx(j + 1)
				if onCurve(k) {
					penX = float32(xs[k])
					penY = float32(ys[k])
					segs = append(segs, PathSegment{Op: OpQuadTo, Args: [6]float32{cpX, cpY, penX, penY}})
					i = k
					step++
				} else {
					mx := int16((int32(xs[j]) + int32(xs[k])) / 2)
					my := int16((int32(ys[j]) + int32(ys[k])) / 2)
					segs = append(segs, PathSegment{Op: OpQuadTo, Args: [6]float32{cpX, cpY, float32(mx), float32(my)}})
					penX = float32(mx)
					penY = float32(my)
					i = j
				}
			}
		}
		contourStart = contourEnd + 1
	}
	return segs
}

// extractCompositePath recursively extracts segments from a composite glyph
// applying the per-component transform.
func extractCompositePath(f *TrueTypeFont, cg *CompositeGlyph, depth int) []PathSegment {
	var segs []PathSegment
	for _, comp := range cg.components {
		subSegs := extractGlyphPath(f, f.glyf[comp.glyphIndex], depth+1)
		if len(subSegs) == 0 {
			continue
		}

		dx := float32(comp.arg1)
		dy := float32(comp.arg2)

		m00, m01, m10, m11 := float32(1.0), float32(0.0), float32(0.0), float32(1.0)
		if comp.hasScale {
			s := f2dot14ToFloat32(comp.transform[0])
			m00, m11 = s, s
		} else if comp.hasXYScale {
			m00 = f2dot14ToFloat32(comp.transform[0])
			m11 = f2dot14ToFloat32(comp.transform[3])
		} else if comp.has2x2 {
			m00 = f2dot14ToFloat32(comp.transform[0])
			m01 = f2dot14ToFloat32(comp.transform[1])
			m10 = f2dot14ToFloat32(comp.transform[2])
			m11 = f2dot14ToFloat32(comp.transform[3])
		}

		transformPt := func(x, y float32) (float32, float32) {
			return m00*x + m01*y + dx, m10*x + m11*y + dy
		}

		for _, seg := range subSegs {
			s := seg
			switch s.Op {
			case OpMoveTo:
				s.Args[0], s.Args[1] = transformPt(s.Args[0], s.Args[1])
			case OpLineTo:
				s.Args[0], s.Args[1] = transformPt(s.Args[0], s.Args[1])
			case OpQuadTo:
				s.Args[0], s.Args[1] = transformPt(s.Args[0], s.Args[1])
				s.Args[2], s.Args[3] = transformPt(s.Args[2], s.Args[3])
			case OpCubeTo:
				s.Args[0], s.Args[1] = transformPt(s.Args[0], s.Args[1])
				s.Args[2], s.Args[3] = transformPt(s.Args[2], s.Args[3])
				s.Args[4], s.Args[5] = transformPt(s.Args[4], s.Args[5])
			}
			segs = append(segs, s)
		}
	}
	return segs
}

// f2dot14ToFloat32 converts an F2Dot14 fixed-point value to float32.
func f2dot14ToFloat32(v int16) float32 {
	return float32(v) / 16384.0
}

// pathBBox computes the bounding box of a set of path segments.
func pathBBox(segs []PathSegment) (xMin, yMin, xMax, yMax float32) {
	first := true
	for _, seg := range segs {
		var pts [][2]float32
		switch seg.Op {
		case OpMoveTo, OpLineTo:
			pts = [][2]float32{{seg.Args[0], seg.Args[1]}}
		case OpQuadTo:
			pts = [][2]float32{
				{seg.Args[0], seg.Args[1]},
				{seg.Args[2], seg.Args[3]},
			}
		case OpCubeTo:
			pts = [][2]float32{
				{seg.Args[0], seg.Args[1]},
				{seg.Args[2], seg.Args[3]},
				{seg.Args[4], seg.Args[5]},
			}
		}
		for _, p := range pts {
			if first {
				xMin, yMin, xMax, yMax = p[0], p[1], p[0], p[1]
				first = false
			} else {
				if p[0] < xMin {
					xMin = p[0]
				}
				if p[1] < yMin {
					yMin = p[1]
				}
				if p[0] > xMax {
					xMax = p[0]
				}
				if p[1] > yMax {
					yMax = p[1]
				}
			}
		}
	}
	return
}
