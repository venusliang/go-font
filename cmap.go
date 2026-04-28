package gofont

import (
	"errors"
	"sort"
)

type CMap struct {
	version         uint16
	numTables       uint16
	encodingRecords []encodingRecord
	subtables       []CMapSubtable
}

type encodingRecord struct {
	platformID     uint16
	encodingID     uint16
	subtableOffset uint32
}

type CMapSubtable interface {
	Format() uint16
	Map(rune) uint16
	Enumerate(func(rune, uint16))
}

// Format 0: byte encoding table
type CMapFormat0 struct {
	language     uint16
	glyphIdArray [256]uint8
}

func (f *CMapFormat0) Format() uint16 { return 0 }
func (f *CMapFormat0) Map(r rune) uint16 {
	if r < 0 || r > 255 {
		return 0
	}
	return uint16(f.glyphIdArray[r])
}
func (f *CMapFormat0) Enumerate(fn func(rune, uint16)) {
	for i := 0; i < 256; i++ {
		if f.glyphIdArray[i] != 0 {
			fn(rune(i), uint16(f.glyphIdArray[i]))
		}
	}
}

// Format 4: segment mapping to delta values
type CMapFormat4 struct {
	language      uint16
	segCount      uint16
	endCode       []uint16
	startCode     []uint16
	idDelta       []int16
	idRangeOffset []uint16
	glyphIdArray  []uint16
}

func (f *CMapFormat4) Format() uint16 { return 4 }
func (f *CMapFormat4) Map(r rune) uint16 {
	code := uint16(r)
	for i := 0; i < int(f.segCount); i++ {
		if code <= f.endCode[i] {
			if code < f.startCode[i] {
				return 0
			}
			if f.idRangeOffset[i] == 0 {
				return (uint16(int16(code)+f.idDelta[i]) & 0xFFFF)
			}
			// idRangeOffset lookup
			offset := int(f.idRangeOffset[i]) + (int(code)-int(f.startCode[i]))*2
			idx := offset/2 - (int(f.segCount) - i)
			if idx >= 0 && idx < len(f.glyphIdArray) {
				gid := f.glyphIdArray[idx]
				if gid != 0 {
					return (uint16(int16(gid)+f.idDelta[i]) & 0xFFFF)
				}
			}
			return 0
		}
	}
	return 0
}
func (f *CMapFormat4) Enumerate(fn func(rune, uint16)) {
	for i := 0; i < int(f.segCount); i++ {
		for code := uint32(f.startCode[i]); code <= uint32(f.endCode[i]); code++ {
			var gid uint16
			c := uint16(code)
			if f.idRangeOffset[i] == 0 {
				gid = uint16(int16(c)+f.idDelta[i]) & 0xFFFF
			} else {
				offset := int(f.idRangeOffset[i]) + (int(c)-int(f.startCode[i]))*2
				idx := offset/2 - (int(f.segCount) - i)
				if idx >= 0 && idx < len(f.glyphIdArray) {
					gid = f.glyphIdArray[idx]
					if gid != 0 {
						gid = uint16(int16(gid)+f.idDelta[i]) & 0xFFFF
					}
				}
			}
			if gid != 0 {
				fn(rune(code), gid)
			}
		}
	}
}

// Format 6: trimmed table mapping
type CMapFormat6 struct {
	language  uint16
	firstCode uint16
	entryCount uint16
	glyphIdArray []uint16
}

func (f *CMapFormat6) Format() uint16 { return 6 }
func (f *CMapFormat6) Map(r rune) uint16 {
	code := uint16(r)
	if code < f.firstCode || code >= f.firstCode+f.entryCount {
		return 0
	}
	return f.glyphIdArray[code-f.firstCode]
}
func (f *CMapFormat6) Enumerate(fn func(rune, uint16)) {
	for i := uint16(0); i < f.entryCount; i++ {
		gid := f.glyphIdArray[i]
		if gid != 0 {
			fn(rune(f.firstCode+i), gid)
		}
	}
}

