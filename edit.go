package gofont

import (
	"errors"
	"fmt"
	"sort"
)

// ensureRuneMap lazily initializes the runeToGlyphID map from parsed cmap data.
func (ttf *TrueTypeFont) ensureRuneMap() {
	if ttf.runeToGlyphID != nil {
		return
	}
	ttf.runeToGlyphID = make(map[rune]uint16)
	if ttf.cmap == nil {
		return
	}
	for _, sub := range ttf.cmap.subtables {
		sub.Enumerate(func(r rune, gid uint16) {
			ttf.runeToGlyphID[r] = gid
		})
	}
}

// RuneToGlyphID returns the glyph ID for a Unicode code point, or 0 if not mapped.
func (ttf *TrueTypeFont) RuneToGlyphID(r rune) uint16 {
	ttf.ensureRuneMap()
	return ttf.runeToGlyphID[r]
}

// GlyphForRune returns the glyph data for a Unicode code point, or nil if not mapped.
func (ttf *TrueTypeFont) GlyphForRune(r rune) *Glyph {
	gid := ttf.RuneToGlyphID(r)
	if gid == 0 {
		return nil
	}
	if int(gid) >= len(ttf.glyf) {
		return nil
	}
	return ttf.glyf[gid]
}

// SetRuneMapping maps a Unicode code point to the given glyph ID.
// Returns an error if glyphID is out of range.
func (ttf *TrueTypeFont) SetRuneMapping(r rune, glyphID uint16) error {
	ttf.ensureRuneMap()
	if int(glyphID) >= len(ttf.glyf) {
		return errors.New("glyph ID out of range")
	}
	ttf.runeToGlyphID[r] = glyphID
	return nil
}

// RemoveRuneMapping removes the mapping for a Unicode code point.
func (ttf *TrueTypeFont) RemoveRuneMapping(r rune) {
	ttf.ensureRuneMap()
	delete(ttf.runeToGlyphID, r)
}

// NumGlyphs returns the number of glyphs in the font.
func (ttf *TrueTypeFont) NumGlyphs() int {
	return len(ttf.glyf)
}

// GlyphAt returns the glyph at the given index, or nil if out of range.
func (ttf *TrueTypeFont) GlyphAt(index int) *Glyph {
	if index < 0 || index >= len(ttf.glyf) {
		return nil
	}
	return ttf.glyf[index]
}

// SetGlyphAt replaces the glyph data at the given index.
// Returns an error if the index is out of range.
func (ttf *TrueTypeFont) SetGlyphAt(index int, g *Glyph) error {
	if index < 0 || index >= len(ttf.glyf) {
		return errors.New("glyph index out of range")
	}
	ttf.glyf[index] = g
	ttf.recalcMaxp()
	return nil
}

