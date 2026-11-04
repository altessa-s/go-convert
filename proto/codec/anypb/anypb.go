// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package anypb

import (
	"errors"
	"reflect"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"

	convcodec "github.com/altessa-s/go-convert/codec"
	reflectutils "github.com/altessa-s/go-convert/internal/reflect"
	pbany "google.golang.org/protobuf/types/known/anypb"
)

var (
	anyType          = reflect.TypeFor[pbany.Any]()
	protoMessageType = reflect.TypeFor[proto.Message]()
)

// codec holds the configuration shared by every call of the codec returned
// by [New].
type codec struct {
	ignoreNil bool
	registry  *Registry
	unmarshal proto.UnmarshalOptions
}

// New creates an error codec for conversion between google.protobuf.Any and
// concrete protobuf messages or Go domain types. Register it with
// converter.WithErrorCodecs.
//
// Supported conversions, all in both directions and for pointer and
// non-pointer fields:
//   - Any <-> a protobuf message type: the message is packed with
//     anypb.MarshalFrom and unpacked with anypb.UnmarshalTo.
//   - Any <-> a domain type registered with [WithRegistry]: the Any is
//     unpacked into the registered message, which the rest of the codec
//     chain (next) converts into the domain type; packing converts the domain
//     value into the message through next first. Codecs for the message and
//     domain pair, such as tspb for timestamppb.Timestamp and time.Time, must
//     be registered for the inner conversion to apply.
//   - Any <-> an interface type, such as any or proto.Message: unpacking
//     picks the registered domain type for the type URL, or else a message
//     resolved by [WithResolver] (protoregistry.GlobalTypes by default), and
//     requires it to implement the interface. Packing accepts any protobuf
//     message or registered domain value held by the interface.
//
// A nil source sets the destination to its zero value; with [WithIgnoreNil]
// the destination is left untouched.
//
// Conversion fails with a [TypeMismatchError] when the type URL does not
// match the destination, an [UnknownTypeError] for type URLs or Go types the
// codec cannot map, an [Error] when protobuf marshaling fails, and with any
// error returned by next. Other field pairs are passed to next unchanged.
//
// Example:
//
//	registry := anypb.NewRegistry().Register(&timestamppb.Timestamp{}, reflect.TypeFor[time.Time]())
//	_, err := converter.ConvertE(src, &dst,
//	    converter.WithErrorCodecs(anypb.New(anypb.WithRegistry(registry))),
//	    converter.WithCodecs(tspb.New()))
func New(opts ...Option) convcodec.CodecE {
	o := newOptions(opts...)
	c := &codec{
		ignoreNil: o.ignoreNil,
		registry:  o.registry,
		unmarshal: proto.UnmarshalOptions{Resolver: o.resolver},
	}
	return c.convert
}

// convert is the CodecE entry point.
func (c *codec) convert(path string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
	srcType := reflectutils.IndirectType(src.Type())
	dstType := reflectutils.IndirectType(dst.Type())

	switch {
	case srcType == anyType && dstType != anyType && c.supports(dstType):
		value := reflectutils.IndirectValue(src)
		if !value.IsValid() {
			c.clear(dst)
			return nil
		}
		return c.unpack(path, messagePtr(value), dst, dstType, next)

	case dstType == anyType && srcType != anyType && c.supports(srcType):
		value := sourceValue(src)
		if !value.IsValid() {
			c.clear(dst)
			return nil
		}
		if value.Type() == anyType {
			// An interface holding an Any is assigned as is.
			setDst(dst, addressable(value).Addr())
			return nil
		}
		packed, err := c.pack(path, value, next)
		if err != nil {
			return err
		}
		setDst(dst, reflect.ValueOf(packed))
		return nil

	default:
		return next(path, src, dst)
	}
}

// supports reports whether values of type t convert to and from an Any.
func (c *codec) supports(t reflect.Type) bool {
	if isMessage(t) || t.Kind() == reflect.Interface {
		return true
	}
	_, ok := c.registry.byDomainType(t)
	return ok
}

// clear applies the nil-source rule to dst.
func (c *codec) clear(dst reflect.Value) {
	if !c.ignoreNil && dst.CanSet() {
		dst.SetZero()
	}
}

// unpack converts the Any src into dst, whose innermost type is dstType.
func (c *codec) unpack(path string, src *pbany.Any, dst reflect.Value, dstType reflect.Type, next convcodec.CodecHandlerE) error {
	if isMessage(dstType) {
		target := reflect.New(dstType)
		message, _ := reflect.TypeAssert[proto.Message](target)
		if !src.MessageIs(message) {
			return TypeMismatchError{Path: path, TypeURL: src.GetTypeUrl(), Want: string(message.ProtoReflect().Descriptor().FullName())}
		}
		if err := c.unmarshal.Unmarshal(src.GetValue(), message); err != nil {
			return Error{Path: path, Err: err}
		}
		setDst(dst, target)
		return nil
	}

	if entry, ok := c.registry.byDomainType(dstType); ok {
		if src.MessageName() != entry.message.Descriptor().FullName() {
			return TypeMismatchError{Path: path, TypeURL: src.GetTypeUrl(), Want: string(entry.message.Descriptor().FullName())}
		}
		target, err := c.toDomain(path, src, entry, next)
		if err != nil {
			return err
		}
		setDst(dst, target)
		return nil
	}

	// An interface destination: the type URL decides the concrete type.
	target, err := c.resolve(path, src, next)
	if err != nil {
		return err
	}
	if !target.Type().Implements(dstType) && !target.Elem().Type().Implements(dstType) {
		return TypeMismatchError{Path: path, TypeURL: src.GetTypeUrl(), Want: dstType.String()}
	}
	setDst(dst, target)
	return nil
}