// Format 12: segmented coverage (32-bit)
type SequentialMapGroup struct {
	startCharCode uint32
	endCharCode   uint32
	startGlyphID  uint32
}

type CMapFormat12 struct {
	language uint32
	numGroups uint32
	groups   []SequentialMapGroup
}

func (f *CMapFormat12) Format() uint16 { return 12 }
func (f *CMapFormat12) Map(r rune) uint16 {
	code := uint32(r)
	for _, g := range f.groups {
		if code >= g.startCharCode && code <= g.endCharCode {
			return uint16(g.startGlyphID + (code - g.startCharCode))
		}
	}
	return 0
}
func (f *CMapFormat12) Enumerate(fn func(rune, uint16)) {
	for _, g := range f.groups {
		for code := g.startCharCode; code <= g.endCharCode; code++ {
			gid := uint16(g.startGlyphID + (code - g.startCharCode))
			if gid != 0 {
				fn(rune(code), gid)
			}
		}
	}
}

func parseCmap(dir TableDirectory, data []byte) (cmap *CMap, err error) {
	if len(data) < 4 {
		return nil, errors.New("cmap table too small")
	}

	binary := BinaryFrom(data, false)

	cmap = &CMap{
		version:         binary.U16(),
		numTables:       binary.U16(),
		encodingRecords: make([]encodingRecord, 0, binary.Offset()/8),
	}

	for i := 0; i < int(cmap.numTables); i++ {
		record := encodingRecord{
			platformID:     binary.U16(),
			encodingID:     binary.U16(),
			subtableOffset: binary.U32(),
		}
		cmap.encodingRecords = append(cmap.encodingRecords, record)
	}

	// Parse each subtable
	for _, record := range cmap.encodingRecords {
		if int(record.subtableOffset) >= len(data) {
			continue
		}
		subBinary := BinaryFrom(data[record.subtableOffset:], false)
		format := subBinary.U16()

		var subtable CMapSubtable
		switch format {
		case 0:
			subtable = parseCmapFormat0(subBinary)
		case 4:
			subtable, err = parseCmapFormat4(subBinary)
		case 6:
			subtable = parseCmapFormat6(subBinary)
		case 12:
			subtable, err = parseCmapFormat12(subBinary)
		}
		if err != nil {
			return
		}
		if subtable != nil {
			cmap.subtables = append(cmap.subtables, subtable)
		}
	}

	return
}

func parseCmapFormat0(binary Binary) *CMapFormat0 {
	f := &CMapFormat0{}
	binary.U16() // length
	f.language = binary.U16()
	for i := 0; i < 256; i++ {
		f.glyphIdArray[i] = binary.U8()
	}
	return f
}

func parseCmapFormat4(binary Binary) (*CMapFormat4, error) {
	f := &CMapFormat4{}
	length := binary.U16() // length (already past format field)
	_ = length
	f.language = binary.U16()
	segCountX2 := binary.U16()
	f.segCount = segCountX2 / 2
	binary.U16() // searchRange
	binary.U16() // entrySelector
	binary.U16() // rangeShift

	f.endCode = make([]uint16, f.segCount)
	for i := 0; i < int(f.segCount); i++ {
		f.endCode[i] = binary.U16()
	}

	binary.U16() // reservedPad

	f.startCode = make([]uint16, f.segCount)
	for i := 0; i < int(f.segCount); i++ {
		f.startCode[i] = binary.U16()
	}

	f.idDelta = make([]int16, f.segCount)
	for i := 0; i < int(f.segCount); i++ {
		f.idDelta[i] = binary.I16()
	}

	idRangeOffsetPos := binary.Offset()
	f.idRangeOffset = make([]uint16, f.segCount)
	for i := 0; i < int(f.segCount); i++ {
		f.idRangeOffset[i] = binary.U16()
	}

	// Remaining bytes are the glyphIdArray
	remaining := binary.Bytes(binary.Offset()) // just peek
	if len(remaining) > 0 {
		// We need to read the glyphIdArray from after the idRangeOffset array
		// The offset calculations in Map() are relative to idRangeOffset positions
		// Store the remaining data as glyphIdArray
		glyphArrayBinary := BinaryFrom(remaining, false)
		count := len(remaining) / 2
		f.glyphIdArray = make([]uint16, count)
		for i := 0; i < count; i++ {
			f.glyphIdArray[i] = glyphArrayBinary.U16()
		}
	}

	_ = idRangeOffsetPos
	return f, nil
}