// RemoveGlyphs removes glyphs at the given indices and compacts all related tables.
// Returns a remap table: old index → new index. Indices not in the map were removed.
// Glyph 0 (.notdef) cannot be removed.
func (ttf *TrueTypeFont) RemoveGlyphs(indices []int) (remap map[int]int, err error) {
	if len(indices) == 0 {
		return nil, nil
	}

	remove := make(map[int]bool)
	for _, idx := range indices {
		if idx == 0 {
			return nil, errors.New("cannot remove glyph 0 (.notdef)")
		}
		if idx < 0 || idx >= len(ttf.glyf) {
			return nil, errors.New("glyph index out of range")
		}
		remove[idx] = true
	}

	// Build remap: old index → new index
	remap = make(map[int]int, len(ttf.glyf)-len(remove))
	newIdx := 0
	for oldIdx := 0; oldIdx < len(ttf.glyf); oldIdx++ {
		if !remove[oldIdx] {
			remap[oldIdx] = newIdx
			newIdx++
		}
	}

	// Compact glyphs
	newGlyphs := make([]*Glyph, 0, newIdx)
	for oldIdx := 0; oldIdx < len(ttf.glyf); oldIdx++ {
		if !remove[oldIdx] {
			newGlyphs = append(newGlyphs, ttf.glyf[oldIdx])
		}
	}

	// Compact hmtx
	var newHmtx Hmtx
	if ttf.hmtx != nil {
		newHmtx.hMetrics = make([]LongHorMetric, 0, newIdx)
		newHmtx.leftSideBearing = make([]int16, 0)
		numHMetrics := len(ttf.hmtx.hMetrics)

		for oldIdx := 0; oldIdx < len(ttf.glyf); oldIdx++ {
			if remove[oldIdx] {
				continue
			}
			if oldIdx < numHMetrics {
				newHmtx.hMetrics = append(newHmtx.hMetrics, ttf.hmtx.hMetrics[oldIdx])
			} else {
				newHmtx.leftSideBearing = append(newHmtx.leftSideBearing, ttf.hmtx.leftSideBearing[oldIdx-numHMetrics])
			}
		}
	}

	// Fix composite glyph component references
	for _, g := range newGlyphs {
		if g != nil && g.compositeGlyph != nil {
			for j := range g.compositeGlyph.components {
				oldGID := int(g.compositeGlyph.components[j].glyphIndex)
				if newGID, ok := remap[oldGID]; ok {
					g.compositeGlyph.components[j].glyphIndex = uint16(newGID)
				}
			}
		}
	}

	ttf.glyf = newGlyphs
	ttf.hmtx = &newHmtx
	if ttf.maxp != nil {
		ttf.maxp.numGlyphs = uint16(len(newGlyphs))
	}
	if ttf.hhea != nil {
		ttf.hhea.numberOfHMetrics = uint16(len(newHmtx.hMetrics))
	}

	// Update post table
	if ttf.post != nil && ttf.post.version == 0x00020000 {
		newGNI := make([]uint16, len(newGlyphs))
		for oldIdx, newIdx := range remap {
			if oldIdx < len(ttf.post.glyphNameIndex) {
				newGNI[newIdx] = ttf.post.glyphNameIndex[oldIdx]
			}
		}
		ttf.post.glyphNameIndex = newGNI
		ttf.post.numGlyphs = uint16(len(newGlyphs))
	}

	// Update runeToGlyphID map
	ttf.ensureRuneMap()
	for r, oldGID := range ttf.runeToGlyphID {
		if newGID, ok := remap[int(oldGID)]; ok {
			ttf.runeToGlyphID[r] = uint16(newGID)
		} else {
			delete(ttf.runeToGlyphID, r)
		}
	}

	// Recalculate global bounding box in head
	ttf.recalcHeadBBox()

	// Update OS/2 character range
	ttf.updateOS2CharRange()

	ttf.recalcMaxp()
	return
}

// recalcHeadBBox recalculates the global font bounding box in the head table
// from the current glyph data.
func (ttf *TrueTypeFont) recalcHeadBBox() {
	if ttf.head == nil || len(ttf.glyf) == 0 {
		return
	}
	first := true
	for _, g := range ttf.glyf {
		if g == nil {
			continue
		}
		// Skip empty glyphs (numberOfContours == 0 with no data)
		if g.header.numberOfContours == 0 && g.simpleGlyph == nil && g.compositeGlyph == nil {
			continue
		}
		if first {
			ttf.head.xMin = g.header.xMin
			ttf.head.yMin = g.header.yMin
			ttf.head.xMax = g.header.xMax
			ttf.head.yMax = g.header.yMax
			first = false
		} else {
			if g.header.xMin < ttf.head.xMin {
				ttf.head.xMin = g.header.xMin
			}
			if g.header.yMin < ttf.head.yMin {
				ttf.head.yMin = g.header.yMin
			}
			if g.header.xMax > ttf.head.xMax {
				ttf.head.xMax = g.header.xMax
			}
			if g.header.yMax > ttf.head.yMax {
				ttf.head.yMax = g.header.yMax
			}
		}
	}
}

