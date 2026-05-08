package gofont

import (
	"errors"
	"fmt"
)

// CFFPathOp represents a path operation in a CFF charstring outline.
type CFFPathOp uint8

const (
	CFFOpMoveTo  CFFPathOp = iota // rmoveto, hmoveto, vmoveto
	CFFOpLineTo                    // rlineto, hlineto, vlineto
	CFFOpCurveTo                   // rrcurveto, hhcurveto, vvcurveto, hvcurveto, vhcurveto
)

// CFFPathSegment represents one path drawing segment in a CFF outline.
type CFFPathSegment struct {
	Op   CFFPathOp
	Args [6]int32 // relative coordinates; at most 6 for CurveTo
}

// CFFOutline holds the parsed outline data for a single CFF glyph.
type CFFOutline struct {
	segments []CFFPathSegment
	width    int32
	hasWidth bool
	xMin     int16
	yMin     int16
	xMax     int16
	yMax     int16
}

// NumSegments returns the number of path segments.
func (o *CFFOutline) NumSegments() int {
	if o == nil {
		return 0
	}
	return len(o.segments)
}

// Segments returns all path segments.
func (o *CFFOutline) Segments() []CFFPathSegment {
	if o == nil {
		return nil
	}
	return o.segments
}

// Width returns the glyph width. ok is true if width was specified in charstring.
func (o *CFFOutline) Width() (width int32, ok bool) {
	if o == nil {
		return 0, false
	}
	return o.width, o.hasWidth
}

// BBox returns the computed bounding box.
func (o *CFFOutline) BBox() (xMin, yMin, xMax, yMax int16) {
	if o == nil {
		return 0, 0, 0, 0
	}
	return o.xMin, o.yMin, o.xMax, o.yMax
}

// decodeCharString decodes a Type 2 charstring into outline segments.
func decodeCharString(
	data []byte,
	globalSubrs *CFFINDEX,
	localSubrs *CFFINDEX,
	defaultWidthX int32,
	nominalWidthX int32,
) (*CFFOutline, error) {
	vm := &charstringVM{
		stack:         make([]int32, 0, 48),
		globalSubrs:   globalSubrs,
		localSubrs:    localSubrs,
		defaultWidthX: defaultWidthX,
		nominalWidthX: nominalWidthX,
	}

	err := vm.execute(data, 0)
	if err != nil {
		return nil, err
	}

	outline := &CFFOutline{
		segments: vm.segments,
		width:    vm.width,
		hasWidth: vm.hasWidth,
	}

	// Compute bounding box from absolute positions
	if len(vm.segments) > 0 {
		x, y := int32(0), int32(0)
		first := true
		for _, seg := range vm.segments {
			switch seg.Op {
			case CFFOpMoveTo:
				x += seg.Args[0]
				y += seg.Args[1]
			case CFFOpLineTo:
				x += seg.Args[0]
				y += seg.Args[1]
			case CFFOpCurveTo:
				x += seg.Args[4]
				y += seg.Args[5]
			}
			if first {
				outline.xMin = int16(x)
				outline.yMin = int16(y)
				outline.xMax = int16(x)
				outline.yMax = int16(y)
				first = false
			}
			// For curves, also check control points
			checks := []struct{ cx, cy int32 }{
				{x, y},
			}
			if seg.Op == CFFOpCurveTo {
				// Control point 1
				cx1 := x - seg.Args[4] + seg.Args[0]
				cy1 := y - seg.Args[5] + seg.Args[1]
				// Control point 2
				cx2 := x - seg.Args[4] + seg.Args[2]
				cy2 := y - seg.Args[5] + seg.Args[3]
				checks = append(checks, struct{ cx, cy int32 }{cx1, cy1})
				checks = append(checks, struct{ cx, cy int32 }{cx2, cy2})
			}
			for _, c := range checks {
				ix, iy := int16(c.cx), int16(c.cy)
				if ix < outline.xMin {
					outline.xMin = ix
				}
				if iy < outline.yMin {
					outline.yMin = iy
				}
				if ix > outline.xMax {
					outline.xMax = ix
				}
				if iy > outline.yMax {
					outline.yMax = iy
				}
			}
		}
	}

	return outline, nil
}

// charstringVM is the Type 2 charstring virtual machine.
type charstringVM struct {
	stack          []int32
	x, y           int32
	segments       []CFFPathSegment
	width          int32
	hasWidth       bool
	widthSet       bool
	globalSubrs    *CFFINDEX
	localSubrs     *CFFINDEX
	defaultWidthX  int32
	nominalWidthX  int32
	hintState      int // number of hints for hintmask/cntrmask
	callDepth      int
}