func parseCmapFormat6(binary Binary) *CMapFormat6 {
	f := &CMapFormat6{}
	f.language = binary.U16()
	f.firstCode = binary.U16()
	f.entryCount = binary.U16()
	f.glyphIdArray = make([]uint16, f.entryCount)
	for i := 0; i < int(f.entryCount); i++ {
		f.glyphIdArray[i] = binary.U16()
	}
	return f
}

func parseCmapFormat12(binary Binary) (*CMapFormat12, error) {
	f := &CMapFormat12{}
	binary.U16() // reserved
	f.language = binary.U32()
	f.numGroups = binary.U32()
	f.groups = make([]SequentialMapGroup, f.numGroups)
	for i := 0; i < int(f.numGroups); i++ {
		f.groups[i] = SequentialMapGroup{
			startCharCode: binary.U32(),
			endCharCode:   binary.U32(),
			startGlyphID:  binary.U32(),
		}
	}
	return f, nil
}

func writeCmap(cmap *CMap) []byte {
	// Serialize each unique subtable
	subtableData := make(map[int][]byte) // index -> serialized bytes
	for i, sub := range cmap.subtables {
		switch s := sub.(type) {
		case *CMapFormat0:
			subtableData[i] = writeCmapFormat0(s)
		case *CMapFormat4:
			subtableData[i] = writeCmapFormat4(s)
		case *CMapFormat6:
			subtableData[i] = writeCmapFormat6(s)
		case *CMapFormat12:
			subtableData[i] = writeCmapFormat12(s)
		}
	}

	// Build offset map: encoding record index -> subtable data offset
	// Encoding records reference subtables by subtableOffset.
	// We assign one subtable per encoding record by matching order.
	headerSize := 4 + len(cmap.encodingRecords)*8

	offsets := make([]uint32, len(cmap.encodingRecords))
	curOffset := uint32(headerSize)
	subIdx := 0
	for i := 0; i < len(cmap.encodingRecords); i++ {
		// Check if this encoding record points to the same offset as a previous one
		duplicate := false
		for j := 0; j < i; j++ {
			if cmap.encodingRecords[j].subtableOffset == cmap.encodingRecords[i].subtableOffset {
				offsets[i] = offsets[j]
				duplicate = true
				break
			}
		}
		if !duplicate {
			offsets[i] = curOffset
			if subIdx < len(subtableData) {
				curOffset += uint32(len(subtableData[subIdx]))
			}
			subIdx++
		}
	}

	data := make([]byte, curOffset)
	binary := BinaryFrom(data, false)

	// Header
	binary.PutU16(cmap.version)
	binary.PutU16(cmap.numTables)

	for i, rec := range cmap.encodingRecords {
		binary.PutU16(rec.platformID)
		binary.PutU16(rec.encodingID)
		binary.PutU32(offsets[i])
	}

	// Write subtables (unique ones only)
	written := make(map[uint32]bool)
	subIdx = 0
	for i := range cmap.encodingRecords {
		if written[offsets[i]] {
			continue
		}
		written[offsets[i]] = true
		if subIdx < len(subtableData) {
			copy(data[offsets[i]:], subtableData[subIdx])
			subIdx++
		}
	}

	return data
}

func writeCmapFormat0(f *CMapFormat0) []byte {
	data := make([]byte, 262)
	binary := BinaryFrom(data, false)
	binary.PutU16(0) // format
	binary.PutU16(262) // length
	binary.PutU16(f.language)
	for i := 0; i < 256; i++ {
		binary.PutU8(f.glyphIdArray[i])
	}
	return data
}