// updateOS2CharRange updates the usFirstCharIndex and usLastCharIndex fields
// in the OS/2 table based on the current rune mappings.
func (ttf *TrueTypeFont) updateOS2CharRange() {
	if ttf.os2 == nil || len(ttf.runeToGlyphID) == 0 {
		return
	}
	first := true
	for r := range ttf.runeToGlyphID {
		u := uint16(r)
		if u == 0 {
			continue
		}
		if first {
			ttf.os2.usFirstCharIndex = u
			ttf.os2.usLastCharIndex = u
			first = false
		} else {
			if u < ttf.os2.usFirstCharIndex {
				ttf.os2.usFirstCharIndex = u
			}
			if u > ttf.os2.usLastCharIndex {
				ttf.os2.usLastCharIndex = u
			}
		}
	}
}

// recalcMaxp recalculates maxp statistics from current glyph data.
func (ttf *TrueTypeFont) recalcMaxp() {
	if ttf.maxp == nil {
		return
	}
	var maxPts, maxContours, maxCompPts, maxCompContours uint16
	var maxCompDepth uint16

	for _, g := range ttf.glyf {
		if g == nil {
			continue
		}
		if g.simpleGlyph != nil {
			nPts := uint16(len(g.simpleGlyph.xCoordinates))
			nContours := uint16(len(g.simpleGlyph.endPtsOfContours))
			if nPts > maxPts {
				maxPts = nPts
			}
			if nContours > maxContours {
				maxContours = nContours
			}
		}
		if g.compositeGlyph != nil {
			var compPts, compContours uint16
			for _, c := range g.compositeGlyph.components {
				compGID := int(c.glyphIndex)
				if compGID < len(ttf.glyf) && ttf.glyf[compGID] != nil && ttf.glyf[compGID].simpleGlyph != nil {
					sg := ttf.glyf[compGID].simpleGlyph
					compPts += uint16(len(sg.xCoordinates))
					compContours += uint16(len(sg.endPtsOfContours))
				}
			}
			if compPts > maxCompPts {
				maxCompPts = compPts
			}
			if compContours > maxCompContours {
				maxCompContours = compContours
			}
			for _, c := range g.compositeGlyph.components {
				if depth := compositeDepth(ttf.glyf, c.glyphIndex, 0); depth > maxCompDepth {
					maxCompDepth = depth
				}
			}
		}
	}

	ttf.maxp.maxPoints = maxPts
	ttf.maxp.maxContours = maxContours
	ttf.maxp.maxCompositePoints = maxCompPts
	ttf.maxp.maxCompositeContours = maxCompContours
	ttf.maxp.maxComponentDepth = maxCompDepth
}

// compositeDepth returns the nesting depth of a composite glyph reference.
func compositeDepth(glyphs []*Glyph, glyphIndex uint16, visited uint8) uint16 {
	if visited > 32 || int(glyphIndex) >= len(glyphs) {
		return 0
	}
	g := glyphs[glyphIndex]
	if g == nil || g.compositeGlyph == nil {
		return 0
	}
	var maxChild uint16
	for _, c := range g.compositeGlyph.components {
		d := compositeDepth(glyphs, c.glyphIndex, visited+1)
		if d > maxChild {
			maxChild = d
		}
	}
	return 1 + maxChild
}

// RuneMappings returns all rune-to-glyph mappings sorted by rune value.
func (ttf *TrueTypeFont) RuneMappings() []struct {
	Rune    rune
	GlyphID uint16
} {
	ttf.ensureRuneMap()
	result := make([]struct {
		Rune    rune
		GlyphID uint16
	}, 0, len(ttf.runeToGlyphID))
	for r, gid := range ttf.runeToGlyphID {
		result = append(result, struct {
			Rune    rune
			GlyphID uint16
		}{r, gid})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Rune < result[j].Rune
	})
	return result
}

// --- P1: Basic Query Methods ---

// UnitsPerEm returns the font's units per em value (design space size).
func (ttf *TrueTypeFont) UnitsPerEm() uint16 {
	if ttf.head == nil {
		return 0
	}
	return ttf.head.unitsPerEm
}

