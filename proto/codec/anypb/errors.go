// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package anypb

import (
	"fmt"
	"reflect"
)

// TypeMismatchError reports an Any whose type URL does not name the message
// type the destination requires.
type TypeMismatchError struct {
	// Path is the source field path given to the codec.
	Path string
	// TypeURL is the type URL of the source Any.
	TypeURL string
	// Want is the message full name, or the Go type, the destination needs.
	Want string
}

func (e TypeMismatchError) Error() string {
	return fmt.Sprintf("anypb: %q: type URL %q does not match %s", e.Path, e.TypeURL, e.Want)
}

// UnknownTypeError reports a type the codec cannot map: an Any type URL that
// neither the [Registry] nor the resolver knows, or a Go source type that is
// neither a protobuf message nor a registered domain type.
type UnknownTypeError struct {
	// Path is the source field path given to the codec.
	Path string
	// TypeURL is the unresolved type URL, when unpacking an Any.
	TypeURL string
	// Type is the unregistered Go type, when packing into an Any.
	Type reflect.Type
}

func (e UnknownTypeError) Error() string {
	if e.Type != nil {
		return fmt.Sprintf("anypb: %q: %s is not a protobuf message or a registered domain type", e.Path, e.Type)
	}
	return fmt.Sprintf("anypb: %q: unknown message type %q", e.Path, e.TypeURL)
}

// Error reports a protobuf marshaling or unmarshaling failure.
type Error struct {
	// Path is the source field path given to the codec.
	Path string
	// Err is the underlying protobuf error.
	Err error
}

func (e Error) Error() string {
	return fmt.Sprintf("anypb: %q: %v", e.Path, e.Err)
}

// Unwrap returns the underlying error.
func (e Error) Unwrap() error { return e.Err }
