// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package jsonpb

import "fmt"

// ValueError reports a value that google.protobuf.Value cannot hold: a
// non-finite number (NaN or ±Inf) or a Go type with no JSON counterpart. It
// is returned by the [NewE] codec.
type ValueError struct {
	// Path is the source field path given to the codec, followed by the
	// position of the value inside nested lists ("[2]") and maps (".key").
	Path string
	// Value is the offending value.
	Value any
	// Reason says why the value was rejected.
	Reason string
}

func (e ValueError) Error() string {
	return fmt.Sprintf("jsonpb: %q: cannot convert %T value %v: %s", e.Path, e.Value, e.Value, e.Reason)
}

// Error reports a JSON encoding or decoding failure in the [NewE] codec, such
// as a json.RawMessage that is not a JSON object.
type Error struct {
	// Path is the source field path given to the codec.
	Path string
	// Err is the underlying encoding/json error.
	Err error
}

func (e Error) Error() string {
	return fmt.Sprintf("jsonpb: %q: %v", e.Path, e.Err)
}

// Unwrap returns the underlying error.
func (e Error) Unwrap() error { return e.Err }
