package gofont

import "errors"

type Loca struct {
	offsets []uint32
}

func parseLoca(data []byte, numGlyphs int, indexToLocFormat int16) (*Loca, error) {
	loca := &Loca{
		offsets: make([]uint32, numGlyphs+1),
	}

	binary := BinaryFrom(data, false)

	switch indexToLocFormat {
	case 0: // short format: uint16 values, multiply by 2
		if len(data) < (numGlyphs+1)*2 {
			return nil, errors.New("loca table too small for short format")
		}
		for i := 0; i <= numGlyphs; i++ {
			loca.offsets[i] = uint32(binary.U16()) * 2
		}
	case 1: // long format: uint32 values
		if len(data) < (numGlyphs+1)*4 {
			return nil, errors.New("loca table too small for long format")
		}
		for i := 0; i <= numGlyphs; i++ {
			loca.offsets[i] = binary.U32()
		}
	default:
		return nil, errors.New("invalid indexToLocFormat")
	}

	return loca, nil
}

func writeLoca(loca *Loca, indexToLocFormat int16) []byte {
	var data []byte

	switch indexToLocFormat {
	case 0:
		data = make([]byte, len(loca.offsets)*2)
		binary := BinaryFrom(data, false)
		for _, off := range loca.offsets {
			binary.PutU16(uint16(off / 2))
		}
	case 1:
		data = make([]byte, len(loca.offsets)*4)
		binary := BinaryFrom(data, false)
		for _, off := range loca.offsets {
			binary.PutU32(off)
		}
	}

	return data
}