func (vm *charstringVM) push(v int32) {
	if len(vm.stack) < 48 {
		vm.stack = append(vm.stack, v)
	}
}

func (vm *charstringVM) pop() (int32, bool) {
	n := len(vm.stack)
	if n == 0 {
		return 0, false
	}
	v := vm.stack[n-1]
	vm.stack = vm.stack[:n-1]
	return v, true
}

func (vm *charstringVM) execute(data []byte, depth int) error {
	if depth > 10 {
		return errors.New("charstring call depth exceeded")
	}

	for i := 0; i < len(data); {
		b0 := data[i]

		// Operand encoding
		if b0 >= 32 && b0 <= 246 {
			vm.push(int32(b0) - 139)
			i++
			continue
		}
		if b0 >= 247 && b0 <= 250 {
			if i+1 >= len(data) {
				return errors.New("charstring truncated (247-250)")
			}
			val := (int32(b0)-247)*256 + int32(data[i+1]) + 108
			vm.push(val)
			i += 2
			continue
		}
		if b0 >= 251 && b0 <= 254 {
			if i+1 >= len(data) {
				return errors.New("charstring truncated (251-254)")
			}
			val := -(int32(b0)-251)*256 - int32(data[i+1]) - 108
			vm.push(val)
			i += 2
			continue
		}
		if b0 == 255 {
			// Fixed-point 16.16 — treat as int32 for charstring coordinates
			if i+4 >= len(data) {
				return errors.New("charstring truncated (255)")
			}
			val := int32(data[i+1])<<24 | int32(data[i+2])<<16 | int32(data[i+3])<<8 | int32(data[i+4])
			vm.push(val >> 16) // Convert fixed 16.16 to integer
			i += 5
			continue
		}
		if b0 == 28 {
			if i+2 >= len(data) {
				return errors.New("charstring truncated (28)")
			}
			val := int32(int16(uint16(data[i+1])<<8 | uint16(data[i+2])))
			vm.push(val)
			i += 3
			continue
		}

		// Operators (0-27, 31)
		switch b0 {
		case 14: // endchar
			// If there's a width on the stack, consume it
			if len(vm.stack) == 1 && !vm.widthSet {
				vm.width = vm.nominalWidth()
				vm.hasWidth = true
				vm.widthSet = true
			}
			vm.stack = vm.stack[:0]
			return nil

		case 21: // rmoveto
			dy, _ := vm.pop()
			dx, _ := vm.pop()
			vm.checkWidth(2)
			vm.x += dx
			vm.y += dy
			vm.segments = append(vm.segments, CFFPathSegment{Op: CFFOpMoveTo, Args: [6]int32{dx, dy}})
			vm.stack = vm.stack[:0]

		case 22: // hmoveto
			dx, _ := vm.pop()
			vm.checkWidth(1)
			vm.x += dx
			vm.segments = append(vm.segments, CFFPathSegment{Op: CFFOpMoveTo, Args: [6]int32{dx, 0}})
			vm.stack = vm.stack[:0]

		case 4: // vmoveto
			dy, _ := vm.pop()
			vm.checkWidth(1)
			vm.y += dy
			vm.segments = append(vm.segments, CFFPathSegment{Op: CFFOpMoveTo, Args: [6]int32{0, dy}})
			vm.stack = vm.stack[:0]

		case 5: // rlineto
			dy, _ := vm.pop()
			dx, _ := vm.pop()
			vm.x += dx
			vm.y += dy
			vm.segments = append(vm.segments, CFFPathSegment{Op: CFFOpLineTo, Args: [6]int32{dx, dy}})
			vm.stack = vm.stack[:0]

		case 6: // hlineto
			dx, _ := vm.pop()
			vm.x += dx
			vm.segments = append(vm.segments, CFFPathSegment{Op: CFFOpLineTo, Args: [6]int32{dx, 0}})
			vm.stack = vm.stack[:0]

		case 7: // vlineto
			dy, _ := vm.pop()
			vm.y += dy
			vm.segments = append(vm.segments, CFFPathSegment{Op: CFFOpLineTo, Args: [6]int32{0, dy}})
			vm.stack = vm.stack[:0]

		case 8: // rrcurveto
			if len(vm.stack) < 6 {
				return fmt.Errorf("rrcurveto needs 6 args, got %d", len(vm.stack))
			}
			args := [6]int32{}
			for j := 5; j >= 0; j-- {
				args[j], _ = vm.pop()
			}
			vm.x += args[4]
			vm.y += args[5]
			vm.segments = append(vm.segments, CFFPathSegment{Op: CFFOpCurveTo, Args: args})
			vm.stack = vm.stack[:0]

		case 24: // rcurveline
			// N curves + 1 line: pairs of 6 + final 2
			for len(vm.stack) >= 8 {
				args := [6]int32{}
				for j := 5; j >= 0; j-- {
					args[j], _ = vm.pop()
				}
				vm.x += args[4]
				vm.y += args[5]
				vm.segments = append(vm.segments, CFFPathSegment{Op: CFFOpCurveTo, Args: args})
			}
			if len(vm.stack) >= 2 {
				dy, _ := vm.pop()
				dx, _ := vm.pop()
				vm.x += dx
				vm.y += dy
				vm.segments = append(vm.segments, CFFPathSegment{Op: CFFOpLineTo, Args: [6]int32{dx, dy}})
			}
			vm.stack = vm.stack[:0]

		case 25: // rlinecurve
			// N lines + 1 curve: pairs of 2 + final 6
			for len(vm.stack) >= 8 {
				dy, _ := vm.pop()
				dx, _ := vm.pop()
				vm.x += dx
				vm.y += dy
				vm.segments = append(vm.segments, CFFPathSegment{Op: CFFOpLineTo, Args: [6]int32{dx, dy}})
			}
			if len(vm.stack) >= 6 {
				args := [6]int32{}
				for j := 5; j >= 0; j-- {
					args[j], _ = vm.pop()
				}
				vm.x += args[4]
				vm.y += args[5]
				vm.segments = append(vm.segments, CFFPathSegment{Op: CFFOpCurveTo, Args: args})
			}
			vm.stack = vm.stack[:0]

		case 26: // vvcurveto
			for len(vm.stack) >= 4 {
				args := [6]int32{}
				n := len(vm.stack)
				if n%4 == 0 {
					// dy1, dx2, dy2, dy3
					args[1], _ = vm.pop() // dy1
					args[2], _ = vm.pop() // dx2
					args[3], _ = vm.pop() // dy2
					args[5], _ = vm.pop() // dy3
					args[0] = 0           // dx1
					args[4] = 0           // dx3
				} else {
					// dx1, dy1, dx2, dy2, dy3
					args[5], _ = vm.pop() // dy3
					args[3], _ = vm.pop() // dy2
					args[2], _ = vm.pop() // dx2
					args[1], _ = vm.pop() // dy1
					args[0], _ = vm.pop() // dx1
					args[4] = 0 // dx3
				}
				vm.x += args[4]
				vm.y += args[5]
				vm.segments = append(vm.segments, CFFPathSegment{Op: CFFOpCurveTo, Args: args})
			}
			vm.stack = vm.stack[:0]

		case 27: // hhcurveto
			for len(vm.stack) >= 4 {
				args := [6]int32{}
				n := len(vm.stack)
				if n%4 == 0 {
					// dx1, dx2, dy2, dx3
					args[0], _ = vm.pop() // dx1
					args[2], _ = vm.pop() // dx2
					args[3], _ = vm.pop() // dy2
					args[4], _ = vm.pop() // dx3
					args[1] = 0           // dy1
					args[5] = 0           // dy3
				} else {
					// dy1, dx1, dx2, dy2, dx3
					args[4], _ = vm.pop() // dx3
					args[3], _ = vm.pop() // dy2
					args[2], _ = vm.pop() // dx2
					args[0], _ = vm.pop() // dx1
					args[1], _ = vm.pop() // dy1
					args[5] = 0           // dy3
				}
				vm.x += args[4]
				vm.y += args[5]
				vm.segments = append(vm.segments, CFFPathSegment{Op: CFFOpCurveTo, Args: args})
			}
			vm.stack = vm.stack[:0]

		case 30: // vhcurveto
			for len(vm.stack) >= 4 {
				args := [6]int32{}
				if len(vm.stack) == 4 {
					// dy1, dx2, dy2, dx3
					args[1], _ = vm.pop() // dy1
					args[2], _ = vm.pop() // dx2
					args[3], _ = vm.pop() // dy2
					args[4], _ = vm.pop() // dx3
					args[0] = 0           // dx1
					args[5] = 0           // dy3
				} else {
					// dy1, dx2, dy2, dx3, dy3
					args[5], _ = vm.pop() // dy3
					args[4], _ = vm.pop() // dx3
					args[3], _ = vm.pop() // dy2
					args[2], _ = vm.pop() // dx2
					args[1], _ = vm.pop() // dy1
					args[0] = 0           // dx1
				}
				vm.x += args[4]
				vm.y += args[5]
				vm.segments = append(vm.segments, CFFPathSegment{Op: CFFOpCurveTo, Args: args})
			}
			vm.stack = vm.stack[:0]

		case 31: // hvcurveto
			for len(vm.stack) >= 4 {
				args := [6]int32{}
				if len(vm.stack) == 4 {
					// dx1, dx2, dy2, dy3
					args[0], _ = vm.pop() // dx1
					args[2], _ = vm.pop() // dx2
					args[3], _ = vm.pop() // dy2
					args[5], _ = vm.pop() // dy3
					args[1] = 0           // dy1
					args[4] = 0           // dx3
				} else {
					// dx1, dx2, dy2, dy3, dx3
					args[4], _ = vm.pop() // dx3
					args[5], _ = vm.pop() // dy3
					args[3], _ = vm.pop() // dy2
					args[2], _ = vm.pop() // dx2
					args[0], _ = vm.pop() // dx1
					args[1] = 0           // dy1
				}
				vm.x += args[4]
				vm.y += args[5]
				vm.segments = append(vm.segments, CFFPathSegment{Op: CFFOpCurveTo, Args: args})
			}
			vm.stack = vm.stack[:0]

		case 1: // hstem
			vm.hintState += len(vm.stack) / 2
			vm.stack = vm.stack[:0]
		case 3: // vstem
			vm.hintState += len(vm.stack) / 2
			vm.stack = vm.stack[:0]
		case 18: // hstemhm
			vm.hintState += len(vm.stack) / 2
			vm.stack = vm.stack[:0]
		case 23: // vstemhm
			vm.hintState += len(vm.stack) / 2
			vm.stack = vm.stack[:0]
		case 19: // hintmask
			vm.hintState += len(vm.stack) / 2
			vm.stack = vm.stack[:0]
			bytesNeeded := (vm.hintState + 7) / 8
			i += bytesNeeded
			continue
		case 20: // cntrmask
			vm.hintState += len(vm.stack) / 2
			vm.stack = vm.stack[:0]
			bytesNeeded := (vm.hintState + 7) / 8
			i += bytesNeeded
			continue

		case 10: // callsubr
			subrNum, ok := vm.pop()
			if !ok {
				return errors.New("callsubr: stack underflow")
			}
			subrData, err := indexElement(vm.localSubrs, vm.subrBias(vm.localSubrs)+int(subrNum))
			if err != nil {
				return fmt.Errorf("callsubr: %w", err)
			}
			if err := vm.execute(subrData, depth+1); err != nil {
				return err
			}

		case 29: // callgsubr
			subrNum, ok := vm.pop()
			if !ok {
				return errors.New("callgsubr: stack underflow")
			}
			subrData, err := indexElement(vm.globalSubrs, vm.subrBias(vm.globalSubrs)+int(subrNum))
			if err != nil {
				return fmt.Errorf("callgsubr: %w", err)
			}
			if err := vm.execute(subrData, depth+1); err != nil {
				return err
			}

		case 11: // return
			return nil

		default:
			// Skip unknown operators
			i++
			continue
		}
		i++
	}

	return nil
}

