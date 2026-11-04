// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package engine

import "reflect"

// scalarClass groups the kinds [setScalar] converts between.
type scalarClass uint8

const (
	scalarNone scalarClass = iota
	scalarInt
	scalarUint
	scalarFloat
	scalarBool
	scalarString
)

// scalarClassOf returns the scalar class of k.
func scalarClassOf(k reflect.Kind) scalarClass {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return scalarInt
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return scalarUint
	case reflect.Float32, reflect.Float64:
		return scalarFloat
	case reflect.Bool:
		return scalarBool
	case reflect.String:
		return scalarString
	default:
		return scalarNone
	}
}

// isScalarPair reports whether [setScalar] converts a src kind value into a
// dst kind value. Integer to string is excluded, as it is formatted in
// decimal rather than converted, and so is float32 to float32, which
// [reflect.Value.Convert] copies bit for bit to preserve NaN payloads.
func isScalarPair(src, dst reflect.Kind) bool {
	srcClass, dstClass := scalarClassOf(src), scalarClassOf(dst)
	switch srcClass {
	case scalarInt, scalarUint:
		return dstClass == scalarInt || dstClass == scalarUint || dstClass == scalarFloat
	case scalarFloat:
		if src == reflect.Float32 && dst == reflect.Float32 {
			return false
		}

		return dstClass == scalarInt || dstClass == scalarUint || dstClass == scalarFloat
	case scalarBool, scalarString:
		return dstClass == srcClass
	default:
		return false
	}
}

// setScalar stores src converted to the type of dst in dst, producing the
// same value as dst.Set(src.Convert(dst.Type())) without allocating the
// intermediate value, except that integers round to float32 once, like a Go
// conversion, instead of through float64. The kinds must satisfy [isScalarPair], src must be
// exported and dst settable. Each case mirrors the conversion function
// reflect selects for the kind pair (cvtInt, cvtUint, cvtIntFloat,
// cvtUintFloat, cvtFloatInt, cvtFloatUint, cvtFloat and cvtDirect).
func setScalar(src, dst reflect.Value) {
	dstClass := scalarClassOf(dst.Kind())

	switch scalarClassOf(src.Kind()) {
	case scalarInt:
		f := float64(src.Int())
		if dst.Kind() == reflect.Float32 {
			f = float64(float32(src.Int()))
		}

		setNumber(dst, dstClass, src.Int(), uint64(src.Int()), f)
	case scalarUint:
		f := float64(src.Uint())
		if dst.Kind() == reflect.Float32 {
			f = float64(float32(src.Uint()))
		}

		setNumber(dst, dstClass, int64(src.Uint()), src.Uint(), f)
	case scalarFloat:
		setNumber(dst, dstClass, int64(src.Float()), uint64(src.Float()), src.Float())
	case scalarBool:
		dst.SetBool(src.Bool())
	default:
		dst.SetString(src.String())
	}
}

// setNumber stores the representation of one number matching the class of
// dst: i for signed, u for unsigned and f for floating-point destinations.
func setNumber(dst reflect.Value, dstClass scalarClass, i int64, u uint64, f float64) {
	switch dstClass {
	case scalarInt:
		dst.SetInt(i)
	case scalarUint:
		dst.SetUint(u)
	default:
		dst.SetFloat(f)
	}
}
