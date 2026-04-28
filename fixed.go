package gofonts

import "fmt"

// Fixed is a 16.16 fixed-point number.
type Fixed16_16 struct {
	Int  int16
	Frac uint16
}

func (f Fixed16_16) Float() float64 {
	return float64(f.Int) + float64(f.Frac)/65536
}

func (f Fixed16_16) String() string {
	return fmt.Sprintf("%d.%d", f.Int, f.Frac)
}

// 16-bit signed fixed number with the low 14 bits representing fraction.
type Fixed2_14 int16

func (f Fixed2_14) String() string {
	return fmt.Sprintf("%d", f)
}

func (f Fixed2_14) Float() float64 {
	return float64(f) / (1 << 14)
}
