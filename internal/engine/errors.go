// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package engine

import (
	"fmt"
	"reflect"

	reflectutils "github.com/altessa-s/go-convert/internal/reflect"
)

// UnsupportedTypeError reports a source and destination combination the
// converter cannot handle.
type UnsupportedTypeError struct {
	Source      any
	Destination any
	Reason      string
	// Err is the underlying error when the failure was raised as one.
	Err error
}

// Error reports the source and destination types and the reason.
func (e UnsupportedTypeError) Error() string {
	return fmt.Sprintf("convert: unsupported conversion from %T to %T: %s", e.Source, e.Destination, e.Reason)
}

// Unwrap returns the underlying error, if any.
func (e UnsupportedTypeError) Unwrap() error { return e.Err }

// OverflowError reports a narrowing numeric conversion that would lose data.
type OverflowError struct {
	Field string
	From  reflect.Kind
	To    reflect.Kind
	Value any
}

// Error reports the field, the value and the kinds involved.
func (e OverflowError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("convert: field %q: %s value %v overflows %s", e.Field, e.From, e.Value, e.To)
	}

	return fmt.Sprintf("convert: %s value %v overflows %s", e.From, e.Value, e.To)
}

// newOverflowError describes the overflow of src when written to dst.
func newOverflowError(field string, src reflect.Value, dst reflect.Kind) OverflowError {
	src = reflectutils.IndirectValue(src)
	return OverflowError{Field: field, From: src.Kind(), To: dst, Value: src.Interface()}
}

// DepthError reports input nested deeper than the configured limit, which
// includes cyclic pointer graphs.
type DepthError struct {
	Limit int
	Path  string
}

// Error reports the limit and, when tracked, the path.
func (e DepthError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("convert: nesting exceeds %d levels at %q", e.Limit, e.Path)
	}

	return fmt.Sprintf("convert: nesting exceeds %d levels", e.Limit)
}

// codecError distinguishes errors returned by codecs from conversion panics.
type codecError struct {
	error
}

// Unwrap returns the codec error.
func (e codecError) Unwrap() error { return e.error }
