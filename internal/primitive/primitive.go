// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package primitive

import (
	"reflect"
	"unsafe"

	"github.com/altessa-s/go-convert/reflectx"
)

// Func converts the primitive value at src into the primitive value at dst.
//
// Both pointers must address values whose kinds match the table entry the
// function was registered under. [Convert] guarantees this: it obtains the
// pointers from addressable reflect values of exactly those kinds, so the
// casts never reinterpret memory of a different layout.
type Func func(src, dst unsafe.Pointer)

// table indexes conversion functions by source and destination kind. A flat
// array avoids hashing on the hot path.
var table = buildTable()

// Lookup returns the conversion function for the kind pair.
func Lookup(from, to reflect.Kind) (Func, bool) {
	if from >= numKinds || to >= numKinds {
		return nil, false
	}

	fn := table[from][to]

	return fn, fn != nil
}

// Convert writes src into dst when both are primitive kinds with a
// registered conversion. A nil destination pointer is allocated unless src
// is the zero value. With checkOverflow set, a lossy narrowing conversion
// writes nothing and reports overflow.
func Convert(src, dst reflect.Value, checkOverflow bool) (handled, overflow bool) {
	srcVal := reflectx.IndirectValue(src)
	if !srcVal.IsValid() || !reflectx.IsPrimitive(srcVal.Kind()) {
		return false, false
	}

	dstType := reflectx.IndirectType(dst.Type())

	fn, ok := Lookup(srcVal.Kind(), dstType.Kind())
	if !ok {
		return false, false
	}

	dstVal := dst
	if dst.Kind() == reflect.Pointer && dst.IsNil() {
		if srcVal.IsZero() || !dst.CanSet() {
			return false, false
		}

		reflectx.MakeDst(&dstVal, dstType)
	}

	dstVal = reflectx.IndirectValue(dstVal)
	if !dstVal.IsValid() || !dstVal.CanSet() || !srcVal.CanAddr() || !dstVal.CanAddr() {
		return false, false
	}

	if checkOverflow && WouldOverflow(srcVal, dstVal) {
		return false, true
	}

	fn(unsafe.Pointer(srcVal.UnsafeAddr()), unsafe.Pointer(dstVal.UnsafeAddr())) // #nosec G103 -- see Func

	return true, false
}

const numKinds = reflect.UnsafePointer + 1

