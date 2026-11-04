// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package anypb

import (
	"fmt"
	"reflect"

	"google.golang.org/protobuf/reflect/protoreflect"
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

// Error reports the field path, the type URL and the expected type.
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

// Error reports the field path and the unresolved type.
func (e UnknownTypeError) Error() string {
	if e.Type != nil {
		return fmt.Sprintf("anypb: %q: %s is not a protobuf message or a registered domain type", e.Path, e.Type)
	}

	return fmt.Sprintf("anypb: %q: unknown message type %q", e.Path, e.TypeURL)
}

// RegistrationError reports a pair [Registry.Register] refuses.
type RegistrationError struct {
	// Message is the full name of the message type, when known.
	Message protoreflect.FullName
	// Domain is the domain type, when known.
	Domain reflect.Type
	// Reason describes the problem.
	Reason string
}

// Error reports the pair and the problem.
func (e RegistrationError) Error() string {
	if e.Message == "" && e.Domain == nil {
		return "anypb: register: " + e.Reason
	}

	return fmt.Sprintf("anypb: register %s as %v: %s", e.Message, e.Domain, e.Reason)
}

// Error reports a protobuf marshaling or unmarshaling failure.
type Error struct {
	// Path is the source field path given to the codec.
	Path string
	// Err is the underlying protobuf error.
	Err error
}

// Error reports the field path and the cause.
func (e Error) Error() string {
	return fmt.Sprintf("anypb: %q: %v", e.Path, e.Err)
}

// Unwrap returns the underlying error.
func (e Error) Unwrap() error { return e.Err }
