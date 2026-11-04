// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package enumpb

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"google.golang.org/protobuf/reflect/protoreflect"

	convcodec "github.com/altessa-s/go-convert/codec"
	reflectutils "github.com/altessa-s/go-convert/internal/reflect"
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

func (e UnknownValueError) Error() string {
	value := fmt.Sprint(e.Value)
	if s, ok := e.Value.(string); ok {
		value = fmt.Sprintf("%q", s)
	}
	return fmt.Sprintf("enumpb: field %q: %s has no value for %s", e.Path, e.Enum, value)
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
// fails with [UnknownValueError], which converter.ConvertE returns unchanged.
// A nil source pointer leaves the destination unchanged. Register the codec
// with converter.WithErrorCodecs.
//
// Example:
//
//	codec := enumpb.New(enumpb.WithTrimEnumPrefix(), enumpb.WithCase(enumpb.CaseLower))
//	err := converter.ConvertE(msg, &dto, converter.WithErrorCodecs(codec))
func New(opts ...Option) convcodec.CodecE {
	c := &stringCodec{opts: newOptions(opts...)}
	return c.convert
}

func (c *stringCodec) convert(path string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
	srcType := reflectutils.IndirectType(src.Type())
	dstType := reflectutils.IndirectType(dst.Type())

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
	value := reflectutils.IndirectValue(src)
	if !value.IsValid() {
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

	reflectutils.MakeDst(&dst, dstType)
	dst.SetString(name)
	return nil
}

// fromString parses the source string into the destination enum.
func (c *stringCodec) fromString(path string, table *enumNames, src, dst reflect.Value, dstType reflect.Type) error {
	value := reflectutils.IndirectValue(src)
	if !value.IsValid() {
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

	reflectutils.MakeDst(&dst, dstType)
	dst.SetInt(int64(number))
	return nil
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

func isUpper(c byte) bool { return c >= 'A' && c <= 'Z' }
func isLower(c byte) bool { return c >= 'a' && c <= 'z' }
func isDigit(c byte) bool { return c >= '0' && c <= '9' }
