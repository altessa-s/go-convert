// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package enumpb

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/altessa-s/go-convert/reflectx"

	"google.golang.org/protobuf/reflect/protoreflect"

	convcodec "github.com/altessa-s/go-convert/codec"
)

var enumInterface = reflect.TypeFor[protoreflect.Enum]()

// UnknownValueError reports an enum number, name or mapped Go value that has
// no counterpart on the other side of the conversion.
type UnknownValueError struct {
	// Enum is the full name of the protobuf enum, such as
	// "google.protobuf.FieldDescriptorProto.Type".
	Enum protoreflect.FullName
	// Path is the dotted source field path.
	Path string
	// Value is the unknown [protoreflect.EnumNumber], name or Go value.
	Value any
}

// Error reports the field path, the enum and the unknown value.
func (e UnknownValueError) Error() string {
	value := fmt.Sprint(e.Value)
	if s, ok := e.Value.(string); ok {
		value = fmt.Sprintf("%q", s)
	}

	return fmt.Sprintf("enumpb: field %q: %s has no value for %s", e.Path, e.Enum, value)
}

// ConfigError reports options or a table that cannot build a codec, such as
// an invalid [Case] or two enum values mapped to the same Go value.
type ConfigError struct {
	// Func is the constructor that failed, such as "NewMapping".
	Func string
	// Reason describes the problem.
	Reason string
}

// Error reports the constructor and the problem.
func (e ConfigError) Error() string {
	return fmt.Sprintf("enumpb: %s: %s", e.Func, e.Reason)
}

// enumNames is the name table of one enum type under one option set.
type enumNames struct {
	name    protoreflect.FullName
	names   map[protoreflect.EnumNumber]string
	numbers map[string]protoreflect.EnumNumber
}

// stringCodec converts protobuf enums to and from strings, caching the name
// table of every enum type it meets.
type stringCodec struct {
	opts  *options
	cache sync.Map // reflect.Type -> *enumNames, nil for non-enum int32 types
}

// New creates an error codec for conversion between protobuf enums and
// strings (any string kind, and pointers to either side).
//
// An enum is written as its value name, transformed by [WithTrimPrefix],
// [WithTrimEnumPrefix] and [WithCase]; a string is parsed with the same
// transformation. A number or name without a value in the enum descriptor
// fails with [UnknownValueError], which convert.ConvertE returns unchanged.
// A nil source pointer sets the destination to its zero value. New returns a
// [ConfigError] when [WithCase] is given a value other than [CaseKeep],
// [CaseLower] and [CaseUpper]. Register the codec with convert.WithErrorCodecs.
//
// Example:
//
//	codec := enumpb.MustNew(enumpb.WithTrimEnumPrefix(), enumpb.WithCase(enumpb.CaseLower))
//	_, err := convert.ConvertE(msg, &dto, convert.WithErrorCodecs(codec))
func New(opts ...Option) (convcodec.CodecE, error) {
	o, err := checkedOptions("New", opts)
	if err != nil {
		return nil, err
	}

	c := &stringCodec{opts: o}

	return c.convert, nil
}

// MustNew is [New] for options known to be valid; it panics with the
// [ConfigError] instead of returning it.
func MustNew(opts ...Option) convcodec.CodecE {
	return must(New(opts...))
}

// must returns codec, panicking with err when it is not nil.
func must(codec convcodec.CodecE, err error) convcodec.CodecE {
	if err != nil {
		panic(err)
	}

	return codec
}

// convert handles enum and string pairs and delegates the rest to next.
func (c *stringCodec) convert(path string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
	srcType := reflectx.IndirectType(src.Type())
	dstType := reflectx.IndirectType(dst.Type())

	switch {
	case dstType.Kind() == reflect.String:
		if table := c.table(srcType); table != nil {
			return c.toString(path, table, src, dst, dstType)
		}
	case srcType.Kind() == reflect.String:
		if table := c.table(dstType); table != nil {
			return c.fromString(path, table, src, dst, dstType)
		}
	default:
	}

	return next(path, src, dst)
}

// toString writes the name of the source enum value into dst.
func (c *stringCodec) toString(path string, table *enumNames, src, dst reflect.Value, dstType reflect.Type) error {
	value := reflectx.IndirectValue(src)
	if !value.IsValid() {
		zeroDst(dst)
		return nil
	}

	number := protoreflect.EnumNumber(value.Int())
	if number == 0 && c.opts.unspecifiedAsEmpty {
		dst.Set(reflect.Zero(dst.Type()))
		return nil
	}

	name, ok := table.names[number]
	if !ok {
		return UnknownValueError{Enum: table.name, Path: path, Value: number}
	}

	reflectx.MakeDst(&dst, dstType)
	dst.SetString(name)

	return nil
}

