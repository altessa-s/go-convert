// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package jsoncodec

//go:generate optgen generate --no-default-modifiers

import (
	"encoding/json/v2"
)

// options holds configuration options for JSON value conversion.
type options struct {
	// byteText ([WithBytes]) also treats plain []byte fields as JSON text.
	// It is off by default so that ordinary byte slices are not decoded.
	byteText bool `opt:"Bytes"`
	// stringText ([WithStrings]) also treats plain string fields as JSON
	// text. It is off by default so that ordinary string fields are not
	// decoded.
	stringText bool `opt:"Strings"`
	// jsonOptions ([WithJSONOptions]) are passed to json.Marshal and
	// json.Unmarshal from encoding/json/v2, such as json.Deterministic or
	// json.RejectUnknownMembers.
	jsonOptions []json.Options `opt:"JSONOptions" optgen:"append"`
	// maxBytes ([WithMaxBytes]) is the longest JSON text, in bytes, that is
	// decoded; longer text is rejected with an [Error] wrapping [SizeError]
	// before decoding starts. Zero, the default, or a negative value means no
	// limit.
	maxBytes int `opt:"MaxBytes"`
}
