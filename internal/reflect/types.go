// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package reflect

import (
	"reflect"
	"strconv"
)

// IndirectType returns the underlying type after dereferencing all pointer levels.
// If the type is not a pointer, it returns the type unchanged.
//
// Example:
//
//	t := IndirectType(reflect.TypeOf(&User{})) // returns reflect.Type of User
func IndirectType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	return t
}

// IndirectValue follows every non-nil pointer layer in v. It returns an
// invalid Value when any pointer in the chain is nil.
func IndirectValue(v reflect.Value) reflect.Value {
	for v.IsValid() && v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return reflect.Value{}
		}

		v = v.Elem()
	}

	return v
}

// MakeDst ensures the destination reflect.Value is properly initialized for assignment.
// If the destination is a pointer type, it creates a new instance and redirects the Value
// to point to the newly created object's element. This is necessary for proper assignment
// when converting to pointer types.
func MakeDst(dst *reflect.Value, t reflect.Type) {
	for dst.IsValid() && dst.Kind() == reflect.Pointer {
		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}

		*dst = dst.Elem()
	}
}

// IsPrimitive checks if a kind represents a primitive type.
// Primitive types include all integers, unsigned integers, floats, bool, and string.
func IsPrimitive(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Bool, reflect.String:
		return true
	default:
		return false
	}
}

// ConvertValue converts v to t where Go allows it, fixing the conversions
// whose reflect semantics would lose the value's meaning:
//   - integer -> string formats the number in decimal; a plain Convert
//     would yield the UTF-8 encoding of the code point (65 -> "A").
//   - slice -> array (or array pointer) is refused when the slice is shorter
//     than the array; a plain Convert would panic.
//
// string <-> []byte and string <-> []rune keep Go's semantics (UTF-8 bytes,
// decoded code points). It reports false when v cannot be converted to t.
func ConvertValue(v reflect.Value, t reflect.Type) (reflect.Value, bool) {
	if t.Kind() == reflect.String {
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return reflect.ValueOf(strconv.FormatInt(v.Int(), 10)).Convert(t), true
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			return reflect.ValueOf(strconv.FormatUint(v.Uint(), 10)).Convert(t), true
		default:
		}
	}

	if !v.CanConvert(t) {
		return reflect.Value{}, false
	}

	return v.Convert(t), true
}

// SetZero sets v to its zero value when v is valid and settable.
func SetZero(v reflect.Value) {
	if v.IsValid() && v.CanSet() {
		v.SetZero()
	}
}