// resolve unpacks src into the registered domain type for its type URL, or
// else into a new message from the resolver. It returns a pointer.
func (c *codec) resolve(path string, src *pbany.Any, next convcodec.CodecHandlerE) (reflect.Value, error) {
	name := src.MessageName()
	if name == "" {
		return reflect.Value{}, UnknownTypeError{Path: path, TypeURL: src.GetTypeUrl()}
	}
	if entry, ok := c.registry.byMessageName(name); ok {
		return c.toDomain(path, src, entry, next)
	}
	message, err := pbany.UnmarshalNew(src, c.unmarshal)
	if errors.Is(err, protoregistry.NotFound) {
		return reflect.Value{}, UnknownTypeError{Path: path, TypeURL: src.GetTypeUrl()}
	}
	if err != nil {
		return reflect.Value{}, Error{Path: path, Err: err}
	}
	return reflect.ValueOf(message), nil
}

// toDomain unpacks src into the registered message, then converts the
// message into a new domain value through next. It returns a pointer to the
// domain value.
func (c *codec) toDomain(path string, src *pbany.Any, entry registration, next convcodec.CodecHandlerE) (reflect.Value, error) {
	message := entry.message.New().Interface()
	if err := c.unmarshal.Unmarshal(src.GetValue(), message); err != nil {
		return reflect.Value{}, Error{Path: path, Err: err}
	}
	target := reflect.New(entry.domain)
	if err := next(path, reflect.ValueOf(message), target.Elem()); err != nil {
		return reflect.Value{}, err
	}
	return target, nil
}

// pack converts a non-nil source value into a new Any.
func (c *codec) pack(path string, src reflect.Value, next convcodec.CodecHandlerE) (*pbany.Any, error) {
	var message proto.Message
	switch entry, registered := c.registry.byDomainType(src.Type()); {
	case isMessage(src.Type()):
		message = messageOf(src)
	case registered:
		message = entry.message.New().Interface()
		target := reflect.ValueOf(message).Elem()
		if err := next(path, src, target); err != nil {
			return nil, err
		}
	default:
		return nil, UnknownTypeError{Path: path, Type: src.Type()}
	}

	packed := &pbany.Any{}
	if err := pbany.MarshalFrom(packed, message, proto.MarshalOptions{}); err != nil {
		return nil, Error{Path: path, Err: err}
	}
	return packed, nil
}

// isMessage reports whether t is a generated protobuf message struct.
func isMessage(t reflect.Type) bool {
	return t.Kind() == reflect.Struct && reflect.PointerTo(t).Implements(protoMessageType)
}

// sourceValue dereferences pointers and unwraps interfaces, returning an
// invalid value when the source is nil.
func sourceValue(src reflect.Value) reflect.Value {
	value := reflectutils.IndirectValue(src)
	for value.IsValid() && value.Kind() == reflect.Interface {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = reflectutils.IndirectValue(value.Elem())
	}
	return value
}

// addressable returns v, or an addressable copy of it.
func addressable(v reflect.Value) reflect.Value {
	if v.CanAddr() {
		return v
	}
	tmp := reflect.New(v.Type()).Elem()
	tmp.Set(v)
	return tmp
}

// messagePtr returns a pointer to the Any held by v.
func messagePtr(v reflect.Value) *pbany.Any {
	p, _ := reflect.TypeAssert[*pbany.Any](addressable(v).Addr())
	return p
}

// messageOf returns the protobuf message held by the struct value v.
func messageOf(v reflect.Value) proto.Message {
	m, _ := reflect.TypeAssert[proto.Message](addressable(v).Addr())
	return m
}

// setDst stores the pointer out into dst, allocating pointer layers. An
// interface destination receives the pointed-to value when it implements the
// interface and is not a protobuf message, and the pointer otherwise.
func setDst(dst, out reflect.Value) {
	for dst.Kind() == reflect.Pointer {
		if dst.Type() == out.Type() {
			dst.Set(out)
			return
		}
		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}
		dst = dst.Elem()
	}
	elem := out.Elem()
	if dst.Kind() == reflect.Interface && (isMessage(elem.Type()) || !elem.Type().Implements(dst.Type())) {
		dst.Set(out)
		return
	}
	dst.Set(elem)
}
