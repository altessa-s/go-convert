// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package textcodec

import (
	"encoding"
	"fmt"
	"reflect"

	convcodec "github.com/altessa-s/go-convert/codec"
	reflectutils "github.com/altessa-s/go-convert/internal/reflect"
)

var (
	marshalerType   = reflect.TypeFor[encoding.TextMarshaler]()
	unmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()
)

// Error reports a failed MarshalText or UnmarshalText call.
type Error struct {
	// Path is the full source field path; it is empty at the root.
	Path string
	// Type is the type whose MarshalText or UnmarshalText failed.
	Type reflect.Type
	// Err is the error returned by MarshalText or UnmarshalText.
	Err error
}

func (e *Error) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("textcodec: %s: %v", e.Type, e.Err)
	}
	return fmt.Sprintf("textcodec: field %q: %s: %v", e.Path, e.Type, e.Err)
}

// Unwrap returns the error from MarshalText or UnmarshalText.
func (e *Error) Unwrap() error { return e.Err }

// New creates an error codec converting through encoding.TextMarshaler and
// encoding.TextUnmarshaler.
//
// Supported conversions (pointer variants on either side included):
//   - TextMarshaler -> string or []byte kind: calls MarshalText
//   - string or []byte kind -> TextUnmarshaler: calls UnmarshalText
//   - TextMarshaler -> TextUnmarshaler of another type: MarshalText, then UnmarshalText
//
// A method on either the value or the pointer receiver qualifies. Identical
// source and destination types, and interface types, are passed to next.
// Failures are returned as an [*Error]. Empty text sets the destination to its
// zero value unless [WithParseEmpty] is set; a nil source pointer leaves the
// destination unchanged.
//
// Example:
//
//	codec := textcodec.New()
//	codec := textcodec.New(textcodec.WithParseEmpty())
func New(opts ...Option) convcodec.CodecE {
	o := newOptions(opts...)

	return func(fieldPath string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
		if handled, err := tryConvert(fieldPath, src, dst, o); handled {
			return err
		}

		return next(fieldPath, src, dst)
	}
}

// tryConvert converts src into dst when one side is a text type. It reports
// whether the conversion was handled (including a nil source).
func tryConvert(fieldPath string, src, dst reflect.Value, opts *options) (bool, error) {
	srcType := reflectutils.IndirectType(src.Type())
	dstType := reflectutils.IndirectType(dst.Type())
	if srcType == dstType || srcType.Kind() == reflect.Interface || dstType.Kind() == reflect.Interface {
		return false, nil
	}

	dstUnmarshaler := reflect.PointerTo(dstType).Implements(unmarshalerType)
	srcMarshaler := isMarshaler(srcType)

	switch {
	case srcMarshaler && (dstUnmarshaler || isText(dstType)):
	case dstUnmarshaler && isText(srcType):
	default:
		return false, nil
	}

	srcValue := reflectutils.IndirectValue(src)
	if !srcValue.IsValid() {
		return true, nil
	}
	if !srcValue.CanInterface() {
		return false, nil
	}

	var text []byte
	if srcMarshaler {
		var err error
		if text, err = marshal(srcValue); err != nil {
			return true, &Error{Path: fieldPath, Type: srcType, Err: err}
		}
	} else {
		text = readText(srcValue)
	}

	if len(text) == 0 && (!dstUnmarshaler || !opts.parseEmpty) {
		dst.SetZero()
		return true, nil
	}

	if !dstUnmarshaler {
		reflectutils.MakeDst(&dst, dstType)
		writeText(dst, text)
		return true, nil
	}

	value, err := unmarshal(dstType, text)
	if err != nil {
		return true, &Error{Path: fieldPath, Type: dstType, Err: err}
	}
	reflectutils.MakeDst(&dst, dstType)
	dst.Set(value)

	return true, nil
}

// isMarshaler reports whether t or *t implements encoding.TextMarshaler.
func isMarshaler(t reflect.Type) bool {
	return t.Implements(marshalerType) || reflect.PointerTo(t).Implements(marshalerType)
}

// isText reports whether t is a string or a byte slice kind.
func isText(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.String:
		return true
	case reflect.Slice:
		return t.Elem().Kind() == reflect.Uint8
	default:
		return false
	}
}

// marshal calls MarshalText on v, addressing a copy of v when only the pointer
// type implements encoding.TextMarshaler and v is not addressable.
func marshal(v reflect.Value) ([]byte, error) {
	if !v.Type().Implements(marshalerType) {
		if v.CanAddr() {
			v = v.Addr()
		} else {
			ptr := reflect.New(v.Type())
			ptr.Elem().Set(v)
			v = ptr
		}
	}

	m, ok := reflect.TypeAssert[encoding.TextMarshaler](v)
	if !ok {
		return nil, fmt.Errorf("%s does not implement encoding.TextMarshaler", v.Type())
	}

	return m.MarshalText()
}

// unmarshal returns a new value of type t holding text decoded by its
// UnmarshalText method. A fresh value keeps the destination unchanged when
// decoding fails.
func unmarshal(t reflect.Type, text []byte) (reflect.Value, error) {
	ptr := reflect.New(t)
	u, ok := reflect.TypeAssert[encoding.TextUnmarshaler](ptr)
	if !ok {
		return reflect.Value{}, fmt.Errorf("%s does not implement encoding.TextUnmarshaler", ptr.Type())
	}
	if err := u.UnmarshalText(text); err != nil {
		return reflect.Value{}, err
	}

	return ptr.Elem(), nil
}

// readText returns the text held by a string or byte slice value.
func readText(v reflect.Value) []byte {
	if v.Kind() == reflect.String {
		return []byte(v.String())
	}

	return v.Bytes()
}

// writeText stores text into a string or byte slice value.
func writeText(dst reflect.Value, text []byte) {
	if dst.Kind() == reflect.String {
		dst.SetString(string(text))
		return
	}

	dst.SetBytes(text)
}
