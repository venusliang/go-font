package gofont

import (
	"errors"
	"fmt"
)

// CFF holds parsed data from a CFF (Compact Font Format) table.
type CFF struct {
	header      CFFHeader
	nameIndex   CFFINDEX
	topDict     CFFTopDict
	stringIndex CFFINDEX
	globalSubrs CFFINDEX
	charset     CFFCharset
	charStrings CFFINDEX
	privateDict CFFPrivateDict
	localSubrs  CFFINDEX
	raw         []byte

	// Cached glyph names (lazily populated)
	glyphNames []string
}

// CFFHeader is the CFF table header.
type CFFHeader struct {
	majorVersion uint8
	minorVersion uint8
	hdrSize      uint8
	offSize      uint8
}

// CFFINDEX represents a CFF INDEX structure.
// An INDEX stores a count of items, with offsets into a data region.
// Element i is data[offsets[i]-1 : offsets[i+1]-1] (1-based offsets).
type CFFINDEX struct {
	count   uint16
	offsets []uint32
	data    []byte
}

// CFFTopDict holds key values parsed from the Top DICT.
type CFFTopDict struct {
	charsetOffset  uint32
	charStringsOff uint32
	privateSize    uint32
	privateOffset  uint32
}

// CFFPrivateDict holds Private DICT values.
type CFFPrivateDict struct {
	defaultWidthX   int32
	nominalWidthX   int32
	localSubrOffset uint32
}

// CFFCharset maps glyph IDs to SIDs.
type CFFCharset struct {
	format  uint8
	entries []uint16 // SIDs; glyph 0 is always .notdef (SID 0), entries[0] = glyph 1's SID, etc.
}

// parseCFF parses a CFF table from raw bytes.
func parseCFF(data []byte) (*CFF, error) {
	if len(data) < 4 {
		return nil, errors.New("CFF table too short")
	}

	cff := &CFF{raw: data}
	b := BinaryFrom(data, false)

	// 1. Header
	cff.header.majorVersion = b.U8()
	cff.header.minorVersion = b.U8()
	cff.header.hdrSize = b.U8()
	cff.header.offSize = b.U8()

	if cff.header.majorVersion != 1 {
		return nil, fmt.Errorf("unsupported CFF version: %d", cff.header.majorVersion)
	}

	// 2. Name INDEX
	off := int(cff.header.hdrSize)
	nameIdx, bytesRead, err := parseCFFINDEX(data, off)
	if err != nil {
		return nil, fmt.Errorf("parsing Name INDEX: %w", err)
	}
	cff.nameIndex = nameIdx
	off += bytesRead

	// 3. Top DICT INDEX
	topDictIdx, bytesRead, err := parseCFFINDEX(data, off)
	if err != nil {
		return nil, fmt.Errorf("parsing Top DICT INDEX: %w", err)
	}
	off += bytesRead

	// Parse the Top DICT data (first font)
	if topDictIdx.count == 0 {
		return nil, errors.New("CFF Top DICT INDEX is empty")
	}
	topDictData, err := indexElement(&topDictIdx, 0)
	if err != nil {
		return nil, err
	}
	cff.topDict, err = parseCFFTopDict(topDictData)
	if err != nil {
		return nil, fmt.Errorf("parsing Top DICT: %w", err)
	}

	// 4. String INDEX
	stringIdx, bytesRead, err := parseCFFINDEX(data, off)
	if err != nil {
		return nil, fmt.Errorf("parsing String INDEX: %w", err)
	}
	cff.stringIndex = stringIdx
	off += bytesRead

	// 5. Global Subr INDEX
	globalSubrs, _, err := parseCFFINDEX(data, off)
	if err != nil {
		return nil, fmt.Errorf("parsing Global Subr INDEX: %w", err)
	}
	cff.globalSubrs = globalSubrs

	// 6. CharStrings INDEX
	if cff.topDict.charStringsOff == 0 {
		return nil, errors.New("CFF Top DICT missing CharStrings offset")
	}
	charStringsIdx, _, err := parseCFFINDEX(data, int(cff.topDict.charStringsOff))
	if err != nil {
		return nil, fmt.Errorf("parsing CharStrings INDEX: %w", err)
	}
	cff.charStrings = charStringsIdx

	// 7. Charset
	numGlyphs := int(cff.charStrings.count)
	if cff.topDict.charsetOffset == 0 {
		// Default: ISOAdobe charset (glyph 0 = .notdef, SIDs match glyph IDs for first 229)
		cff.charset = CFFCharset{format: 0}
		cff.charset.entries = make([]uint16, numGlyphs-1)
		for i := range cff.charset.entries {
			cff.charset.entries[i] = uint16(i + 1)
		}
	} else {
		cff.charset, err = parseCFFCharset(data, int(cff.topDict.charsetOffset), numGlyphs)
		if err != nil {
			return nil, fmt.Errorf("parsing Charset: %w", err)
		}
	}

	// 8. Private DICT
	if cff.topDict.privateSize > 0 && cff.topDict.privateOffset > 0 {
		privEnd := int(cff.topDict.privateOffset + cff.topDict.privateSize)
		if privEnd > len(data) {
			return nil, errors.New("CFF Private DICT out of bounds")
		}
		cff.privateDict, err = parseCFFPrivateDict(data[cff.topDict.privateOffset:privEnd])
		if err != nil {
			return nil, fmt.Errorf("parsing Private DICT: %w", err)
		}

		// 9. Local Subr INDEX
		if cff.privateDict.localSubrOffset != 0 {
			localSubrOff := int(cff.topDict.privateOffset + cff.privateDict.localSubrOffset)
			if localSubrOff >= len(data) {
				return nil, errors.New("CFF Local Subr INDEX out of bounds")
			}
			cff.localSubrs, _, err = parseCFFINDEX(data, localSubrOff)
			if err != nil {
				return nil, fmt.Errorf("parsing Local Subr INDEX: %w", err)
			}
		}
	}

	return cff, nil
}