// buildTable indexes the conversion functions by source and destination kind.
func buildTable() *[numKinds][numKinds]Func {
	pairs := map[[2]reflect.Kind]Func{
		// Int conversions
		{reflect.Int, reflect.Int8}:    func(src, dst unsafe.Pointer) { *(*int8)(dst) = int8(*(*int)(src)) },
		{reflect.Int, reflect.Int16}:   func(src, dst unsafe.Pointer) { *(*int16)(dst) = int16(*(*int)(src)) },
		{reflect.Int, reflect.Int32}:   func(src, dst unsafe.Pointer) { *(*int32)(dst) = int32(*(*int)(src)) },
		{reflect.Int, reflect.Int64}:   func(src, dst unsafe.Pointer) { *(*int64)(dst) = int64(*(*int)(src)) },
		{reflect.Int8, reflect.Int}:    func(src, dst unsafe.Pointer) { *(*int)(dst) = int(*(*int8)(src)) },
		{reflect.Int8, reflect.Int16}:  func(src, dst unsafe.Pointer) { *(*int16)(dst) = int16(*(*int8)(src)) },
		{reflect.Int8, reflect.Int32}:  func(src, dst unsafe.Pointer) { *(*int32)(dst) = int32(*(*int8)(src)) },
		{reflect.Int8, reflect.Int64}:  func(src, dst unsafe.Pointer) { *(*int64)(dst) = int64(*(*int8)(src)) },
		{reflect.Int16, reflect.Int}:   func(src, dst unsafe.Pointer) { *(*int)(dst) = int(*(*int16)(src)) },
		{reflect.Int16, reflect.Int8}:  func(src, dst unsafe.Pointer) { *(*int8)(dst) = int8(*(*int16)(src)) },
		{reflect.Int16, reflect.Int32}: func(src, dst unsafe.Pointer) { *(*int32)(dst) = int32(*(*int16)(src)) },
		{reflect.Int16, reflect.Int64}: func(src, dst unsafe.Pointer) { *(*int64)(dst) = int64(*(*int16)(src)) },
		{reflect.Int32, reflect.Int}:   func(src, dst unsafe.Pointer) { *(*int)(dst) = int(*(*int32)(src)) },
		{reflect.Int32, reflect.Int8}:  func(src, dst unsafe.Pointer) { *(*int8)(dst) = int8(*(*int32)(src)) },
		{reflect.Int32, reflect.Int16}: func(src, dst unsafe.Pointer) { *(*int16)(dst) = int16(*(*int32)(src)) },
		{reflect.Int32, reflect.Int64}: func(src, dst unsafe.Pointer) { *(*int64)(dst) = int64(*(*int32)(src)) },
		{reflect.Int64, reflect.Int}:   func(src, dst unsafe.Pointer) { *(*int)(dst) = int(*(*int64)(src)) },
		{reflect.Int64, reflect.Int8}:  func(src, dst unsafe.Pointer) { *(*int8)(dst) = int8(*(*int64)(src)) },
		{reflect.Int64, reflect.Int16}: func(src, dst unsafe.Pointer) { *(*int16)(dst) = int16(*(*int64)(src)) },
		{reflect.Int64, reflect.Int32}: func(src, dst unsafe.Pointer) { *(*int32)(dst) = int32(*(*int64)(src)) },

		// Uint conversions
		{reflect.Uint, reflect.Uint8}:    func(src, dst unsafe.Pointer) { *(*uint8)(dst) = uint8(*(*uint)(src)) },
		{reflect.Uint, reflect.Uint16}:   func(src, dst unsafe.Pointer) { *(*uint16)(dst) = uint16(*(*uint)(src)) },
		{reflect.Uint, reflect.Uint32}:   func(src, dst unsafe.Pointer) { *(*uint32)(dst) = uint32(*(*uint)(src)) },
		{reflect.Uint, reflect.Uint64}:   func(src, dst unsafe.Pointer) { *(*uint64)(dst) = uint64(*(*uint)(src)) },
		{reflect.Uint8, reflect.Uint}:    func(src, dst unsafe.Pointer) { *(*uint)(dst) = uint(*(*uint8)(src)) },
		{reflect.Uint8, reflect.Uint16}:  func(src, dst unsafe.Pointer) { *(*uint16)(dst) = uint16(*(*uint8)(src)) },
		{reflect.Uint8, reflect.Uint32}:  func(src, dst unsafe.Pointer) { *(*uint32)(dst) = uint32(*(*uint8)(src)) },
		{reflect.Uint8, reflect.Uint64}:  func(src, dst unsafe.Pointer) { *(*uint64)(dst) = uint64(*(*uint8)(src)) },
		{reflect.Uint16, reflect.Uint}:   func(src, dst unsafe.Pointer) { *(*uint)(dst) = uint(*(*uint16)(src)) },
		{reflect.Uint16, reflect.Uint8}:  func(src, dst unsafe.Pointer) { *(*uint8)(dst) = uint8(*(*uint16)(src)) },
		{reflect.Uint16, reflect.Uint32}: func(src, dst unsafe.Pointer) { *(*uint32)(dst) = uint32(*(*uint16)(src)) },
		{reflect.Uint16, reflect.Uint64}: func(src, dst unsafe.Pointer) { *(*uint64)(dst) = uint64(*(*uint16)(src)) },
		{reflect.Uint32, reflect.Uint}:   func(src, dst unsafe.Pointer) { *(*uint)(dst) = uint(*(*uint32)(src)) },
		{reflect.Uint32, reflect.Uint8}:  func(src, dst unsafe.Pointer) { *(*uint8)(dst) = uint8(*(*uint32)(src)) },
		{reflect.Uint32, reflect.Uint16}: func(src, dst unsafe.Pointer) { *(*uint16)(dst) = uint16(*(*uint32)(src)) },
		{reflect.Uint32, reflect.Uint64}: func(src, dst unsafe.Pointer) { *(*uint64)(dst) = uint64(*(*uint32)(src)) },
		{reflect.Uint64, reflect.Uint}:   func(src, dst unsafe.Pointer) { *(*uint)(dst) = uint(*(*uint64)(src)) },
		{reflect.Uint64, reflect.Uint8}:  func(src, dst unsafe.Pointer) { *(*uint8)(dst) = uint8(*(*uint64)(src)) },
		{reflect.Uint64, reflect.Uint16}: func(src, dst unsafe.Pointer) { *(*uint16)(dst) = uint16(*(*uint64)(src)) },
		{reflect.Uint64, reflect.Uint32}: func(src, dst unsafe.Pointer) { *(*uint32)(dst) = uint32(*(*uint64)(src)) },

		// Float conversions
		{reflect.Float32, reflect.Float64}: func(src, dst unsafe.Pointer) { *(*float64)(dst) = float64(*(*float32)(src)) },
		{reflect.Float64, reflect.Float32}: func(src, dst unsafe.Pointer) { *(*float32)(dst) = float32(*(*float64)(src)) },

		// Int to Float conversions
		{reflect.Int, reflect.Float32}:   func(src, dst unsafe.Pointer) { *(*float32)(dst) = float32(*(*int)(src)) },
		{reflect.Int, reflect.Float64}:   func(src, dst unsafe.Pointer) { *(*float64)(dst) = float64(*(*int)(src)) },
		{reflect.Int8, reflect.Float32}:  func(src, dst unsafe.Pointer) { *(*float32)(dst) = float32(*(*int8)(src)) },
		{reflect.Int8, reflect.Float64}:  func(src, dst unsafe.Pointer) { *(*float64)(dst) = float64(*(*int8)(src)) },
		{reflect.Int16, reflect.Float32}: func(src, dst unsafe.Pointer) { *(*float32)(dst) = float32(*(*int16)(src)) },
		{reflect.Int16, reflect.Float64}: func(src, dst unsafe.Pointer) { *(*float64)(dst) = float64(*(*int16)(src)) },
		{reflect.Int32, reflect.Float32}: func(src, dst unsafe.Pointer) { *(*float32)(dst) = float32(*(*int32)(src)) },
		{reflect.Int32, reflect.Float64}: func(src, dst unsafe.Pointer) { *(*float64)(dst) = float64(*(*int32)(src)) },
		{reflect.Int64, reflect.Float32}: func(src, dst unsafe.Pointer) { *(*float32)(dst) = float32(*(*int64)(src)) },
		{reflect.Int64, reflect.Float64}: func(src, dst unsafe.Pointer) { *(*float64)(dst) = float64(*(*int64)(src)) },

		// Uint to Float conversions
		{reflect.Uint, reflect.Float32}:   func(src, dst unsafe.Pointer) { *(*float32)(dst) = float32(*(*uint)(src)) },
		{reflect.Uint, reflect.Float64}:   func(src, dst unsafe.Pointer) { *(*float64)(dst) = float64(*(*uint)(src)) },
		{reflect.Uint8, reflect.Float32}:  func(src, dst unsafe.Pointer) { *(*float32)(dst) = float32(*(*uint8)(src)) },
		{reflect.Uint8, reflect.Float64}:  func(src, dst unsafe.Pointer) { *(*float64)(dst) = float64(*(*uint8)(src)) },
		{reflect.Uint16, reflect.Float32}: func(src, dst unsafe.Pointer) { *(*float32)(dst) = float32(*(*uint16)(src)) },
		{reflect.Uint16, reflect.Float64}: func(src, dst unsafe.Pointer) { *(*float64)(dst) = float64(*(*uint16)(src)) },
		{reflect.Uint32, reflect.Float32}: func(src, dst unsafe.Pointer) { *(*float32)(dst) = float32(*(*uint32)(src)) },
		{reflect.Uint32, reflect.Float64}: func(src, dst unsafe.Pointer) { *(*float64)(dst) = float64(*(*uint32)(src)) },
		{reflect.Uint64, reflect.Float32}: func(src, dst unsafe.Pointer) { *(*float32)(dst) = float32(*(*uint64)(src)) },
		{reflect.Uint64, reflect.Float64}: func(src, dst unsafe.Pointer) { *(*float64)(dst) = float64(*(*uint64)(src)) },

		// Float to Int conversions
		{reflect.Float32, reflect.Int}:   func(src, dst unsafe.Pointer) { *(*int)(dst) = int(*(*float32)(src)) },
		{reflect.Float32, reflect.Int8}:  func(src, dst unsafe.Pointer) { *(*int8)(dst) = int8(*(*float32)(src)) },
		{reflect.Float32, reflect.Int16}: func(src, dst unsafe.Pointer) { *(*int16)(dst) = int16(*(*float32)(src)) },
		{reflect.Float32, reflect.Int32}: func(src, dst unsafe.Pointer) { *(*int32)(dst) = int32(*(*float32)(src)) },
		{reflect.Float32, reflect.Int64}: func(src, dst unsafe.Pointer) { *(*int64)(dst) = int64(*(*float32)(src)) },
		{reflect.Float64, reflect.Int}:   func(src, dst unsafe.Pointer) { *(*int)(dst) = int(*(*float64)(src)) },
		{reflect.Float64, reflect.Int8}:  func(src, dst unsafe.Pointer) { *(*int8)(dst) = int8(*(*float64)(src)) },
		{reflect.Float64, reflect.Int16}: func(src, dst unsafe.Pointer) { *(*int16)(dst) = int16(*(*float64)(src)) },
		{reflect.Float64, reflect.Int32}: func(src, dst unsafe.Pointer) { *(*int32)(dst) = int32(*(*float64)(src)) },
		{reflect.Float64, reflect.Int64}: func(src, dst unsafe.Pointer) { *(*int64)(dst) = int64(*(*float64)(src)) },

		// Float to Uint conversions
		{reflect.Float32, reflect.Uint}:   func(src, dst unsafe.Pointer) { *(*uint)(dst) = uint(*(*float32)(src)) },
		{reflect.Float32, reflect.Uint8}:  func(src, dst unsafe.Pointer) { *(*uint8)(dst) = uint8(*(*float32)(src)) },
		{reflect.Float32, reflect.Uint16}: func(src, dst unsafe.Pointer) { *(*uint16)(dst) = uint16(*(*float32)(src)) },
		{reflect.Float32, reflect.Uint32}: func(src, dst unsafe.Pointer) { *(*uint32)(dst) = uint32(*(*float32)(src)) },
		{reflect.Float32, reflect.Uint64}: func(src, dst unsafe.Pointer) { *(*uint64)(dst) = uint64(*(*float32)(src)) },
		{reflect.Float64, reflect.Uint}:   func(src, dst unsafe.Pointer) { *(*uint)(dst) = uint(*(*float64)(src)) },
		{reflect.Float64, reflect.Uint8}:  func(src, dst unsafe.Pointer) { *(*uint8)(dst) = uint8(*(*float64)(src)) },
		{reflect.Float64, reflect.Uint16}: func(src, dst unsafe.Pointer) { *(*uint16)(dst) = uint16(*(*float64)(src)) },
		{reflect.Float64, reflect.Uint32}: func(src, dst unsafe.Pointer) { *(*uint32)(dst) = uint32(*(*float64)(src)) },
		{reflect.Float64, reflect.Uint64}: func(src, dst unsafe.Pointer) { *(*uint64)(dst) = uint64(*(*float64)(src)) },

		// Same-kind conversions between distinct defined types.
		{reflect.Bool, reflect.Bool}:     func(src, dst unsafe.Pointer) { *(*bool)(dst) = *(*bool)(src) },
		{reflect.String, reflect.String}: func(src, dst unsafe.Pointer) { *(*string)(dst) = *(*string)(src) },
	}

	var t [numKinds][numKinds]Func
	for kinds, fn := range pairs {
		t[kinds[0]][kinds[1]] = fn
	}

	return &t
}

