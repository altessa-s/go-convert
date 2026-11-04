// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package enumpb

import (
	"fmt"
	"reflect"

	"google.golang.org/protobuf/reflect/protoreflect"

	convcodec "github.com/altessa-s/go-convert/codec"
	reflectutils "github.com/altessa-s/go-convert/internal/reflect"
)

// Enum is the constraint satisfied by generated protobuf enum types.
type Enum interface {
	~int32
	protoreflect.Enum
}

// mappingCodec converts between one protobuf enum type P and one Go type G
// through a bijective table.
type mappingCodec[P Enum, G comparable] struct {
	opts     *options
	enumType reflect.Type
	goType   reflect.Type
	enumName protoreflect.FullName
	toGo     map[P]reflect.Value
	toEnum   map[G]P
	zeroGo   G
}

// NewMapping creates an error codec for conversion between the protobuf enum
// type P and a Go type G, typically a hand-written Go enum, through table.
// Pointers to either side are supported.
//
// The table must be one-to-one; NewMapping panics when two enum values map to
// the same Go value. A value missing from the table fails with
// [UnknownValueError]. With [WithUnspecifiedAsEmpty], the zero enum number
// and the zero (or nil) Go value map to each other unless the table lists
// them; the name options do not apply, but an invalid [Case] still panics. A
// nil source pointer sets the destination to its zero value. Register the
// codec with converter.WithErrorCodecs.
//
// Example:
//
//	codec := enumpb.NewMapping(map[userv1.Status]user.Status{
//		userv1.Status_STATUS_ACTIVE:  user.Active,
//		userv1.Status_STATUS_BLOCKED: user.Blocked,
//	}, enumpb.WithUnspecifiedAsEmpty())
func NewMapping[P Enum, G comparable](table map[P]G, opts ...Option) convcodec.CodecE {
	var zeroEnum P

	c := &mappingCodec[P, G]{
		opts:     checkedOptions("NewMapping", opts),
		enumType: reflect.TypeFor[P](),
		goType:   reflect.TypeFor[G](),
		enumName: zeroEnum.Descriptor().FullName(),
		toGo:     make(map[P]reflect.Value, len(table)),
		toEnum:   make(map[G]P, len(table)),
	}
	for enum, value := range table {
		if previous, ok := c.toEnum[value]; ok {
			panic(fmt.Sprintf("enumpb: NewMapping: %v and %v both map to %v", previous, enum, value))
		}

		c.toEnum[value] = enum
		c.toGo[enum] = reflect.ValueOf(&value).Elem()
	}

	return c.convert
}

// convert handles pairs of P and G and delegates the rest to next.
func (c *mappingCodec[P, G]) convert(path string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
	srcType := reflectutils.IndirectType(src.Type())
	dstType := reflectutils.IndirectType(dst.Type())

	switch {
	case srcType == c.enumType && dstType == c.goType:
		return c.toGoValue(path, src, dst)
	case srcType == c.goType && dstType == c.enumType:
		return c.toEnumValue(path, src, dst)
	default:
		return next(path, src, dst)
	}
}

// toGoValue writes the Go value mapped from the source enum into dst.
func (c *mappingCodec[P, G]) toGoValue(path string, src, dst reflect.Value) error {
	value := reflectutils.IndirectValue(src)
	if !value.IsValid() {
		zeroDst(dst)
		return nil
	}

	enum := P(value.Int())

	mapped, ok := c.toGo[enum]
	if !ok {
		if enum == 0 && c.opts.unspecifiedAsEmpty {
			dst.Set(reflect.Zero(dst.Type()))
			return nil
		}

		return UnknownValueError{Enum: c.enumName, Path: path, Value: protoreflect.EnumNumber(enum)}
	}

	reflectutils.MakeDst(&dst, c.goType)
	dst.Set(mapped)

	return nil
}

// toEnumValue writes the enum mapped from the source Go value into dst.
func (c *mappingCodec[P, G]) toEnumValue(path string, src, dst reflect.Value) error {
	value := reflectutils.IndirectValue(src)
	if !value.IsValid() {
		zeroDst(dst)
		return nil
	}

	// The source type is G, so the assertion cannot fail.
	goValue, _ := reflect.TypeAssert[G](value)

	enum, ok := c.toEnum[goValue]
	if !ok {
		if goValue == c.zeroGo && c.opts.unspecifiedAsEmpty {
			dst.Set(reflect.Zero(dst.Type()))
			return nil
		}

		return UnknownValueError{Enum: c.enumName, Path: path, Value: goValue}
	}

	reflectutils.MakeDst(&dst, c.enumType)
	dst.SetInt(int64(enum))

	return nil
}