// fromString parses the source string into the destination enum.
func (c *stringCodec) fromString(path string, table *enumNames, src, dst reflect.Value, dstType reflect.Type) error {
	value := reflectx.IndirectValue(src)
	if !value.IsValid() {
		zeroDst(dst)
		return nil
	}

	name := value.String()
	if name == "" && c.opts.unspecifiedAsEmpty {
		dst.Set(reflect.Zero(dst.Type()))
		return nil
	}

	number, ok := table.numbers[foldCase(name, c.opts.nameCase)]
	if !ok {
		return UnknownValueError{Enum: table.name, Path: path, Value: name}
	}

	reflectx.MakeDst(&dst, dstType)
	dst.SetInt(int64(number))

	return nil
}

// checkedOptions applies opts, failing on an invalid [Case] in the name of
// the constructor fn.
func checkedOptions(fn string, opts []Option) (*options, error) {
	o := newOptions(opts...)
	if o.nameCase < CaseKeep || o.nameCase > CaseUpper {
		return nil, ConfigError{Func: fn, Reason: fmt.Sprintf("invalid Case %d; use CaseKeep, CaseLower or CaseUpper", int(o.nameCase))}
	}

	return o, nil
}

// zeroDst applies the nil-source rule: dst is set to its zero value, such as
// a nil pointer, "" or the zero enum.
func zeroDst(dst reflect.Value) {
	if dst.CanSet() {
		dst.SetZero()
	}
}

// table returns the name table for t, or nil when t is not a protobuf enum.
func (c *stringCodec) table(t reflect.Type) *enumNames {
	if t.Kind() != reflect.Int32 {
		return nil
	}

	if cached, ok := c.cache.Load(t); ok {
		table, _ := cached.(*enumNames)
		return table
	}

	var table *enumNames
	if desc := enumDescriptor(t); desc != nil {
		table = newEnumNames(desc, c.opts)
	}

	c.cache.Store(t, table)

	return table
}

// enumDescriptor returns the descriptor of a generated enum type, or nil
// when t does not implement [protoreflect.Enum].
func enumDescriptor(t reflect.Type) protoreflect.EnumDescriptor {
	if !t.Implements(enumInterface) {
		return nil
	}

	enum, ok := reflect.TypeAssert[protoreflect.Enum](reflect.Zero(t))
	if !ok {
		return nil
	}

	return enum.Descriptor()
}

// newEnumNames builds the name table of desc. Aliases parse to their number;
// the first declared alias is written.
func newEnumNames(desc protoreflect.EnumDescriptor, opts *options) *enumNames {
	prefix := opts.trimPrefix
	if opts.trimEnumPrefix {
		prefix = enumPrefix(string(desc.Name()))
	}

	values := desc.Values()

	table := &enumNames{
		name:    desc.FullName(),
		names:   make(map[protoreflect.EnumNumber]string, values.Len()),
		numbers: make(map[string]protoreflect.EnumNumber, values.Len()),
	}
	for i := range values.Len() {
		value := values.Get(i)

		name := valueName(string(value.Name()), prefix, opts.nameCase)
		if _, ok := table.names[value.Number()]; !ok {
			table.names[value.Number()] = name
		}

		if _, ok := table.numbers[name]; !ok {
			table.numbers[name] = value.Number()
		}
	}

	return table
}

// valueName applies the prefix and case options to a declared value name.
func valueName(name, prefix string, nameCase Case) string {
	if prefix != "" && len(name) > len(prefix) && strings.HasPrefix(name, prefix) {
		name = name[len(prefix):]
	}

	return foldCase(name, nameCase)
}

// foldCase converts name to the configured case.
func foldCase(name string, nameCase Case) string {
	switch nameCase {
	case CaseLower:
		return strings.ToLower(name)
	case CaseUpper:
		return strings.ToUpper(name)
	default:
		return name
	}
}

// enumPrefix converts an enum name in CamelCase to its value prefix in
// UPPER_SNAKE_CASE: FieldPresence -> FIELD_PRESENCE_, HTTPMethod ->
// HTTP_METHOD_.
func enumPrefix(name string) string {
	var b strings.Builder
	b.Grow(len(name) + len(name)/2 + 1)

	for i := range len(name) {
		c := name[i]
		if i > 0 && isUpper(c) {
			prev := name[i-1]

			nextLower := i+1 < len(name) && isLower(name[i+1])
			if isLower(prev) || isDigit(prev) || (isUpper(prev) && nextLower) {
				b.WriteByte('_')
			}
		}

		if isLower(c) {
			c -= 'a' - 'A'
		}

		b.WriteByte(c)
	}

	b.WriteByte('_')

	return b.String()
}

// isUpper reports whether c is an ASCII upper-case letter.
func isUpper(c byte) bool { return c >= 'A' && c <= 'Z' }

// isLower reports whether c is an ASCII lower-case letter.
func isLower(c byte) bool { return c >= 'a' && c <= 'z' }

// isDigit reports whether c is an ASCII digit.
func isDigit(c byte) bool { return c >= '0' && c <= '9' }
