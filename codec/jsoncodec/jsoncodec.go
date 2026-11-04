// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package jsoncodec

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"reflect"

	"github.com/altessa-s/go-convert/reflectx"

	convcodec "github.com/altessa-s/go-convert/codec"
)

var (
	// valueType is jsontext.Value; encoding/json.RawMessage is an alias of it.
	valueType  = reflect.TypeFor[jsontext.Value]()
	bytesType  = reflect.TypeFor[[]byte]()
	stringType = reflect.TypeFor[string]()
)

// Op names the JSON operation that failed.
type Op string

const (
	// OpMarshal is a failure encoding a Go value as JSON text.
	OpMarshal Op = "marshal"
	// OpUnmarshal is a failure decoding JSON text into a Go value.
	OpUnmarshal Op = "unmarshal"
)

// Error reports a JSON marshal or unmarshal failure for a field. It wraps
// the encoding/json/v2 error, such as a *json.SemanticError or a
// *jsontext.SyntacticError, and is returned unchanged by convert.ConvertE.
type Error struct {
	// Path is the full source field path.
	Path string
	// Op is the failed operation.
	Op Op
	// Err is the underlying encoding/json/v2 error.
	Err error
}

// Error reports the field path, the operation and the cause.
func (e Error) Error() string {
	return fmt.Sprintf("jsoncodec: field %q: %s: %v", e.Path, e.Op, e.Err)
}

// Unwrap returns the underlying encoding/json/v2 error.
func (e Error) Unwrap() error { return e.Err }

// SizeError reports JSON text longer than the limit set with [WithMaxBytes].
// It is the Err of an [Error] with [OpUnmarshal].
type SizeError struct {
	// Size is the length of the rejected text in bytes.
	Size int
	// Limit is the configured maximum in bytes.
	Limit int
}

// Error reports the size of the text and the limit.
func (e SizeError) Error() string {
	return fmt.Sprintf("JSON text of %d bytes exceeds the limit of %d", e.Size, e.Limit)
}

// New returns an error codec that converts between JSON text and Go values
// with encoding/json/v2.
//
// JSON text is a jsontext.Value or json.RawMessage, and also a plain []byte
// with [WithBytes] or a plain string with [WithStrings]. Go values are
// structs, maps, slices (other than byte slices) and empty interfaces. Both
// sides may be pointers. The codec claims a field pair only when one side is
// JSON text and the other a Go value; every other pair is delegated to next.
//
//   - text → value: unmarshals into a fresh value and assigns it, so the
//     destination is replaced, not merged, and left unchanged on error.
//     Empty text and a nil source zero the destination.
//   - value → text: marshals the value. A nil pointer or nil interface
//     source zeroes the destination; nil maps and slices follow the JSON
//     options (encoding/json/v2 writes {} and [] by default).
//
// Failures are returned as an [Error] wrapping the encoding/json/v2 error.
// Options passed with [WithJSONOptions] apply to both directions. With
// [WithMaxBytes], text longer than the limit is rejected before decoding
// with an [Error] wrapping [SizeError].
//
// Example:
//
//	codec := jsoncodec.New(jsoncodec.WithJSONOptions(json.RejectUnknownMembers(true)))
//	_, err := convert.ConvertE(src, &dst, convert.WithErrorCodecs(codec))
func New(opts ...Option) convcodec.CodecE {
	c := &codec{options: *newOptions(opts...)}
	return c.convert
}

type codec struct {
	options
}

// convert handles JSON text and value pairs and delegates the rest to next.
func (c *codec) convert(path string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
	srcType := reflectx.IndirectType(src.Type())

	dstType := reflectx.IndirectType(dst.Type())
	switch {
	case c.isText(srcType) && isValue(dstType):
		return c.unmarshal(path, src, dst, dstType)
	case isValue(srcType) && c.isText(dstType):
		return c.marshal(path, src, dst, dstType)
	default:
		return next(path, src, dst)
	}
}

// isText reports whether t holds JSON text.
func (c *codec) isText(t reflect.Type) bool {
	return t == valueType || c.byteText && t == bytesType || c.stringText && t == stringType
}

// isValue reports whether t is a Go value the codec encodes: a struct, map,
// non-byte slice or empty interface.
func isValue(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Struct, reflect.Map:
		return true
	case reflect.Slice:
		return t.Elem().Kind() != reflect.Uint8
	case reflect.Interface:
		return t.NumMethod() == 0
	default:
		return false
	}
}

// unmarshal decodes the JSON text src into a fresh value and assigns it to dst.
func (c *codec) unmarshal(path string, src, dst reflect.Value, dstType reflect.Type) error {
	src = reflectx.IndirectValue(src)

	var text []byte

	switch {
	case !src.IsValid():
	case src.Kind() == reflect.String:
		text = []byte(src.String())
	default:
		text = src.Bytes()
	}

	if len(text) == 0 {
		dst.SetZero()
		return nil
	}

	if c.maxBytes > 0 && len(text) > c.maxBytes {
		return Error{Path: path, Op: OpUnmarshal, Err: SizeError{Size: len(text), Limit: c.maxBytes}}
	}

	value := reflect.New(dstType)
	if err := json.Unmarshal(text, value.Interface(), c.jsonOptions...); err != nil {
		return Error{Path: path, Op: OpUnmarshal, Err: err}
	}

	reflectx.MakeDst(&dst, dstType)
	dst.Set(value.Elem())

	return nil
}

// marshal encodes src as JSON text into dst.
func (c *codec) marshal(path string, src, dst reflect.Value, dstType reflect.Type) error {
	src = reflectx.IndirectValue(src)
	if !src.IsValid() || src.Kind() == reflect.Interface && src.IsNil() {
		dst.SetZero()
		return nil
	}

	// Marshal through a pointer when possible so that pointer-receiver
	// MarshalJSON methods are honored.
	in := src
	if src.CanAddr() {
		in = src.Addr()
	}

	text, err := json.Marshal(in.Interface(), c.jsonOptions...)
	if err != nil {
		return Error{Path: path, Op: OpMarshal, Err: err}
	}

	reflectx.MakeDst(&dst, dstType)

	if dst.Kind() == reflect.String {
		dst.SetString(string(text))
	} else {
		dst.SetBytes(text)
	}

	return nil
}
