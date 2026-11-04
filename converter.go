// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package convert

import (
	"iter"
	"reflect"

	"github.com/altessa-s/go-convert/internal/engine"
)

// Converter converts values of type T into destinations of type U with a
// fixed set of options. It is safe for concurrent use.
//
// T is a struct, a pointer to a struct, or a slice of either; U is a non-nil
// pointer to the same kind. Unsupported combinations are reported at run time.
type Converter[T, U any] struct {
	engine *engine.Engine
}

// New returns a converter configured by opts. Type metadata is cached
// process-wide, so creating converters is cheap; reuse one to avoid
// re-applying options.
func New[T, U any](opts ...Option) *Converter[T, U] {
	return &Converter[T, U]{engine: engine.New(newOptions(opts...).config())}
}

// NewAny returns a converter that accepts any source and destination types.
func NewAny(opts ...Option) *Converter[any, any] {
	return New[any, any](opts...)
}

// Convert converts src into dst and returns dst. A nil src leaves dst
// unchanged. Convert panics with an [UnsupportedTypeError], [OverflowError],
// [DepthError] or a codec error; use [ConvertE] to receive them as errors.
func Convert[T, U any](src T, dst U, opts ...Option) U {
	New[T, U](opts...).Convert(src, dst)
	return dst
}

// ConvertE is like [Convert] but returns failures as errors. dst may be
// partially populated when an error is returned.
func ConvertE[T, U any](src T, dst U, opts ...Option) (U, error) {
	return dst, New[T, U](opts...).ConvertE(src, dst)
}

// ConvertSeq returns an iterator that converts each element of src into a U
// on demand.
func ConvertSeq[U, T any](src []T, opts ...Option) iter.Seq[U] {
	return New[T, U](opts...).ConvertSeq(src)
}

// ConvertMapSeq returns an iterator that converts each entry of src into a
// K2 key and a V2 value on demand. Iteration order is unspecified.
func ConvertMapSeq[K2 comparable, V2 any, K1 comparable, V1 any](src map[K1]V1, opts ...Option) iter.Seq2[K2, V2] {
	return NewAny(opts...).ConvertMapSeq[K2, V2](src)
}

// Convert converts src into dst; see the package-level [Convert].
func (c *Converter[T, U]) Convert(src T, dst U) {
	if err := c.ConvertE(src, dst); err != nil {
		panic(err)
	}
}

// ConvertE converts src into dst and returns any failure as an error.
func (c *Converter[T, U]) ConvertE(src T, dst U) error {
	return publicError(c.engine.Convert(src, dst))
}

// ConvertSeq returns an iterator that converts each element of src into a U
// on demand. Failures panic as in [Converter.Convert].
func (c *Converter[T, U]) ConvertSeq(src []T) iter.Seq[U] {
	return func(yield func(U) bool) {
		for _, item := range src {
			var dst U
			must(c.engine.ConvertValue(reflect.ValueOf(item), reflect.ValueOf(&dst).Elem()))
			if !yield(dst) {
				return
			}
		}
	}
}

// ConvertMapSeq returns an iterator that converts each entry of src into a
// K2 key and a V2 value on demand. Ignore paths never apply to map keys.
// Failures panic as in [Converter.Convert].
func (c *Converter[T, U]) ConvertMapSeq[K2 comparable, V2 any, K1 comparable, V1 any](src map[K1]V1) iter.Seq2[K2, V2] {
	return func(yield func(K2, V2) bool) {
		for k, v := range src {
			var dk K2
			var dv V2
			must(c.engine.ConvertKey(reflect.ValueOf(k), reflect.ValueOf(&dk).Elem()))
			must(c.engine.ConvertValue(reflect.ValueOf(v), reflect.ValueOf(&dv).Elem()))
			if !yield(dk, dv) {
				return
			}
		}
	}
}

// must panics with the public form of err, as iterators cannot return errors.
func must(err error) {
	if err != nil {
		panic(publicError(err))
	}
}