// FontBBox returns the global font bounding box (min/max across all glyphs).
func (ttf *TrueTypeFont) FontBBox() (xMin, yMin, xMax, yMax int16) {
	if ttf.head == nil {
		return 0, 0, 0, 0
	}
	return ttf.head.xMin, ttf.head.yMin, ttf.head.xMax, ttf.head.yMax
}

// Ascent returns the font ascent value from hhea table.
func (ttf *TrueTypeFont) Ascent() int16 {
	if ttf.hhea == nil {
		return 0
	}
	return ttf.hhea.ascent
}

// Descent returns the font descent value from hhea table.
func (ttf *TrueTypeFont) Descent() int16 {
	if ttf.hhea == nil {
		return 0
	}
	return ttf.hhea.descent
}

// AdvanceWidth returns the advance width for the given glyph ID.
func (ttf *TrueTypeFont) AdvanceWidth(glyphID uint16) uint16 {
	if ttf.hmtx == nil || int(glyphID) >= len(ttf.glyf) {
		return 0
	}
	numHMetrics := len(ttf.hmtx.hMetrics)
	if int(glyphID) < numHMetrics {
		return ttf.hmtx.hMetrics[glyphID].advanceWidth
	}
	// Glyphs beyond hMetrics share the last advance width
	if numHMetrics > 0 {
		return ttf.hmtx.hMetrics[numHMetrics-1].advanceWidth
	}
	return 0
}

// LeftSideBearing returns the left side bearing for the given glyph ID.
func (ttf *TrueTypeFont) LeftSideBearing(glyphID uint16) int16 {
	if ttf.hmtx == nil || int(glyphID) >= len(ttf.glyf) {
		return 0
	}
	numHMetrics := len(ttf.hmtx.hMetrics)
	if int(glyphID) < numHMetrics {
		return ttf.hmtx.hMetrics[glyphID].lsb
	}
	idx := int(glyphID) - numHMetrics
	if idx < len(ttf.hmtx.leftSideBearing) {
		return ttf.hmtx.leftSideBearing[idx]
	}
	return 0
}

// AdvanceWidthForRune returns the advance width for the glyph mapped to the given rune.
func (ttf *TrueTypeFont) AdvanceWidthForRune(r rune) uint16 {
	gid := ttf.RuneToGlyphID(r)
	if gid == 0 {
		return 0
	}
	return ttf.AdvanceWidth(gid)
}

// IsSimpleGlyph reports whether the glyph at index is a simple glyph.
func (ttf *TrueTypeFont) IsSimpleGlyph(index int) bool {
	if index < 0 || index >= len(ttf.glyf) || ttf.glyf[index] == nil {
		return false
	}
	return ttf.glyf[index].simpleGlyph != nil
}

// IsCompositeGlyph reports whether the glyph at index is a composite glyph.
func (ttf *TrueTypeFont) IsCompositeGlyph(index int) bool {
	if index < 0 || index >= len(ttf.glyf) || ttf.glyf[index] == nil {
		return false
	}
	return ttf.glyf[index].compositeGlyph != nil
}

// GlyphBBox returns the bounding box of the glyph at index.
// Returns ok=false if the index is out of range.
func (ttf *TrueTypeFont) GlyphBBox(index int) (xMin, yMin, xMax, yMax int16, ok bool) {
	if index < 0 || index >= len(ttf.glyf) || ttf.glyf[index] == nil {
		return 0, 0, 0, 0, false
	}
	h := ttf.glyf[index].header
	return h.xMin, h.yMin, h.xMax, h.yMax, true
}

// PointCount returns the number of points in the glyph at index.
// Returns 0 for composite glyphs or out-of-range indices.
func (ttf *TrueTypeFont) PointCount(index int) int {
	if index < 0 || index >= len(ttf.glyf) || ttf.glyf[index] == nil {
		return 0
	}
	if ttf.glyf[index].simpleGlyph != nil {
		return len(ttf.glyf[index].simpleGlyph.xCoordinates)
	}
	return 0
}