// parseCFFINDEX reads a CFF INDEX structure starting at the given offset.
// Returns the INDEX and the total bytes consumed.
func parseCFFINDEX(data []byte, offset int) (CFFINDEX, int, error) {
	if offset+2 > len(data) {
		return CFFINDEX{}, 0, errors.New("INDEX out of bounds")
	}

	b := BinaryFrom(data[offset:], false)
	count := b.U16()

	if count == 0 {
		return CFFINDEX{count: 0}, 2, nil
	}

	if offset+3 > len(data) {
		return CFFINDEX{}, 0, errors.New("INDEX offSize out of bounds")
	}
	offSize := int(data[offset+2])
	if offSize < 1 || offSize > 4 {
		return CFFINDEX{}, 0, fmt.Errorf("INDEX invalid offSize: %d", offSize)
	}

	// offsets array: count+1 entries, each offSize bytes
	offsetsStart := offset + 3
	offsetsEnd := offsetsStart + (int(count)+1)*offSize
	if offsetsEnd > len(data) {
		return CFFINDEX{}, 0, errors.New("INDEX offsets out of bounds")
	}

	offsets := make([]uint32, count+1)
	for i := 0; i <= int(count); i++ {
		off := offsetsStart + i*offSize
		var v uint32
		for j := 0; j < offSize; j++ {
			v = (v << 8) | uint32(data[off+j])
		}
		offsets[i] = v
	}

	// data region starts after offsets, ends at last offset value
	dataStart := offsetsEnd
	dataEnd := dataStart + int(offsets[count]) - 1
	if dataEnd > len(data) {
		dataEnd = len(data)
	}

	idx := CFFINDEX{
		count:   count,
		offsets: offsets,
		data:    data[dataStart:dataEnd],
	}

	totalBytes := dataEnd - offset
	return idx, totalBytes, nil
}

// indexElement returns the raw bytes for element i (0-based) of the INDEX.
func indexElement(idx *CFFINDEX, i int) ([]byte, error) {
	if i < 0 || i >= int(idx.count) {
		return nil, fmt.Errorf("INDEX element %d out of range (count=%d)", i, idx.count)
	}
	start := int(idx.offsets[i]) - 1
	end := int(idx.offsets[i+1]) - 1
	if start < 0 || end > len(idx.data) || start > end {
		return nil, fmt.Errorf("INDEX element %d has invalid offsets %d-%d (data len=%d)", i, start, end, len(idx.data))
	}
	return idx.data[start:end], nil
}

// --- DICT parsing ---

// parseCFFTopDict parses a Top DICT byte sequence.
func parseCFFTopDict(data []byte) (CFFTopDict, error) {
	var td CFFTopDict
	stack := make([]int32, 0, 16)

	for i := 0; i < len(data); {
		val, bytesRead, isOperand, err := readDICTOperand(data, i)
		if err != nil {
			return td, err
		}
		i += bytesRead

		if isOperand {
			stack = append(stack, val)
			continue
		}

		// Operator
		switch val {
		case 15: // charset
			if len(stack) > 0 {
				td.charsetOffset = uint32(stack[len(stack)-1])
			}
		case 17: // CharStrings
			if len(stack) > 0 {
				td.charStringsOff = uint32(stack[len(stack)-1])
			}
		case 18: // Private (size, offset)
			if len(stack) >= 2 {
				td.privateSize = uint32(stack[len(stack)-2])
				td.privateOffset = uint32(stack[len(stack)-1])
			}
		}
		stack = stack[:0]
	}

	return td, nil
}

