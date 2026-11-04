// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package sqlnull

import (
	"reflect"
	"strings"

	convcodec "github.com/altessa-s/go-convert/codec"
)

const (
	sqlPackage = "database/sql"
	// valueField and validField are the field indexes shared by every
	// database/sql null type: the value first, then Valid.
	valueField = 0
	validField = 1
	nullFields = 2
)

// New returns an error codec that bridges database/sql null types and their
// plain counterparts. It recognizes the generic sql.Null[T] and the legacy
// sql.NullString, NullInt64, NullInt32, NullInt16, NullByte, NullFloat64,
// NullBool and NullTime; below, N is such a type with value type T.
//
// Direct shapes, all in both directions:
//
//   - N ↔ *T: Valid=false ↔ nil pointer; a valid value ↔ a pointer to it.
//   - N ↔ T:  Valid=false ↔ zero T (see [WithZeroAsNull]); a valid value ↔ T.
//   - N ↔ N2: two different null types; Valid=false stays invalid, a valid
//     value is converted T → T2 by the rest of the chain (a zero T2 is
//     stored as Valid=false with [WithZeroAsNull]).
//
// Composition: when one side is N and the other is some type W the codec does
// not bridge directly, it unwraps / wraps N and delegates the T ↔ W conversion
// to the rest of the chain, as codec/optionalcodec does. On the W → N side a
// nil source (nil pointer or interface) is stored as Valid=false without
// calling the chain; any other converted value is valid, except that with
// [WithZeroAsNull] a zero T converted from a non-pointer source is stored as
// Valid=false, as in the N ↔ T and N ↔ N2 shapes. Errors from the chain are
// returned unchanged. Register this codec before the codec that handles T ↔ W.
//
// Pairs with no null type, and identical null types, are delegated to next.
// Null types are recognized as values only, not through pointers.
//
// Example:
//
//	converter.ConvertE(row, &entity, converter.WithErrorCodecs(sqlnull.New()))
func New(opts ...Option) convcodec.CodecE {
	o := newOptions(opts...)

	return func(path string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
		return convert(path, src, dst, next, o)
	}
}

// IsNullType reports whether t is a database/sql null type: sql.Null[T] or one
// of the legacy sql.NullXxx structs, all of which hold a value field followed
// by a Valid bool.
func IsNullType(t reflect.Type) bool {
	if t.Kind() != reflect.Struct || t.PkgPath() != sqlPackage || !strings.HasPrefix(t.Name(), "Null") || t.NumField() != nullFields {
		return false
	}

	valid := t.Field(validField)

	return valid.Name == "Valid" && valid.Type.Kind() == reflect.Bool
}

// convert handles null type pairs and delegates the rest to next.
func convert(path string, src, dst reflect.Value, next convcodec.CodecHandlerE, o *options) error {
	srcType, dstType := src.Type(), dst.Type()

	srcNull, dstNull := IsNullType(srcType), IsNullType(dstType)
	switch {
	case srcNull == dstNull && (!srcNull || srcType == dstType):
		return next(path, src, dst)
	case srcNull && dstNull:
		return nullToNull(path, src, dst, next, o)
	case srcNull:
		return fromNull(path, src, dst, next)
	default:
		return toNull(path, src, dst, next, o)
	}
}

// nullToNull converts between two different null types.
func nullToNull(path string, src, dst reflect.Value, next convcodec.CodecHandlerE, o *options) error {
	if !src.Field(validField).Bool() {
		dst.SetZero()
		return nil
	}

	value := reflect.New(dst.Type().Field(valueField).Type).Elem()
	if err := next(path, src.Field(valueField), value); err != nil {
		return err
	}

	setNullable(dst, value, o)

	return nil
}

// fromNull converts a null type source into a plain destination.
func fromNull(path string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
	if !src.Field(validField).Bool() {
		dst.SetZero()
		return nil
	}

	value := src.Field(valueField)

	inner, dstType := value.Type(), dst.Type()
	switch {
	case dstType == inner:
		dst.Set(value)
	case dstType.Kind() == reflect.Pointer && dstType.Elem() == inner:
		p := reflect.New(inner)
		p.Elem().Set(value)
		dst.Set(p)
	default:
		return next(path, value, dst)
	}

	return nil
}

// toNull converts a plain source into a null type destination.
func toNull(path string, src, dst reflect.Value, next convcodec.CodecHandlerE, o *options) error {
	srcType := src.Type()

	inner := dst.Type().Field(valueField).Type
	switch {
	case srcType == inner:
		setNullable(dst, src, o)
	case srcType.Kind() == reflect.Pointer && srcType.Elem() == inner:
		if src.IsNil() {
			dst.SetZero()
			return nil
		}

		setValid(dst, src.Elem())
	case isNilSource(src):
		dst.SetZero()
	default:
		value := reflect.New(inner).Elem()
		if err := next(path, src, value); err != nil {
			return err
		}

		if srcType.Kind() == reflect.Pointer {
			setValid(dst, value)
			return nil
		}

		setNullable(dst, value, o)
	}

	return nil
}

// setNullable stores value into a null type destination: as NULL when it is
// zero and [WithZeroAsNull] is set, as a valid value otherwise.
func setNullable(dst, value reflect.Value, o *options) {
	if o.zeroAsNull && value.IsZero() {
		dst.SetZero()
		return
	}

	setValid(dst, value)
}

// setValid stores value into a null type destination and marks it valid.
func setValid(dst, value reflect.Value) {
	dst.Field(valueField).Set(value)
	dst.Field(validField).SetBool(true)
}

// isNilSource reports whether src is a nil pointer or interface, or an
// interface holding a nil pointer.
func isNilSource(src reflect.Value) bool {
	for src.Kind() == reflect.Interface {
		if src.IsNil() {
			return true
		}

		src = src.Elem()
	}

	return src.Kind() == reflect.Pointer && src.IsNil()
}