// ContourCount returns the number of contours in the glyph at index.
// Returns 0 for composite glyphs or out-of-range indices.
func (ttf *TrueTypeFont) ContourCount(index int) int {
	if index < 0 || index >= len(ttf.glyf) || ttf.glyf[index] == nil {
		return 0
	}
	if ttf.glyf[index].simpleGlyph != nil {
		return len(ttf.glyf[index].simpleGlyph.endPtsOfContours)
	}
	return 0
}

// nameStringForID returns the name string for a given nameID from the name table.
// It prefers platform 3 (Windows) UTF-16BE strings, falls back to platform 1 (Mac).
func (ttf *TrueTypeFont) nameStringForID(nameID uint16) string {
	if ttf.name == nil {
		return ""
	}
	// Try Windows platform (3) with Unicode encoding (1) — UTF-16BE
	for _, rec := range ttf.name.nameRecords {
		if rec.nameID == nameID && rec.platformID == 3 && rec.encodingID == 1 {
			return decodeUTF16BE(ttf.name.stringStorage[rec.offset : rec.offset+rec.length])
		}
	}
	// Fallback: Mac Roman (platform 1, encoding 0)
	for _, rec := range ttf.name.nameRecords {
		if rec.nameID == nameID && rec.platformID == 1 && rec.encodingID == 0 {
			return string(ttf.name.stringStorage[rec.offset : rec.offset+rec.length])
		}
	}
	return ""
}

// decodeUTF16BE decodes a big-endian UTF-16 byte slice to a Go string.
func decodeUTF16BE(data []byte) string {
	runes := make([]rune, 0, len(data)/2)
	for i := 0; i+1 < len(data); i += 2 {
		r := rune(data[i])<<8 | rune(data[i+1])
		runes = append(runes, r)
	}
	return string(runes)
}

// FontFamily returns the font family name (nameID 1).
func (ttf *TrueTypeFont) FontFamily() string {
	return ttf.nameStringForID(1)
}

// FontFullName returns the full font name (nameID 4).
func (ttf *TrueTypeFont) FontFullName() string {
	return ttf.nameStringForID(4)
}

// --- P2: Edit Operations ---

// SetAdvanceWidth sets the advance width for the given glyph ID.
func (ttf *TrueTypeFont) SetAdvanceWidth(glyphID uint16, width uint16) error {
	if ttf.hmtx == nil || int(glyphID) >= len(ttf.glyf) {
		return errors.New("glyph ID out of range")
	}
	numHMetrics := len(ttf.hmtx.hMetrics)
	if int(glyphID) < numHMetrics {
		ttf.hmtx.hMetrics[glyphID].advanceWidth = width
	} else {
		// All glyphs beyond hMetrics share the last entry's advance width
		ttf.hmtx.hMetrics[numHMetrics-1].advanceWidth = width
	}
	return nil
}

// SetLeftSideBearing sets the left side bearing for the given glyph ID.
func (ttf *TrueTypeFont) SetLeftSideBearing(glyphID uint16, lsb int16) error {
	if ttf.hmtx == nil || int(glyphID) >= len(ttf.glyf) {
		return errors.New("glyph ID out of range")
	}
	numHMetrics := len(ttf.hmtx.hMetrics)
	if int(glyphID) < numHMetrics {
		ttf.hmtx.hMetrics[glyphID].lsb = lsb
	} else {
		idx := int(glyphID) - numHMetrics
		if idx < len(ttf.hmtx.leftSideBearing) {
			ttf.hmtx.leftSideBearing[idx] = lsb
		} else {
			return errors.New("leftSideBearing index out of range")
		}
	}
	return nil
}

