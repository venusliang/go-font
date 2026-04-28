package gofonts

type Binary interface {
	U8() uint8
	I8() int8
	U16() uint16
	I16() int16
	Fixed16_16() Fixed16_16
	U32() uint32
	I32() int32
	U64() uint64
	I64() int64
	PutU8(uint8)
	PutU16(uint16)
	PutFixed16_16(Fixed16_16)
	PutU32(uint32)
	PutU64(uint64)
	Offset() int
	Bytes(int) []uint8
	Read(int) []uint8
	Slice(int) Binary
	Append([]byte)
}

func BinaryFrom(data []byte, littleEndian bool) Binary {
	if littleEndian {
		return &LittleEndian{data, 0}
	}

	return &BigEndian{data, 0}
}

type LittleEndian struct {
	data   []byte
	offset int
}

func (l *LittleEndian) Offset() int {
	return l.offset
}

func (l *LittleEndian) Bytes(n int) []byte {
	bytes := l.data[l.offset : l.offset+n]
	return bytes
}

func (l *LittleEndian) Read(n int) []byte {
	bytes := l.Bytes(n)
	l.offset += n
	return bytes
}

func (l *LittleEndian) Slice(n int) Binary {
	bytes := l.Read(n)

	return &LittleEndian{data: bytes}
}

func (l *LittleEndian) U8() (u8 uint8) {
	u8 = l.data[l.offset]
	l.offset++
	return
}

func (l *LittleEndian) I8() (i8 int8) {
	i8 = int8(l.data[l.offset])
	l.offset++
	return
}

func (l *LittleEndian) U16() (u16 uint16) {
	u16 = uint16(l.data[l.offset]) | uint16(l.data[l.offset+1])<<8
	l.offset += 2
	return
}

func (l *LittleEndian) I16() (i16 int16) {
	i16 = int16(l.data[l.offset]) | int16(l.data[l.offset+1])<<8
	l.offset += 2
	return
}

func (l *LittleEndian) Fixed16_16() (f Fixed16_16) {
	f.Int = int16(l.data[l.offset]) | int16(l.data[l.offset+1])<<8
	f.Frac = uint16(l.data[l.offset]) | uint16(l.data[l.offset+1])<<8
	l.offset += 2
	return
}

func (l *LittleEndian) U32() (u32 uint32) {
	u32 = uint32(l.data[l.offset]) | uint32(l.data[l.offset+1])<<8 | uint32(l.data[l.offset+2])<<16 | uint32(l.data[l.offset+3])<<24
	l.offset += 4
	return
}

func (l *LittleEndian) I32() (i32 int32) {
	i32 = int32(l.data[l.offset]) | int32(l.data[l.offset+1])<<8 | int32(l.data[l.offset+2])<<16 | int32(l.data[l.offset+3])<<24
	l.offset += 4
	return
}

func (l *LittleEndian) U64() (u64 uint64) {
	u64 = uint64(l.data[l.offset]) | uint64(l.data[l.offset+1])<<8 | uint64(l.data[l.offset+2])<<16 | uint64(l.data[l.offset+3])<<24 | uint64(l.data[l.offset+4])<<32 | uint64(l.data[l.offset+5])<<40 | uint64(l.data[l.offset+6])<<48 | uint64(l.data[l.offset+7])<<56
	l.offset += 8
	return
}

func (l *LittleEndian) I64() (i64 int64) {
	i64 = int64(l.data[l.offset]) | int64(l.data[l.offset+1])<<8 | int64(l.data[l.offset+2])<<16 | int64(l.data[l.offset+3])<<24 | int64(l.data[l.offset+4])<<32 | int64(l.data[l.offset+5])<<40 | int64(l.data[l.offset+6])<<48 | int64(l.data[l.offset+7])<<56
	l.offset += 8
	return
}

func (l *LittleEndian) PutU8(u8 uint8) {
	l.data[l.offset] = u8
	l.offset++
}

func (l *LittleEndian) PutU16(u16 uint16) {
	l.data[l.offset] = uint8(u16)
	l.data[l.offset+1] = uint8(u16 >> 8)
	l.offset += 2
}

func (l *LittleEndian) PutFixed16_16(f Fixed16_16) {
	l.data[l.offset] = uint8(f.Int)
	l.data[l.offset+1] = uint8(f.Int >> 8)
	l.data[l.offset+2] = uint8(f.Frac)
	l.data[l.offset+3] = uint8(f.Frac >> 8)
	l.offset += 4
}

func (l *LittleEndian) PutU32(u32 uint32) {
	l.data[l.offset] = uint8(u32)
	l.data[l.offset+1] = uint8(u32 >> 8)
	l.data[l.offset+2] = uint8(u32 >> 16)
	l.data[l.offset+3] = uint8(u32 >> 24)
	l.offset += 4
}

func (l *LittleEndian) PutU64(u64 uint64) {
	l.data[l.offset] = uint8(u64)
	l.data[l.offset+1] = uint8(u64 >> 8)
	l.data[l.offset+2] = uint8(u64 >> 16)
	l.data[l.offset+3] = uint8(u64 >> 24)
	l.data[l.offset+4] = uint8(u64 >> 32)
	l.data[l.offset+5] = uint8(u64 >> 40)
	l.data[l.offset+6] = uint8(u64 >> 48)
	l.data[l.offset+7] = uint8(u64 >> 56)
	l.offset += 8
}