// WouldOverflow reports whether converting srcVal to the type of dstVal loses data.
func WouldOverflow(srcVal, dstVal reflect.Value) bool {
	srcKind := srcVal.Kind()
	dstKind := dstVal.Kind()

	switch {
	// int → int narrowing
	case isSignedInt(dstKind) && isSignedInt(srcKind):
		return reflect.New(dstVal.Type()).Elem().OverflowInt(srcVal.Int())

	// uint → uint narrowing
	case isUnsignedInt(dstKind) && isUnsignedInt(srcKind):
		return reflect.New(dstVal.Type()).Elem().OverflowUint(srcVal.Uint())

	// float → float narrowing (float64 → float32)
	case isFloat(dstKind) && isFloat(srcKind):
		return reflect.New(dstVal.Type()).Elem().OverflowFloat(srcVal.Float())

	// int → uint: negative values overflow
	case isUnsignedInt(dstKind) && isSignedInt(srcKind):
		v := srcVal.Int()
		if v < 0 {
			return true
		}

		return reflect.New(dstVal.Type()).Elem().OverflowUint(uint64(v))

	// uint → int: check if value exceeds signed max
	case isSignedInt(dstKind) && isUnsignedInt(srcKind):
		v := srcVal.Uint()
		if v > uint64(^uint(0)>>1) { // exceeds max int64
			return true
		}

		return reflect.New(dstVal.Type()).Elem().OverflowInt(int64(v))

	// float → int
	case isSignedInt(dstKind) && isFloat(srcKind):
		f := srcVal.Float()

		i := int64(f)
		if float64(i) != f {
			return true
		}

		return reflect.New(dstVal.Type()).Elem().OverflowInt(i)

	// float → uint
	case isUnsignedInt(dstKind) && isFloat(srcKind):
		f := srcVal.Float()
		if f < 0 {
			return true
		}

		u := uint64(f)
		if float64(u) != f {
			return true
		}

		return reflect.New(dstVal.Type()).Elem().OverflowUint(u)

	// int → float: check precision loss for float32
	case isFloat(dstKind) && isSignedInt(srcKind):
		if dstKind == reflect.Float32 {
			return reflect.New(dstVal.Type()).Elem().OverflowFloat(float64(srcVal.Int()))
		}

		return false

	// uint → float: check precision loss for float32
	case isFloat(dstKind) && isUnsignedInt(srcKind):
		if dstKind == reflect.Float32 {
			return reflect.New(dstVal.Type()).Elem().OverflowFloat(float64(srcVal.Uint()))
		}

		return false
	}

	return false
}

// isSignedInt reports whether k is a signed integer kind.
func isSignedInt(k reflect.Kind) bool {
	return k >= reflect.Int && k <= reflect.Int64
}

// isUnsignedInt reports whether k is an unsigned integer kind.
func isUnsignedInt(k reflect.Kind) bool {
	return k >= reflect.Uint && k <= reflect.Uint64 || k == reflect.Uintptr
}

// isFloat reports whether k is a floating-point kind.
func isFloat(k reflect.Kind) bool {
	return k == reflect.Float32 || k == reflect.Float64
}