// setNameForID updates the name string for a given nameID.
// It updates the first matching record in platform 3 (Windows UTF-16BE)
// or creates a new record if none exists.
func (ttf *TrueTypeFont) setNameForID(nameID uint16, name string) {
	if ttf.name == nil {
		return
	}

	// Encode name as UTF-16BE
	encoded := encodeUTF16BE(name)

	// Try to find and update an existing platform 3, encoding 1 record
	for i := range ttf.name.nameRecords {
		rec := &ttf.name.nameRecords[i]
		if rec.nameID == nameID && rec.platformID == 3 && rec.encodingID == 1 {
			// Replace the string in stringStorage
			// Build new storage: old before + new + old after
			prefix := ttf.name.stringStorage[:rec.offset]
			suffix := ttf.name.stringStorage[rec.offset+rec.length:]
			ttf.name.stringStorage = append(append(prefix, encoded...), suffix...)

			// Fix offsets for subsequent records
			delta := int(len(encoded)) - int(rec.length)
			rec.length = uint16(len(encoded))
			for j := range ttf.name.nameRecords {
				if ttf.name.nameRecords[j].offset > rec.offset {
					ttf.name.nameRecords[j].offset += uint16(delta)
				}
			}
			return
		}
	}

	// No existing record — append a new one
	offset := uint16(len(ttf.name.stringStorage))
	ttf.name.stringStorage = append(ttf.name.stringStorage, encoded...)
	ttf.name.nameRecords = append(ttf.name.nameRecords, NameRecord{
		platformID: 3,
		encodingID: 1,
		languageID: 0x0409, // English US
		nameID:     nameID,
		length:     uint16(len(encoded)),
		offset:     offset,
	})
	ttf.name.count = uint16(len(ttf.name.nameRecords))
}

// encodeUTF16BE encodes a Go string to big-endian UTF-16 bytes.
func encodeUTF16BE(s string) []byte {
	out := make([]byte, 0, len(s)*2)
	for _, r := range s {
		// BMP only — surrogates not handled
		if r <= 0xFFFF {
			out = append(out, byte(r>>8), byte(r))
		}
	}
	return out
}

// SetFontFamily sets the font family name (nameID 1).
func (ttf *TrueTypeFont) SetFontFamily(name string) {
	ttf.setNameForID(1, name)
}

// SetFontFullName sets the full font name (nameID 4).
func (ttf *TrueTypeFont) SetFontFullName(name string) {
	ttf.setNameForID(4, name)
}

// TranslateGlyph shifts all coordinates of the glyph at index by (dx, dy).
func (ttf *TrueTypeFont) TranslateGlyph(index int, dx, dy int16) error {
	if index < 0 || index >= len(ttf.glyf) || ttf.glyf[index] == nil {
		return errors.New("glyph index out of range")
	}
	g := ttf.glyf[index]

	// Update bounding box
	g.header.xMin += dx
	g.header.yMin += dy
	g.header.xMax += dx
	g.header.yMax += dy

	if g.simpleGlyph != nil {
		for i := range g.simpleGlyph.xCoordinates {
			g.simpleGlyph.xCoordinates[i] += dx
		}
		for i := range g.simpleGlyph.yCoordinates {
			g.simpleGlyph.yCoordinates[i] += dy
		}
	}

	return nil
}

// ScaleGlyph scales all coordinates of the glyph at index by factors sx and sy.
// Coordinates are rounded to nearest integer after scaling.
func (ttf *TrueTypeFont) ScaleGlyph(index int, sx, sy float64) error {
	if index < 0 || index >= len(ttf.glyf) || ttf.glyf[index] == nil {
		return errors.New("glyph index out of range")
	}
	g := ttf.glyf[index]

	scaleAndRound := func(v int16, s float64) int16 {
		return int16(float64(v)*s + 0.5)
	}

	g.header.xMin = scaleAndRound(g.header.xMin, sx)
	g.header.yMin = scaleAndRound(g.header.yMin, sy)
	g.header.xMax = scaleAndRound(g.header.xMax, sx)
	g.header.yMax = scaleAndRound(g.header.yMax, sy)

	if g.simpleGlyph != nil {
		for i := range g.simpleGlyph.xCoordinates {
			g.simpleGlyph.xCoordinates[i] = scaleAndRound(g.simpleGlyph.xCoordinates[i], sx)
		}
		for i := range g.simpleGlyph.yCoordinates {
			g.simpleGlyph.yCoordinates[i] = scaleAndRound(g.simpleGlyph.yCoordinates[i], sy)
		}
	}

	ttf.recalcMaxp()
	return nil
}

