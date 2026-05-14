package gofont

// Prep holds the prep (Control Value Program) table data.
// This TrueType bytecode is executed whenever the font size or transformation changes.
type Prep struct {
	instructions []byte
}

// Cvt holds the cvt (Control Value Table) data.
// It contains an array of FWord (int16) values used by the TrueType interpreter.
type Cvt struct {
	values []int16
}

// Fpgm holds the fpgm (Font Program) table data.
// This TrueType bytecode is executed once when the font is first loaded.
type Fpgm struct {
	instructions []byte
}

func parsePrep(data []byte) *Prep {
	ins := make([]byte, len(data))
	copy(ins, data)
	return &Prep{instructions: ins}
}

func writePrep(prep *Prep) []byte {
	out := make([]byte, len(prep.instructions))
	copy(out, prep.instructions)
	return out
}

func parseCvt(data []byte) *Cvt {
	n := len(data) / 2
	values := make([]int16, n)
	binary := BinaryFrom(data, false)
	for i := 0; i < n; i++ {
		values[i] = binary.I16()
	}
	return &Cvt{values: values}
}

func writeCvt(cvt *Cvt) []byte {
	data := make([]byte, len(cvt.values)*2)
	binary := BinaryFrom(data, false)
	for _, v := range cvt.values {
		binary.PutU16(uint16(v))
	}
	return data
}

func parseFpgm(data []byte) *Fpgm {
	ins := make([]byte, len(data))
	copy(ins, data)
	return &Fpgm{instructions: ins}
}

func writeFpgm(fpgm *Fpgm) []byte {
	out := make([]byte, len(fpgm.instructions))
	copy(out, fpgm.instructions)
	return out
}
