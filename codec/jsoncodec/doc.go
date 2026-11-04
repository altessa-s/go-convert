// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package jsoncodec provides an error codec for bidirectional conversion
// between JSON text and Go values using encoding/json/v2.
//
// [New] treats jsontext.Value and json.RawMessage fields as JSON text, and
// also plain []byte or string fields when enabled with [WithBytes] or
// [WithStrings]; these are off by default so ordinary byte slices and strings
// are never decoded. JSON text is unmarshaled into, and marshaled from,
// structs, maps, slices and empty interfaces. Both pointer and non-pointer
// variants are supported. Options from encoding/json/v2 and
// encoding/json/jsontext are passed through with [WithJSONOptions], and
// [WithMaxBytes] limits the size of decoded JSON text, which is unlimited by
// default.
//
// Marshal and unmarshal failures are returned as an [Error] that wraps the
// encoding/json/v2 error; convert.ConvertE returns it unchanged.
//
// Example:
//
//	codec := jsoncodec.New(jsoncodec.WithStrings())
//	_, err := convert.ConvertE(src, &dst, convert.WithErrorCodecs(codec))
package jsoncodec