// AppendGlyph adds a new glyph to the font and returns its index.
func (ttf *TrueTypeFont) AppendGlyph(g *Glyph) (int, error) {
	if g == nil {
		return 0, errors.New("glyph cannot be nil")
	}
	idx := len(ttf.glyf)
	ttf.glyf = append(ttf.glyf, g)

	// Extend hmtx: add a new entry with zero metrics
	if ttf.hmtx != nil {
		numHMetrics := len(ttf.hmtx.hMetrics)
		if numHMetrics > 0 {
			// Add to leftSideBearing array
			ttf.hmtx.leftSideBearing = append(ttf.hmtx.leftSideBearing, 0)
		}
	}

	if ttf.maxp != nil {
		ttf.maxp.numGlyphs = uint16(len(ttf.glyf))
	}

	ttf.recalcMaxp()
	return idx, nil
}

// --- P3: Advanced Operations ---

// Subset removes all glyphs that are not needed by the specified runes.
// Glyph 0 (.notdef) is always kept. Glyphs not referenced by any rune in keepRunes
// are removed. Composite glyph dependencies are automatically preserved.
func (ttf *TrueTypeFont) Subset(keepRunes []rune) error {
	ttf.ensureRuneMap()

	// Collect glyph IDs to keep
	keep := map[int]bool{0: true} // always keep .notdef
	for _, r := range keepRunes {
		if gid := ttf.runeToGlyphID[r]; gid != 0 {
			keep[int(gid)] = true
		}
	}

	// Expand with composite glyph dependencies
	expanded := make(map[int]bool)
	for gid := range keep {
		ttf.collectCompositeDeps(gid, expanded)
	}
	for gid := range expanded {
		keep[gid] = true
	}

	// Build the remove list
	var remove []int
	for i := 0; i < len(ttf.glyf); i++ {
		if !keep[i] {
			remove = append(remove, i)
		}
	}

	_, err := ttf.RemoveGlyphs(remove)
	return err
}

// collectCompositeDeps recursively collects all glyph IDs referenced by
// composite glyphs starting from the given glyph index.
func (ttf *TrueTypeFont) collectCompositeDeps(glyphIndex int, seen map[int]bool) {
	if seen[glyphIndex] || glyphIndex < 0 || glyphIndex >= len(ttf.glyf) {
		return
	}
	seen[glyphIndex] = true
	g := ttf.glyf[glyphIndex]
	if g != nil && g.compositeGlyph != nil {
		for _, comp := range g.compositeGlyph.components {
			ttf.collectCompositeDeps(int(comp.glyphIndex), seen)
		}
	}
}

// CopyGlyph copies the glyph data from srcIndex to dstIndex.
func (ttf *TrueTypeFont) CopyGlyph(srcIndex, dstIndex int) error {
	src := ttf.GlyphAt(srcIndex)
	if src == nil {
		return errors.New("source glyph index out of range")
	}
	return ttf.SetGlyphAt(dstIndex, src)
}

// SetRuneMappings sets multiple rune-to-glyph mappings at once.
func (ttf *TrueTypeFont) SetRuneMappings(mappings map[rune]uint16) error {
	ttf.ensureRuneMap()
	for r, gid := range mappings {
		if int(gid) >= len(ttf.glyf) {
			return fmt.Errorf("glyph ID %d out of range for rune U+%04X", gid, r)
		}
	}
	for r, gid := range mappings {
		ttf.runeToGlyphID[r] = gid
	}
	return nil
}

// MappedRunes returns all Unicode code points that have a glyph mapping.
func (ttf *TrueTypeFont) MappedRunes() []rune {
	ttf.ensureRuneMap()
	runes := make([]rune, 0, len(ttf.runeToGlyphID))
	for r := range ttf.runeToGlyphID {
		runes = append(runes, r)
	}
	sort.Slice(runes, func(i, j int) bool {
		return runes[i] < runes[j]
	})
	return runes
}
