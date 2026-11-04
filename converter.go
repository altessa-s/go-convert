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
	return &Converter[T, U]{engine: engineFor(opts)}
}

// defaultEngine serves every conversion configured without options. An
// engine is immutable, so one instance can be shared.
var defaultEngine = engine.New(newOptions().config())

// engineFor returns the engine configured by opts, reusing [defaultEngine]
// when there are none.
func engineFor(opts []Option) *engine.Engine {
	if len(opts) == 0 {
		return defaultEngine
	}

	return engine.New(newOptions(opts...).config())
}

// NewAny returns a converter that accepts any source and destination types.
func NewAny(opts ...Option) *Converter[any, any] {
	return New[any, any](opts...)
}

// Convert converts src into dst and returns dst. A nil src leaves dst
// unchanged. Convert panics with an [UnsupportedTypeError], [OverflowError],
// [DepthError] or a codec error; use [ConvertE] to receive them as errors.
func Convert[T, U any](src T, dst U, opts ...Option) U {
	if err := publicError(engineFor(opts).Convert(src, dst)); err != nil {
		panic(err)
	}

	return dst
}

// ConvertE is like [Convert] but returns failures as errors. dst may be
// partially populated when an error is returned.
func ConvertE[T, U any](src T, dst U, opts ...Option) (U, error) {
	return dst, publicError(engineFor(opts).Convert(src, dst))
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
		// One destination, zeroed before each element, receives every
		// conversion; yield gets a copy.
		var dst, zero U

		slot := reflect.ValueOf(&dst).Elem()
		for _, item := range src {
			dst = zero

			must(c.engine.ConvertValue(reflect.ValueOf(item), slot))

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
		// One key and one value destination, zeroed before each entry,
		// receive every conversion; yield gets copies.
		var (
			dk, zeroKey   K2
			dv, zeroValue V2
		)

		keySlot, valueSlot := reflect.ValueOf(&dk).Elem(), reflect.ValueOf(&dv).Elem()
		for k, v := range src {
			dk, dv = zeroKey, zeroValue

			must(c.engine.ConvertKey(reflect.ValueOf(k), keySlot))
			must(c.engine.ConvertValue(reflect.ValueOf(v), valueSlot))

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
