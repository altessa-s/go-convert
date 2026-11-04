// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package enumpb

import (
	"fmt"
	"reflect"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/altessa-s/go-convert/reflectx"

	convcodec "github.com/altessa-s/go-convert/codec"
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
	goValues map[P]G
	zeroGo   G
	// zeroPair maps the zero enum number and the zero Go value to each other:
	// [WithUnspecifiedAsEmpty] is set and the table lists neither of them, so
	// the pair cannot break the one-to-one table.
	zeroPair bool
}

// NewMapping creates an error codec for conversion between the protobuf enum
// type P and a Go type G, typically a hand-written Go enum, through table.
// Pointers to either side are supported.
//
// The table must be one-to-one; NewMapping returns a [ConfigError] when two
// enum values map to the same Go value. A value missing from the table fails
// with [UnknownValueError]. With [WithUnspecifiedAsEmpty], the zero enum
// number and the zero (or nil) Go value map to each other when the table
// lists neither of them, so the pair never breaks the one-to-one table; the
// name options do not apply, but an invalid [Case] is still a
// [ConfigError]. A nil source pointer sets the destination to its zero value.
// Register the codec with convert.WithErrorCodecs.
//
// Example:
//
//	codec, err := enumpb.NewMapping(map[userv1.Status]user.Status{
//		userv1.Status_STATUS_ACTIVE:  user.Active,
//		userv1.Status_STATUS_BLOCKED: user.Blocked,
//	}, enumpb.WithUnspecifiedAsEmpty())
func NewMapping[P Enum, G comparable](table map[P]G, opts ...Option) (convcodec.CodecE, error) {
	c, err := newMappingCodec("NewMapping", table, opts)
	if err != nil {
		return nil, err
	}

	return c.convert, nil
}

// MustNewMapping is [NewMapping] for tables known to be valid, such as
// package-level literals; it panics with the [ConfigError] instead of
// returning it.
func MustNewMapping[P Enum, G comparable](table map[P]G, opts ...Option) convcodec.CodecE {
	return must(NewMapping(table, opts...))
}

// newMappingCodec builds the codec of table for the constructor named caller,
// which names it in a [ConfigError].
func newMappingCodec[P Enum, G comparable](caller string, table map[P]G, opts []Option) (*mappingCodec[P, G], error) {
	var zeroEnum P

	o, err := checkedOptions(caller, opts)
	if err != nil {
		return nil, err
	}

	c := &mappingCodec[P, G]{
		opts:     o,
		enumType: reflect.TypeFor[P](),
		goType:   reflect.TypeFor[G](),
		enumName: zeroEnum.Descriptor().FullName(),
		toGo:     make(map[P]reflect.Value, len(table)),
		toEnum:   make(map[G]P, len(table)),
		goValues: make(map[P]G, len(table)),
	}
	for enum, value := range table {
		if previous, ok := c.toEnum[value]; ok {
			return nil, ConfigError{Func: caller, Reason: fmt.Sprintf("%v and %v both map to %v", previous, enum, value)}
		}

		c.toEnum[value] = enum
		c.toGo[enum] = reflect.ValueOf(&value).Elem()
		c.goValues[enum] = value
	}

	_, zeroEnumListed := c.toGo[0]
	_, zeroGoListed := c.toEnum[c.zeroGo]
	c.zeroPair = o.unspecifiedAsEmpty && !zeroEnumListed && !zeroGoListed

	return c, nil
}

// convert handles pairs of P and G and delegates the rest to next.
func (c *mappingCodec[P, G]) convert(path string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
	srcType := reflectx.IndirectType(src.Type())
	dstType := reflectx.IndirectType(dst.Type())

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
	value := reflectx.IndirectValue(src)
	if !value.IsValid() {
		zeroDst(dst)
		return nil
	}

	enum := P(value.Int())

	mapped, ok := c.toGo[enum]
	if !ok {
		if enum == 0 && c.zeroPair {
			dst.Set(reflect.Zero(dst.Type()))
			return nil
		}

		return UnknownValueError{Enum: c.enumName, Path: path, Value: protoreflect.EnumNumber(enum)}
	}

	reflectx.MakeDst(&dst, c.goType)
	dst.Set(mapped)

	return nil
}

// toEnumValue writes the enum mapped from the source Go value into dst.
func (c *mappingCodec[P, G]) toEnumValue(path string, src, dst reflect.Value) error {
	value := reflectx.IndirectValue(src)
	if !value.IsValid() {
		zeroDst(dst)
		return nil
	}

	// The source type is G, so the assertion cannot fail.
	goValue, _ := reflect.TypeAssert[G](value)

	enum, ok := c.toEnum[goValue]
	if !ok {
		if goValue == c.zeroGo && c.zeroPair {
			dst.Set(reflect.Zero(dst.Type()))
			return nil
		}

		return UnknownValueError{Enum: c.enumName, Path: path, Value: goValue}
	}

	reflectx.MakeDst(&dst, c.enumType)
	dst.SetInt(int64(enum))

	return nil
}
