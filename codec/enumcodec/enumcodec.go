// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package enumcodec

import (
	"fmt"
	"math"
	"reflect"
	"strings"

	convcodec "github.com/altessa-s/go-convert/codec"
	reflectutils "github.com/altessa-s/go-convert/internal/reflect"
)

// UnknownValueError reports an enum value, string or integer that has no
// entry in the codec's table. It is returned unchanged by converter.ConvertE.
type UnknownValueError struct {
	// Path is the full source field path.
	Path string
	// Enum is the enum type of the codec.
	Enum reflect.Type
	// Value is the offending source value: an enum value when formatting, a
	// string or integer when parsing.
	Value any
}

// Error reports the field path, the enum type and the unknown value.
func (e UnknownValueError) Error() string {
	return fmt.Sprintf("enumcodec: field %q: unknown %s value %#v", e.Path, e.Enum, e.Value)
}

// RangeError reports a table integer that does not fit the destination's
// integer type, such as 300 written into a uint8 field.
type RangeError struct {
	// Path is the full source field path.
	Path string
	// Value is the table integer.
	Value int64
	// Kind is the destination integer kind.
	Kind reflect.Kind
}

// Error reports the field path and the value that does not fit the kind.
func (e RangeError) Error() string {
	return fmt.Sprintf("enumcodec: field %q: value %d overflows %s", e.Path, e.Value, e.Kind)
}

// New returns an error codec that converts the enum type E to and from
// strings using names, which maps every known enum value to its spelling.
//
// The codec claims a field pair only when one side is exactly E or *E and the
// other is the predeclared string type (or a pointer to it); every other
// pair, including one with another defined type such as a second enum, is
// delegated to next. Formatting looks the enum value up in
// names; parsing looks the string up in the reverse table, ignoring case with
// [WithCaseInsensitive]. A value missing from the table is an
// [UnknownValueError] unless [WithUnspecified] or [WithUnknownAsZero] applies.
// A nil pointer source zeroes the destination.
//
// New panics when two enum values share a spelling (compared case-insensitively
// with [WithCaseInsensitive]), because parsing would be ambiguous.
//
// Example:
//
//	codec := enumcodec.New(map[Color]string{Red: "red", Green: "green"}, enumcodec.WithUnspecified())
//	converter.ConvertE(src, &dst, converter.WithErrorCodecs(codec))
func New[E comparable](names map[E]string, opts ...Option) convcodec.CodecE {
	o := newOptions(opts...)

	c := newTable(names, o, isStringKind, readString, writeString)
	if o.caseInsensitive {
		c.folded = make(map[string]E, len(names))
		for e, name := range names {
			key := strings.ToLower(name)
			if _, dup := c.folded[key]; dup {
				panic(fmt.Sprintf("enumcodec: %s names %q collide ignoring case", c.enumType, name))
			}

			c.folded[key] = e
		}
	}

	return c.codec
}

// NewInt returns an error codec that converts the enum type E to and from
// integers using values, which maps every known enum value to its number.
//
// The codec claims a field pair only when one side is exactly E or *E and the
// other is a predeclared signed or unsigned integer type, such as int32 (or a
// pointer to one); every other pair, including one with another defined type
// such as a second enum, is delegated to next. An integer missing
// from the table is an [UnknownValueError] unless [WithUnspecified] or
// [WithUnknownAsZero] applies; a table integer that does not fit the
// destination is a [RangeError]. A nil pointer source zeroes the destination.
//
// NewInt panics when two enum values share a number, and when
// [WithCaseInsensitive] is given, because integers have no case.
//
// Example:
//
//	codec := enumcodec.NewInt(map[Color]int64{Red: 1, Green: 2})
func NewInt[E comparable](values map[E]int64, opts ...Option) convcodec.CodecE {
	o := newOptions(opts...)
	if o.caseInsensitive {
		panic(fmt.Sprintf("enumcodec: WithCaseInsensitive does not apply to the integer codec of %s", reflect.TypeFor[E]()))
	}

	return newTable(values, o, isIntegerKind, readInteger, writeInteger).codec
}

// table holds both directions of an enum mapping onto wire values of type W
// (string or int64) and the kind-specific accessors for W.
type table[E, W comparable] struct {
	enumType reflect.Type
	forward  map[E]W
	reverse  map[W]E
	folded   map[string]E
	opts     *options
	// isWire reports whether a kind can hold wire values.
	isWire func(reflect.Kind) bool
	// read extracts a wire value; false means it cannot be in the table.
	read func(reflect.Value) (W, bool)
	// write stores a wire value into a non-pointer destination.
	write func(path string, dst reflect.Value, w W) error
}

// newTable builds the forward and reverse lookups, panicking on duplicates.
func newTable[E, W comparable](
	forward map[E]W, opts *options,
	isWire func(reflect.Kind) bool,
	read func(reflect.Value) (W, bool),
	write func(string, reflect.Value, W) error,
) *table[E, W] {
	t := &table[E, W]{
		enumType: reflect.TypeFor[E](),
		forward:  make(map[E]W, len(forward)),
		reverse:  make(map[W]E, len(forward)),
		opts:     opts,
		isWire:   isWire,
		read:     read,
		write:    write,
	}
	for e, w := range forward {
		if _, dup := t.reverse[w]; dup {
			panic(fmt.Sprintf("enumcodec: %s wire value %v is mapped more than once", t.enumType, w))
		}

		t.forward[e] = w
		t.reverse[w] = e
	}

	return t
}