func writeCmapFormat4(f *CMapFormat4) []byte {
	// Calculate length
	glyphArraySize := len(f.glyphIdArray) * 2
	length := 14 + int(f.segCount)*2 + 2 + int(f.segCount)*2 + int(f.segCount)*2 + int(f.segCount)*2 + glyphArraySize

	data := make([]byte, length)
	binary := BinaryFrom(data, false)

	searchRange, entrySelector, rangeShift := calcSearchParams4(int(f.segCount))

	binary.PutU16(4)              // format
	binary.PutU16(uint16(length)) // length
	binary.PutU16(f.language)
	binary.PutU16(f.segCount * 2)
	binary.PutU16(searchRange)
	binary.PutU16(entrySelector)
	binary.PutU16(rangeShift)

	for _, ec := range f.endCode {
		binary.PutU16(ec)
	}
	binary.PutU16(0) // reservedPad
	for _, sc := range f.startCode {
		binary.PutU16(sc)
	}
	for _, d := range f.idDelta {
		binary.PutU16(uint16(d))
	}
	for _, ro := range f.idRangeOffset {
		binary.PutU16(ro)
	}
	for _, gid := range f.glyphIdArray {
		binary.PutU16(gid)
	}

	return data
}

func writeCmapFormat6(f *CMapFormat6) []byte {
	length := 10 + len(f.glyphIdArray)*2
	data := make([]byte, length)
	binary := BinaryFrom(data, false)
	binary.PutU16(6)
	binary.PutU16(uint16(length))
	binary.PutU16(f.language)
	binary.PutU16(f.firstCode)
	binary.PutU16(f.entryCount)
	for _, gid := range f.glyphIdArray {
		binary.PutU16(gid)
	}
	return data
}

func writeCmapFormat12(f *CMapFormat12) []byte {
	length := 16 + int(f.numGroups)*12
	data := make([]byte, length)
	binary := BinaryFrom(data, false)
	binary.PutU16(12) // format (actually uint16 + reserved uint16 = uint32)
	binary.PutU16(0)  // reserved
	binary.PutU32(uint32(length))
	binary.PutU32(f.language)
	binary.PutU32(f.numGroups)
	for _, g := range f.groups {
		binary.PutU32(g.startCharCode)
		binary.PutU32(g.endCharCode)
		binary.PutU32(g.startGlyphID)
	}
	return data
}

// calcSearchParams4 computes searchRange, entrySelector, rangeShift for cmap format 4
func calcSearchParams4(segCount int) (searchRange, entrySelector, rangeShift uint16) {
	entrySelector = 0
	power := 1
	for power*2 <= segCount {
		power *= 2
		entrySelector++
	}
	searchRange = uint16(power * 2)
	rangeShift = uint16(segCount*2) - searchRange
	return
}

type runeGlyphPair struct {
	codePoint rune
	glyphID   uint16
}

// rebuildCmap builds a new binary CMap from an abstract rune→glyph map.
// It tries to preserve the original cmap's encoding records and format
// choices, but rebuilds the subtable data from the map.
func rebuildCmap(runeToGlyphID map[rune]uint16, orig *CMap) *CMap {
	if len(runeToGlyphID) == 0 || orig == nil {
		return orig
	}

	// Collect sorted pairs
	pairs := make([]runeGlyphPair, 0, len(runeToGlyphID))
	for r, gid := range runeToGlyphID {
		if gid != 0 {
			pairs = append(pairs, runeGlyphPair{r, gid})
		}
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].codePoint < pairs[j].codePoint
	})

	if len(pairs) == 0 {
		return orig
	}

	// Determine which format to use: if any codepoint > 0xFFFF, need format 12
	needFormat12 := pairs[len(pairs)-1].codePoint > 0xFFFF

	newCmap := &CMap{
		version:         orig.version,
		numTables:       orig.numTables,
		encodingRecords: orig.encodingRecords,
	}

	if needFormat12 {
		f12 := buildCmapFormat12(pairs)
		// Also build format 4 for BMP subset
		bmpPairs := make([]runeGlyphPair, 0)
		for _, p := range pairs {
			if p.codePoint <= 0xFFFF {
				bmpPairs = append(bmpPairs, p)
			}
		}
		f4 := buildCmapFormat4(bmpPairs)
		newCmap.subtables = []CMapSubtable{f4, f12}
	} else {
		f4 := buildCmapFormat4(pairs)
		newCmap.subtables = []CMapSubtable{f4}
	}

	return newCmap
}