// parseCFFPrivateDict parses a Private DICT byte sequence.
func parseCFFPrivateDict(data []byte) (CFFPrivateDict, error) {
	var pd CFFPrivateDict
	stack := make([]int32, 0, 16)

	for i := 0; i < len(data); {
		val, bytesRead, isOperand, err := readDICTOperand(data, i)
		if err != nil {
			return pd, err
		}
		i += bytesRead

		if isOperand {
			stack = append(stack, val)
			continue
		}

		switch val {
		case 6: // BlueValues (ignore)
		case 19: // Subrs (local subr offset relative to Private DICT start)
			if len(stack) > 0 {
				pd.localSubrOffset = uint32(stack[len(stack)-1])
			}
		case 20: // defaultWidthX
			if len(stack) > 0 {
				pd.defaultWidthX = stack[len(stack)-1]
			}
		case 21: // nominalWidthX
			if len(stack) > 0 {
				pd.nominalWidthX = stack[len(stack)-1]
			}
		}
		stack = stack[:0]
	}

	return pd, nil
}

// readDICTOperand reads one DICT operand or operator.
// Returns (value, bytesRead, isOperand, error).
// DICT encoding:
//   - 0-11: operators (1 byte)
//   - 12: escape (next byte is extended operator)
//   - 28: 3-byte integer (28, b1, b2) → int16
//   - 29: 5-byte integer (29, b1, b2, b3, b4) → int32
//   - 30: real number (BCD encoded) → we return as int32 (truncated)
//   - 32-246: 1-byte integer: value = b0 - 139
//   - 247-250: 2-byte integer: value = (b0-247)*256 + b1 + 108
//   - 251-254: 2-byte integer: value = -(b0-251)*256 - b1 - 108
func readDICTOperand(data []byte, offset int) (int32, int, bool, error) {
	if offset >= len(data) {
		return 0, 0, false, errors.New("DICT unexpected end")
	}

	b0 := data[offset]

	// Operators: 0-21 (single byte), 12 xx (two bytes)
	if b0 <= 21 {
		if b0 == 12 {
			if offset+1 >= len(data) {
				return 0, 0, false, errors.New("DICT escape unexpected end")
			}
			op := int32(1200) + int32(data[offset+1]) // encode 12 xx as 1200+xx
			return op, 2, false, nil
		}
		return int32(b0), 1, false, nil
	}

	if b0 == 28 {
		if offset+2 >= len(data) {
			return 0, 0, false, errors.New("DICT 28 unexpected end")
		}
		v := int32(int16(uint16(data[offset+1])<<8 | uint16(data[offset+2])))
		return v, 3, true, nil
	}

	if b0 == 29 {
		if offset+4 >= len(data) {
			return 0, 0, false, errors.New("DICT 29 unexpected end")
		}
		v := int32(data[offset+1])<<24 | int32(data[offset+2])<<16 | int32(data[offset+3])<<8 | int32(data[offset+4])
		return v, 5, true, nil
	}

	if b0 == 30 {
		// Real number (BCD encoded) — read and convert to int32
		i := offset + 1
		for i < len(data) {
			nibblePair := data[i]
			// High nibble
			if nibblePair>>4 == 0xf {
				return 0, i + 1 - offset, true, nil
			}
			// Low nibble
			if nibblePair&0x0f == 0x0f {
				return 0, i + 1 - offset, true, nil
			}
			i++
		}
		return 0, i - offset, true, nil
	}

	if b0 >= 32 && b0 <= 246 {
		return int32(b0) - 139, 1, true, nil
	}

	if b0 >= 247 && b0 <= 250 {
		if offset+1 >= len(data) {
			return 0, 0, false, errors.New("DICT 247-250 unexpected end")
		}
		b1 := data[offset+1]
		return (int32(b0)-247)*256 + int32(b1) + 108, 2, true, nil
	}

	if b0 >= 251 && b0 <= 254 {
		if offset+1 >= len(data) {
			return 0, 0, false, errors.New("DICT 251-254 unexpected end")
		}
		b1 := data[offset+1]
		return -(int32(b0)-251)*256 - int32(b1) - 108, 2, true, nil
	}

	return 0, 0, false, fmt.Errorf("DICT invalid byte: %d", b0)
}

// --- Charset parsing ---

