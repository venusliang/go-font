package gofont

import "errors"

type LongHorMetric struct {
	advanceWidth uint16
	lsb          int16
}

type Hmtx struct {
	hMetrics        []LongHorMetric
	leftSideBearing []int16
}

func parseHmtx(data []byte, numHMetrics int, numGlyphs int) (*Hmtx, error) {
	if numGlyphs < numHMetrics {
		return nil, errors.New("numGlyphs must be >= numHMetrics")
	}

	expectedSize := numHMetrics*4 + (numGlyphs-numHMetrics)*2
	if len(data) < expectedSize {
		return nil, errors.New("hmtx table too small")
	}

	binary := BinaryFrom(data, false)

	hmtx := &Hmtx{
		hMetrics:        make([]LongHorMetric, numHMetrics),
		leftSideBearing: make([]int16, numGlyphs-numHMetrics),
	}

	for i := 0; i < numHMetrics; i++ {
		hmtx.hMetrics[i] = LongHorMetric{
			advanceWidth: binary.U16(),
			lsb:          binary.I16(),
		}
	}

	for i := 0; i < numGlyphs-numHMetrics; i++ {
		hmtx.leftSideBearing[i] = binary.I16()
	}

	return hmtx, nil
}

func writeHmtx(hmtx *Hmtx) []byte {
	size := len(hmtx.hMetrics)*4 + len(hmtx.leftSideBearing)*2
	data := make([]byte, size)
	binary := BinaryFrom(data, false)

	for _, m := range hmtx.hMetrics {
		binary.PutU16(m.advanceWidth)
		binary.PutU16(uint16(m.lsb))
	}

	for _, lsb := range hmtx.leftSideBearing {
		binary.PutU16(uint16(lsb))
	}

	return data
}
