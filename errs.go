// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package convert

import (
	"fmt"
	"reflect"

	"github.com/altessa-s/go-convert/internal/engine"
)

// UnsupportedTypeError reports a source and destination combination the
// converter cannot handle, such as a struct converted into a slice pointer.
type UnsupportedTypeError struct {
	Source      any
	Destination any
	Reason      string
	// Err is the underlying error when the failure was raised as one, for
	// example by a codec that panicked with an error.
	Err error
}

func (e UnsupportedTypeError) Error() string {
	return fmt.Sprintf("convert: unsupported conversion from %T to %T: %s", e.Source, e.Destination, e.Reason)
}

// Unwrap returns the underlying error, if any.
func (e UnsupportedTypeError) Unwrap() error { return e.Err }

// OverflowError reports a narrowing numeric conversion that would lose data.
// It is only produced when [WithOverflowCheck] is set.
type OverflowError struct {
	// Field is the name of the field being converted, if known.
	Field string
	From  reflect.Kind
	To    reflect.Kind
	Value any
}

func (e OverflowError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("convert: field %q: %s value %v overflows %s", e.Field, e.From, e.Value, e.To)
	}
	return fmt.Sprintf("convert: %s value %v overflows %s", e.From, e.Value, e.To)
}

// DepthError reports input nested deeper than the limit set by
// [WithMaxDepth], which includes cyclic pointer graphs.
type DepthError struct {
	Limit int
	// Path is the dotted path at which the limit was exceeded; it is only
	// tracked when ignore paths or error codecs are configured.
	Path string
}

func (e DepthError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("convert: nesting exceeds %d levels at %q", e.Limit, e.Path)
	}
	return fmt.Sprintf("convert: nesting exceeds %d levels", e.Limit)
}

// publicError maps the engine's error types to the exported ones. Any other
// error, such as one returned by a codec, is returned unchanged.
func publicError(err error) error {
	switch e := err.(type) {
	case engine.UnsupportedTypeError:
		return UnsupportedTypeError(e)
	case engine.OverflowError:
		return OverflowError(e)
	case engine.DepthError:
		return DepthError(e)
	default:
		return err
	}
}