func (l *LittleEndian) Append(bytes []byte) {
	copy(l.data[l.offset:], bytes)
	l.offset += len(bytes)
}

type BigEndian struct {
	data   []byte
	offset int
}

func (b *BigEndian) Offset() int {
	return b.offset
}

func (b *BigEndian) Bytes(n int) []byte {
	bytes := b.data[b.offset : b.offset+n]
	return bytes
}

func (b *BigEndian) Read(n int) []byte {
	bytes := b.Bytes(n)
	b.offset += n
	return bytes
}

func (l *BigEndian) Slice(n int) Binary {
	bytes := l.Read(n)

	return &BigEndian{data: bytes}
}

func (b *BigEndian) U8() (u8 uint8) {
	u8 = b.data[b.offset]
	b.offset++
	return
}

func (b *BigEndian) I8() (i8 int8) {
	i8 = int8(b.data[b.offset])
	b.offset++
	return
}

func (b *BigEndian) U16() (u16 uint16) {
	u16 = uint16(b.data[b.offset])<<8 | uint16(b.data[b.offset+1])
	b.offset += 2
	return
}

func (b *BigEndian) I16() (i16 int16) {
	i16 = int16(b.data[b.offset])<<8 | int16(b.data[b.offset+1])
	b.offset += 2
	return
}

func (b *BigEndian) Fixed16_16() (f Fixed16_16) {
	f.Int = int16(b.data[b.offset])<<8 | int16(b.data[b.offset+1])
	f.Frac = uint16(b.data[b.offset])<<8 | uint16(b.data[b.offset+1])
	b.offset += 4
	return
}
func (b *BigEndian) U32() (u32 uint32) {
	u32 = uint32(b.data[b.offset])<<24 | uint32(b.data[b.offset+1])<<16 | uint32(b.data[b.offset+2])<<8 | uint32(b.data[b.offset+3])
	b.offset += 4
	return
}

func (b *BigEndian) I32() (i32 int32) {
	i32 = int32(b.data[b.offset])<<24 | int32(b.data[b.offset+1])<<16 | int32(b.data[b.offset+2])<<8 | int32(b.data[b.offset+3])
	b.offset += 4
	return
}

func (b *BigEndian) U64() (u64 uint64) {
	u64 = uint64(b.data[b.offset])<<56 | uint64(b.data[b.offset+1])<<48 | uint64(b.data[b.offset+2])<<40 | uint64(b.data[b.offset+3])<<32 | uint64(b.data[b.offset+4])<<24 | uint64(b.data[b.offset+5])<<16 | uint64(b.data[b.offset+6])<<8 | uint64(b.data[b.offset+7])
	b.offset += 8
	return
}

func (b *BigEndian) I64() (i64 int64) {
	i64 = int64(b.data[b.offset])<<56 | int64(b.data[b.offset+1])<<48 | int64(b.data[b.offset+2])<<40 | int64(b.data[b.offset+3])<<32 | int64(b.data[b.offset+4])<<24 | int64(b.data[b.offset+5])<<16 | int64(b.data[b.offset+6])<<8 | int64(b.data[b.offset+7])
	b.offset += 8
	return
}

func (b *BigEndian) PutU8(u8 uint8) {
	b.data[b.offset] = u8
	b.offset++
}

func (b *BigEndian) PutU16(u16 uint16) {
	b.data[b.offset] = uint8(u16 >> 8)
	b.data[b.offset+1] = uint8(u16)
	b.offset += 2
}

func (b *BigEndian) PutFixed16_16(f Fixed16_16) {
	b.data[b.offset] = uint8(f.Int >> 8)
	b.data[b.offset+1] = uint8(f.Int)
	b.data[b.offset+2] = uint8(f.Frac >> 8)
	b.data[b.offset+3] = uint8(f.Frac)
	b.offset += 4
}

func (b *BigEndian) PutU32(u32 uint32) {
	b.data[b.offset] = uint8(u32 >> 24)
	b.data[b.offset+1] = uint8(u32 >> 16)
	b.data[b.offset+2] = uint8(u32 >> 8)
	b.data[b.offset+3] = uint8(u32)
	b.offset += 4
}

func (b *BigEndian) PutU64(u64 uint64) {
	b.data[b.offset] = uint8(u64 >> 56)
	b.data[b.offset+1] = uint8(u64 >> 48)
	b.data[b.offset+2] = uint8(u64 >> 40)
	b.data[b.offset+3] = uint8(u64 >> 32)
	b.data[b.offset+4] = uint8(u64 >> 24)
	b.data[b.offset+5] = uint8(u64 >> 16)
	b.data[b.offset+6] = uint8(u64 >> 8)
	b.data[b.offset+7] = uint8(u64)
	b.offset += 8
}

func (b *BigEndian) Append(bytes []byte) {
	copy(b.data[b.offset:], bytes)
	b.offset += len(bytes)
}
