// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package b64codec

import (
	"encoding/base64"
	"fmt"
	"reflect"

	"github.com/altessa-s/go-convert/reflectx"

	convcodec "github.com/altessa-s/go-convert/codec"
)

// Error reports a string that is not valid base64 in the configured encoding.
type Error struct {
	// Path is the full source field path; it is empty at the root.
	Path string
	// Err is the decoding error, a base64.CorruptInputError.
	Err error
}

// Error reports the field path and the decoding failure.
func (e *Error) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("b64codec: %v", e.Err)
	}

	return fmt.Sprintf("b64codec: field %q: %v", e.Path, e.Err)
}

// Unwrap returns the decoding error.
func (e *Error) Unwrap() error { return e.Err }

// New creates an error codec for bidirectional conversion between byte
// slices and base64 strings.
//
// Supported conversions (pointer variants on either side included):
//   - []byte kind -> string kind: encodes with the configured encoding
//   - string kind -> []byte kind: decodes with the configured encoding
//
// The encoding is base64.StdEncoding unless [WithEncoding] sets another. An
// empty string decodes to a non-nil empty slice, keeping []byte{} apart from
// nil; only a nil source pointer sets the destination to its zero value (a
// nil slice, "" or a nil pointer).
// Invalid input is returned as an [*Error] and leaves the destination
// unchanged.
//
// Example:
//
//	codec := b64codec.New()
//	codec := b64codec.New(b64codec.WithEncoding(base64.RawURLEncoding))
func New(opts ...Option) convcodec.CodecE {
	o := newOptions(opts...)

	return func(fieldPath string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
		if handled, err := tryConvert(src, dst, o.encoding); handled {
			if err != nil {
				return &Error{Path: fieldPath, Err: err}
			}

			return nil
		}

		return next(fieldPath, src, dst)
	}
}

// tryConvert converts between byte slices and base64 strings. It reports
// whether the conversion was handled; a nil source sets dst to its zero
// value.
func tryConvert(src, dst reflect.Value, enc *base64.Encoding) (bool, error) {
	srcType := reflectx.IndirectType(src.Type())
	dstType := reflectx.IndirectType(dst.Type())

	switch {
	case isBytes(srcType) && dstType.Kind() == reflect.String:
		srcValue := reflectx.IndirectValue(src)
		if !srcValue.IsValid() {
			dst.SetZero()
			return true, nil
		}

		reflectx.MakeDst(&dst, dstType)
		dst.SetString(enc.EncodeToString(srcValue.Bytes()))

		return true, nil

	case srcType.Kind() == reflect.String && isBytes(dstType):
		srcValue := reflectx.IndirectValue(src)
		if !srcValue.IsValid() {
			dst.SetZero()
			return true, nil
		}

		// DecodeString returns a non-nil empty slice for "".
		decoded, err := enc.DecodeString(srcValue.String())
		if err != nil {
			return true, err
		}

		reflectx.MakeDst(&dst, dstType)
		dst.SetBytes(decoded)

		return true, nil

	default:
		return false, nil
	}
}

// isBytes reports whether t is a byte slice kind.
func isBytes(t reflect.Type) bool {
	return t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8
}
