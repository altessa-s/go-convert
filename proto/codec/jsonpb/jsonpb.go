// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package jsonpb

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"reflect"

	"google.golang.org/protobuf/types/known/structpb"

	convcodec "github.com/altessa-s/go-convert/codec"
	reflectutils "github.com/altessa-s/go-convert/internal/reflect"
)

var (
	// rawMessageType is jsontext.Value; encoding/json.RawMessage is an alias
	// of it.
	rawMessageType = reflect.TypeFor[jsontext.Value]()
	structpbType   = reflect.TypeFor[structpb.Struct]()
)

// New creates a Codec for bidirectional conversion between json.RawMessage and structpb.Struct.
//
// Supported conversions (pointer fields only):
//   - *json.RawMessage -> *structpb.Struct: unmarshals JSON into a map then builds a Struct
//   - *structpb.Struct -> *json.RawMessage: marshals the Struct to JSON bytes
//
// Non-pointer variants (json.RawMessage <-> structpb.Struct) are also supported.
// JSON text is parsed and rendered with encoding/json/v2, so duplicate object
// names and invalid UTF-8 are rejected (the field is passed to next) and
// object members are written in sorted order.
//
// Example:
//
//	codec := jsonpb.New()
//	codec := jsonpb.New(jsonpb.WithIgnoreNil())
func New(opts ...Option) convcodec.Codec {
	o := newOptions(opts...)

	return func(fieldName string, src, dst reflect.Value, next convcodec.CodecHandler) {
		if tryConvert(src, dst, o) {
			return
		}

		next(fieldName, src, dst)
	}
}

// tryConvert attempts to convert between json.RawMessage and structpb.Struct.
// Returns true if conversion was handled, false otherwise.
func tryConvert(src, dst reflect.Value, opts *options) bool {
	srcType := reflectutils.IndirectType(src.Type())
	dstType := reflectutils.IndirectType(dst.Type())

	srcValue := reflectutils.IndirectValue(src)

	switch {
	case srcType == rawMessageType && dstType == structpbType:
		return convertRawMessageToStruct(srcValue, dst, dstType, opts)

	case srcType == structpbType && dstType == rawMessageType:
		return convertStructToRawMessage(srcValue, dst, dstType, opts)

	default:
		return false
	}
}

// convertRawMessageToStruct handles json.RawMessage -> structpb.Struct conversion.
func convertRawMessageToStruct(srcValue, dst reflect.Value, dstType reflect.Type, opts *options) bool {
	// A nil pointer, empty text and JSON null are absent: they zero the
	// destination unless WithIgnoreNil keeps it.
	var raw jsontext.Value

	if srcValue.IsValid() {
		var ok bool
		if raw, ok = reflect.TypeAssert[jsontext.Value](srcValue); !ok {
			return false
		}
	}

	if trimmed := bytes.TrimSpace(raw); len(trimmed) == 0 || string(trimmed) == "null" {
		if !opts.ignoreNil {
			reflectutils.SetZero(dst)
		}

		return true
	}

	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return false
	}

	pb, err := structpb.NewStruct(m)
	if err != nil {
		return false
	}

	reflectutils.MakeDst(&dst, dstType)
	dst.Set(reflect.ValueOf(pb).Elem())

	return true
}

// convertStructToRawMessage handles structpb.Struct -> json.RawMessage conversion.
func convertStructToRawMessage(srcValue, dst reflect.Value, dstType reflect.Type, opts *options) bool {
	// A nil source zeroes the destination unless WithIgnoreNil keeps it.
	if !srcValue.IsValid() {
		if !opts.ignoreNil {
			reflectutils.SetZero(dst)
		}

		return true
	}

	var pb *structpb.Struct

	if srcValue.CanAddr() {
		var ok bool

		pb, ok = reflect.TypeAssert[*structpb.Struct](srcValue.Addr())
		if !ok {
			return false
		}
	} else {
		// Value is not addressable (e.g. passed directly via reflect.ValueOf).
		// Create an addressable copy.
		tmp := reflect.New(structpbType)
		tmp.Elem().Set(srcValue)

		var ok bool

		pb, ok = reflect.TypeAssert[*structpb.Struct](tmp)
		if !ok {
			return false
		}
	}

	data, err := json.Marshal(pb.AsMap(), jsonOptions)
	if err != nil {
		return false
	}

	raw := jsontext.Value(data)

	reflectutils.MakeDst(&dst, dstType)
	dst.Set(reflect.ValueOf(raw))

	return true
}