// parseCFFCharset parses a CFF charset at the given offset.
// Glyph 0 is always .notdef (not stored); entries start from glyph 1.
func parseCFFCharset(data []byte, offset int, numGlyphs int) (CFFCharset, error) {
	if offset >= len(data) {
		return CFFCharset{}, errors.New("charset offset out of bounds")
	}

	cs := CFFCharset{format: data[offset]}
	off := offset + 1

	switch cs.format {
	case 0:
		// Format 0: nGlyphs-1 SID entries (uint16 each)
		count := numGlyphs - 1
		cs.entries = make([]uint16, count)
		for i := 0; i < count; i++ {
			if off+2 > len(data) {
				return cs, errors.New("charset format 0 truncated")
			}
			cs.entries[i] = uint16(data[off])<<8 | uint16(data[off+1])
			off += 2
		}

	case 1:
		// Format 1: ranges with uint8 nLeft
		cs.entries = make([]uint16, 0, numGlyphs-1)
		for len(cs.entries) < numGlyphs-1 {
			if off+3 > len(data) {
				return cs, errors.New("charset format 1 truncated")
			}
			sid := uint16(data[off])<<8 | uint16(data[off+1])
			nLeft := int(data[off+2])
			off += 3
			for j := 0; j <= nLeft && len(cs.entries) < numGlyphs-1; j++ {
				cs.entries = append(cs.entries, sid+uint16(j))
			}
		}

	case 2:
		// Format 2: ranges with uint16 nLeft
		cs.entries = make([]uint16, 0, numGlyphs-1)
		for len(cs.entries) < numGlyphs-1 {
			if off+4 > len(data) {
				return cs, errors.New("charset format 2 truncated")
			}
			sid := uint16(data[off])<<8 | uint16(data[off+1])
			nLeft := int(uint16(data[off+2])<<8 | uint16(data[off+3]))
			off += 4
			for j := 0; j <= nLeft && len(cs.entries) < numGlyphs-1; j++ {
				cs.entries = append(cs.entries, sid+uint16(j))
			}
		}

	default:
		return cs, fmt.Errorf("unsupported charset format: %d", cs.format)
	}

	return cs, nil
}

// --- SID resolution ---

// cffSIDToString resolves a SID to a string.
func cffSIDToString(sid uint16, stringIndex *CFFINDEX) string {
	if sid < uint16(len(cffStandardStrings)) {
		return cffStandardStrings[sid]
	}
	// Custom string from String INDEX
	customIdx := int(sid) - len(cffStandardStrings)
	if customIdx < 0 || customIdx >= int(stringIndex.count) {
		return ""
	}
	data, err := indexElement(stringIndex, customIdx)
	if err != nil {
		return ""
	}
	return string(data)
}

// --- CFF public methods ---

// FontName returns the font name from the CFF Name INDEX.
func (cff *CFF) FontName() string {
	if cff.nameIndex.count == 0 {
		return ""
	}
	data, err := indexElement(&cff.nameIndex, 0)
	if err != nil {
		return ""
	}
	return string(data)
}

// NumGlyphs returns the number of glyphs from the CharStrings INDEX.
func (cff *CFF) NumGlyphs() int {
	return int(cff.charStrings.count)
}

// GlyphName returns the glyph name for the given glyph ID.
func (cff *CFF) GlyphName(glyphID int) string {
	if glyphID < 0 || glyphID >= cff.NumGlyphs() {
		return ""
	}
	if glyphID == 0 {
		return ".notdef"
	}
	if cff.glyphNames != nil && glyphID < len(cff.glyphNames) {
		return cff.glyphNames[glyphID]
	}

	// Lazy-compute all glyph names
	cff.buildGlyphNames()
	if glyphID < len(cff.glyphNames) {
		return cff.glyphNames[glyphID]
	}
	return ""
}

func (cff *CFF) buildGlyphNames() {
	n := cff.NumGlyphs()
	cff.glyphNames = make([]string, n)
	cff.glyphNames[0] = ".notdef"
	for i := 1; i < n; i++ {
		if i-1 < len(cff.charset.entries) {
			cff.glyphNames[i] = cffSIDToString(cff.charset.entries[i-1], &cff.stringIndex)
		}
	}
}

// CharStringData returns the raw charstring data for the given glyph ID.
func (cff *CFF) CharStringData(glyphID int) ([]byte, error) {
	return indexElement(&cff.charStrings, glyphID)
}