// checkWidth handles the width operand that appears before moveto operators.
func (vm *charstringVM) checkWidth(expectedArgs int) {
	if !vm.widthSet && len(vm.stack) == expectedArgs+1 {
		w, _ := vm.pop()
		vm.width = vm.nominalWidth()
		vm.hasWidth = true
		vm.widthSet = true
		_ = w
	}
}

// nominalWidth returns nominalWidthX + width operand (if specified).
func (vm *charstringVM) nominalWidth() int32 {
	if len(vm.stack) > 0 && !vm.widthSet {
		return vm.nominalWidthX + vm.stack[len(vm.stack)-1]
	}
	return vm.defaultWidthX
}

// subrBias returns the subroutine bias for the given INDEX.
func (vm *charstringVM) subrBias(idx *CFFINDEX) int {
	count := int(idx.count)
	if count < 1240 {
		return 107
	}
	if count < 33900 {
		return 1131
	}
	return 32768
}

// --- CFF DecodeOutlines method ---

// DecodeOutlines decodes all charstring outlines in the CFF font.
func (cff *CFF) DecodeOutlines() ([]*CFFOutline, error) {
	n := cff.NumGlyphs()
	outlines := make([]*CFFOutline, n)

	for i := 0; i < n; i++ {
		csData, err := indexElement(&cff.charStrings, i)
		if err != nil {
			return nil, fmt.Errorf("glyph %d: %w", i, err)
		}

		outlines[i], err = decodeCharString(
			csData,
			&cff.globalSubrs,
			&cff.localSubrs,
			cff.privateDict.defaultWidthX,
			cff.privateDict.nominalWidthX,
		)
		if err != nil {
			return nil, fmt.Errorf("glyph %d charstring: %w", i, err)
		}
	}

	return outlines, nil
}