// buildCmapFormat4 builds a Format 4 subtable from sorted rune-glyph pairs.
// Uses a simple segment construction: one segment per consecutive range
// where glyph IDs are also consecutive (linear), or per individual pair.
func buildCmapFormat4(pairs []runeGlyphPair) *CMapFormat4 {
	if len(pairs) == 0 {
		return &CMapFormat4{segCount: 1, endCode: []uint16{0xFFFF}, startCode: []uint16{0xFFFF}, idDelta: []int16{1}, idRangeOffset: []uint16{0}}
	}

	type segment struct {
		startCode uint16
		endCode   uint16
		glyphIDs  []uint16 // non-nil means use idRangeOffset, nil means use idDelta
		delta     int16
	}

	var segments []segment
	i := 0
	for i < len(pairs) {
		start := i
		// Try to build a linear segment (consecutive codepoints mapping to consecutive glyph IDs)
		for i+1 < len(pairs) &&
			uint16(pairs[i+1].codePoint) == uint16(pairs[i].codePoint)+1 &&
			pairs[i+1].glyphID == pairs[i].glyphID+1 {
			i++
		}

		startCode := uint16(pairs[start].codePoint)
		endCode := uint16(pairs[i].codePoint)
		startGlyph := pairs[start].glyphID

		if start == i {
			// Single entry — check if delta works
			delta := int16(startGlyph) - int16(startCode)
			segments = append(segments, segment{
				startCode: startCode,
				endCode:   endCode,
				delta:     delta,
			})
		} else {
			// Range of consecutive mappings — use delta
			delta := int16(startGlyph) - int16(startCode)
			segments = append(segments, segment{
				startCode: startCode,
				endCode:   endCode,
				delta:     delta,
			})
		}
		i++
	}

	// Add sentinel segment (0xFFFF)
	segments = append(segments, segment{
		startCode: 0xFFFF,
		endCode:   0xFFFF,
		delta:     1,
	})

	segCount := len(segments)
	f4 := &CMapFormat4{
		segCount:     uint16(segCount),
		endCode:      make([]uint16, segCount),
		startCode:    make([]uint16, segCount),
		idDelta:      make([]int16, segCount),
		idRangeOffset: make([]uint16, segCount),
		language:     0,
	}

	for j, seg := range segments {
		f4.startCode[j] = seg.startCode
		f4.endCode[j] = seg.endCode
		f4.idDelta[j] = seg.delta
		f4.idRangeOffset[j] = 0
	}

	return f4
}

// buildCmapFormat12 builds a Format 12 subtable from sorted rune-glyph pairs.
func buildCmapFormat12(pairs []runeGlyphPair) *CMapFormat12 {
	if len(pairs) == 0 {
		return &CMapFormat12{}
	}

	type group struct {
		startCharCode uint32
		endCharCode   uint32
		startGlyphID  uint32
	}

	var groups []group
	i := 0
	for i < len(pairs) {
		start := i
		for i+1 < len(pairs) &&
			uint32(pairs[i+1].codePoint) == uint32(pairs[i].codePoint)+1 &&
			pairs[i+1].glyphID == pairs[i].glyphID+1 {
			i++
		}
		groups = append(groups, group{
			startCharCode: uint32(pairs[start].codePoint),
			endCharCode:   uint32(pairs[i].codePoint),
			startGlyphID:  uint32(pairs[start].glyphID),
		})
		i++
	}

	f12 := &CMapFormat12{
		language:  0,
		numGroups: uint32(len(groups)),
		groups:    make([]SequentialMapGroup, len(groups)),
	}
	for j, g := range groups {
		f12.groups[j] = SequentialMapGroup{
			startCharCode: g.startCharCode,
			endCharCode:   g.endCharCode,
			startGlyphID:  g.startGlyphID,
		}
	}
	return f12
}