// cffStandardStrings is the standard CFF string table (391 entries, SIDs 0-390).
var cffStandardStrings = []string{
	// 0-37: Core
	".notdef", "space", "exclam", "quotedbl", "numbersign",
	"dollar", "percent", "ampersand", "quoteright", "parenleft",
	"parenright", "asterisk", "plus", "comma", "hyphen",
	"period", "slash", "zero", "one", "two",
	"three", "four", "five", "six", "seven",
	"eight", "nine", "colon", "semicolon", "less",
	"equal", "greater", "question", "at", "A",
	"B", "C", "D",
	// 38-75
	"E", "F", "G", "H", "I", "J", "K", "L", "M",
	"N", "O", "P", "Q", "R", "S", "T", "U", "V",
	"W", "X", "Y", "Z", "bracketleft", "backslash",
	"bracketright", "asciicircum", "underscore",
	"quoteleft", "a", "b", "c", "d", "e", "f",
	"g", "h", "i", "j",
	// 76-113
	"k", "l", "m", "n", "o", "p", "q", "r", "s", "t",
	"u", "v", "w", "x", "y", "z", "braceleft", "bar",
	"braceright", "asciitilde", "exclamdown", "cent",
	"sterling", "fraction", "yen", "florin", "section",
	"currency", "quotesingle", "quotedblleft",
	"guillemotleft", "guilsinglleft", "guilsinglright",
	"fi", "fl",
	// 114-151
	"dagger", "daggerdbl", "periodcentered", "paragraph",
	"bullet", "quotesinglbase", "quotedblbase",
	"quotedblright", "guillemotright", "ellipsis",
	"perthousand", "questiondown", "grave", "acute",
	"circumflex", "tilde", "macron", "breve", "dotaccent",
	"dieresis",
	// 152-189
	"ring", "cedilla", "hungarumlaut", "ogonek", "caron",
	"emdash", "AE", "ordfeminine", "Lslash", "Oslash",
	"OE", "ordmasculine", "ae", "dotlessi", "lslash",
	"oslash", "oe", "germandbls", "Idotaccent",
	// 190-227
	"one", "two", "three", "four", "five", "six",
	"seven", "eight", "nine", "ten", "eleven", "twelve",
	"thirteen", "fourteen", "fifteen", "sixteen",
	"seventeen", "eighteen", "nineteen", "twenty",
	"twentyone", "twentytwo", "twentythree", "twentyfour",
	"twentyfive", "twentysix", "twentyseven", "twentyeight",
	"twentynine", "thirty", "thetaone",
	// 228-265
	"Alpha", "Beta", "Gamma", "Delta", "Epsilon",
	"Zeta", "Eta", "Theta", "Iota", "Kappa", "Lambda",
	"Mu", "Nu", "Xi", "Omicron", "Pi", "Rho", "Sigma",
	"Tau", "Upsilon", "Phi", "Chi", "Psi", "Omega",
	"alpha", "beta", "gamma", "delta", "epsilon",
	"zeta", "eta",
	// 266-303
	"theta", "iota", "kappa", "lambda", "mu", "nu",
	"xi", "omicron", "pi", "rho", "sigma", "tau",
	"upsilon", "phi", "chi", "psi", "omega", "dotmath",
	"dotlessj", "weierstrass", "ringmath", "partialdiff",
	"nabla", "radical", "heart", "club", "diamond",
	"spade", "arrowleft", "arrowup", "arrowright", "arrowdown",
	// 304-341
	"arrowboth", "arrowvertex", "arrowhoriz",
	"registersans", "copyrightsans", "trademarksans",
	"parenlefttp", "parenleftex", "parenleftbt",
	"parenrighttp", "parenrightex", "parenrightbt",
	"bracketlefttp", "bracketleftex", "bracketleftbt",
	"bracketrighttp", "bracketrightex", "bracketrightbt",
	"bracelefttp", "braceleftmid", "braceleftbt",
	"bracerighttp", "bracerightmid", "bracerightbt",
	"angle", "angleleft", "angleright", "infinity",
	"notequal", "approxequal", "equivalence",
	// 342-379
	"radicalex", "plusminus", "second", "minute",
	"lessequal", "greaterequal", "propeller",
	"partialdiff", "florin", "angle", "integral",
	"integraltp", "integralex", "integralbt",
	"radical", "approxequal", "infinity", "logicalnot",
	"intersection", "union", "orthogonal", "ceilingleft",
	"ceilingright", "bracketleft", "bracketright",
	"floorleft", "floorright", "perfraction",
	"propeller", "radical", "logicaland", "logicalor",
	"arrowdblleft",
	// 380-390
	"arrowdblup", "arrowdblright", "arrowdbldown",
	"arrowboth", "arrowvertex", "arrowhoriz",
	"registerserif", "copyrightserif", " trademarkserif",
	"registered", "copyright",
}