// codec is the [convcodec.CodecE] of the table.
func (t *table[E, W]) codec(path string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
	srcType, dstType := src.Type(), dst.Type()
	switch {
	case t.isEnum(srcType) && t.isWireType(dstType):
		return t.format(path, src, dst)
	case t.isWireType(srcType) && t.isEnum(dstType):
		return t.parse(path, src, dst)
	default:
		return next(path, src, dst)
	}
}

// isEnum reports whether typ is E or *E.
func (t *table[E, W]) isEnum(typ reflect.Type) bool {
	return typ == t.enumType || typ.Kind() == reflect.Pointer && typ.Elem() == t.enumType
}

// isWireType reports whether typ, or the type it points to, is a predeclared
// type of a wire kind, such as string or int32, and not the enum type itself.
// Defined types, such as another enum, are never wire types, so a pair of
// enums is not mapped through this table.
func (t *table[E, W]) isWireType(typ reflect.Type) bool {
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	return typ != t.enumType && typ.PkgPath() == "" && typ.Name() != "" && t.isWire(typ.Kind())
}

// format converts an enum source into a wire destination.
func (t *table[E, W]) format(path string, src, dst reflect.Value) error {
	if src.Kind() == reflect.Pointer {
		if src.IsNil() {
			dst.SetZero()
			return nil
		}

		src = src.Elem()
	}

	e, isEnum := reflect.TypeAssert[E](src)
	if !isEnum {
		return UnknownValueError{Path: path, Enum: t.enumType, Value: src.Interface()}
	}

	w, ok := t.forward[e]
	if !ok {
		var zeroEnum E
		if !t.opts.unknownAsZero && (!t.opts.unspecified || e != zeroEnum) {
			return UnknownValueError{Path: path, Enum: t.enumType, Value: e}
		}
	}

	reflectutils.MakeDst(&dst, dst.Type())

	return t.write(path, dst, w)
}

// parse converts a wire source into an enum destination.
func (t *table[E, W]) parse(path string, src, dst reflect.Value) error {
	if src.Kind() == reflect.Pointer {
		if src.IsNil() {
			dst.SetZero()
			return nil
		}

		src = src.Elem()
	}

	e, ok := t.lookup(src)
	if !ok && !t.opts.unknownAsZero {
		return UnknownValueError{Path: path, Enum: t.enumType, Value: src.Interface()}
	}

	reflectutils.MakeDst(&dst, dst.Type())

	if dst.CanAddr() {
		if p, isEnum := reflect.TypeAssert[*E](dst.Addr()); isEnum {
			*p = e
			return nil
		}
	}

	setValue(dst, e)

	return nil
}

// setValue stores e into a non-addressable destination. It is kept apart
// from parse so that e only escapes to the heap on this rare path.
func setValue[E any](dst reflect.Value, e E) {
	dst.Set(reflect.ValueOf(&e).Elem())
}

// lookup finds the enum value of a wire source. The zero enum value is
// returned for the zero wire value with [WithUnspecified].
func (t *table[E, W]) lookup(src reflect.Value) (E, bool) {
	var zero E

	w, ok := t.read(src)
	if !ok {
		return zero, false
	}

	if e, found := t.reverse[w]; found {
		return e, true
	}

	if t.folded != nil {
		if e, found := t.folded[strings.ToLower(src.String())]; found {
			return e, true
		}
	}

	var zeroWire W

	return zero, t.opts.unspecified && w == zeroWire
}

// isStringKind reports whether k is the string kind.
func isStringKind(k reflect.Kind) bool { return k == reflect.String }

// isIntegerKind reports whether k is a signed or unsigned integer kind.
func isIntegerKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

// readString reads the wire value of a string-kind field.
func readString(v reflect.Value) (string, bool) { return v.String(), true }

// writeString stores s in a string-kind destination.
func writeString(_ string, dst reflect.Value, s string) error {
	dst.SetString(s)
	return nil
}

// readInteger reads any integer kind as int64; unsigned values above the
// int64 range cannot be in the table.
func readInteger(v reflect.Value) (int64, bool) {
	if v.CanInt() {
		return v.Int(), true
	}

	u := v.Uint()
	if u > math.MaxInt64 {
		return 0, false
	}

	return int64(u), true
}

// writeInteger stores n in an integer-kind destination, failing when it does not fit.
func writeInteger(path string, dst reflect.Value, n int64) error {
	if dst.CanInt() {
		if dst.OverflowInt(n) {
			return RangeError{Path: path, Value: n, Kind: dst.Kind()}
		}

		dst.SetInt(n)

		return nil
	}

	if n < 0 || dst.OverflowUint(uint64(n)) {
		return RangeError{Path: path, Value: n, Kind: dst.Kind()}
	}

	dst.SetUint(uint64(n))

	return nil
}
