package gofont

import "errors"

// Kern represents the kerning table.
type Kern struct {
	version   uint16
	subtables []KernSubtable
}

// KernSubtable represents a single subtable within the kern table.
type KernSubtable struct {
	version  uint16
	length   uint16
	coverage uint16
	format   uint8 // extracted from coverage bits 8-15

	// Format 0: ordered list of kerning pairs
	pairs []KernPair

	// Other formats: raw bytes
	rawData []byte
}

// KernPair represents a single kerning pair.
type KernPair struct {
	Left  uint16
	Right uint16
	Value int16
}

func parseKern(data []byte) (*Kern, error) {
	if len(data) < 4 {
		return nil, errors.New("kern table too small")
	}

	binary := BinaryFrom(data, false)
	kern := &Kern{
		version:   binary.U16(),
		subtables: make([]KernSubtable, binary.U16()),
	}

	for i := range kern.subtables {
		if binary.Offset()+6 > len(data) {
			return nil, errors.New("kern subtable header out of bounds")
		}

		sub := KernSubtable{
			version:  binary.U16(),
			length:   binary.U16(),
			coverage: binary.U16(),
		}
		sub.format = uint8(sub.coverage >> 8)

		subStart := binary.Offset() - 6 // start of this subtable (after header)
		subDataLen := int(sub.length) - 6

		if subDataLen < 0 || subStart+subDataLen > len(data) {
			return nil, errors.New("kern subtable data out of bounds")
		}

		switch sub.format {
		case 0:
			if subDataLen < 8 {
				return nil, errors.New("kern format 0 subtable too small")
			}
			nPairs := binary.U16()
			_ = binary.U16() // searchRange
			_ = binary.U16() // entrySelector
			_ = binary.U16() // rangeShift

			if nPairs*6 > uint16(binary.Offset()+subDataLen-binary.Offset()) {
				// Sanity check - just read what we can
			}
			sub.pairs = make([]KernPair, nPairs)
			for j := range sub.pairs {
				sub.pairs[j] = KernPair{
					Left:  binary.U16(),
					Right: binary.U16(),
					Value: binary.I16(),
				}
			}
		default:
			// Store raw data for unsupported formats
			sub.rawData = make([]byte, subDataLen)
			copy(sub.rawData, data[binary.Offset():subStart+subDataLen])
			// Advance binary past this subtable
			for binary.Offset() < subStart+subDataLen {
				binary.U8()
			}
		}

		// Advance to next subtable
		nextOffset := subStart + int(sub.length)
		for binary.Offset() < nextOffset && binary.Offset() < len(data) {
			binary.U8()
		}

		kern.subtables[i] = sub
	}

	return kern, nil
}

func writeKern(kern *Kern) []byte {
	// Calculate total size
	size := 4 // version + nTables
	for i := range kern.subtables {
		size += int(kern.subtables[i].length)
	}

	data := make([]byte, size)
	binary := BinaryFrom(data, false)

	binary.PutU16(kern.version)
	binary.PutU16(uint16(len(kern.subtables)))

	for _, sub := range kern.subtables {
		binary.PutU16(sub.version)
		binary.PutU16(sub.length)
		binary.PutU16(sub.coverage)

		switch sub.format {
		case 0:
			nPairs := uint16(len(sub.pairs))
			binary.PutU16(nPairs)
			// Calculate search params
			sr, es, rs := calcSearchParams(int(nPairs))
			binary.PutU16(sr * 6)
			binary.PutU16(es)
			binary.PutU16(rs * 6)
			for _, p := range sub.pairs {
				binary.PutU16(p.Left)
				binary.PutU16(p.Right)
				binary.PutU16(uint16(p.Value))
			}
		default:
			binary.Append(sub.rawData)
		}
	}

	return data
}
