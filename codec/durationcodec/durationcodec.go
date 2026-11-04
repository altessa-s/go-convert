// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package durationcodec

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"

	convcodec "github.com/altessa-s/go-convert/codec"
	reflectutils "github.com/altessa-s/go-convert/internal/reflect"
)

var durationType = reflect.TypeFor[time.Duration]()

// Error reports a value that could not be converted to or from a
// time.Duration.
type Error struct {
	// Path is the full source field path; it is empty at the root.
	Path string
	// Value is the source value that failed to convert.
	Value any
	// Err describes the failure. Out-of-range and non-finite numbers wrap
	// strconv.ErrRange.
	Err error
}

func (e *Error) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("durationcodec: %v", e.Err)
	}
	return fmt.Sprintf("durationcodec: field %q: %v", e.Path, e.Err)
}

// Unwrap returns the underlying failure.
func (e *Error) Unwrap() error { return e.Err }

// New creates an error codec for conversion between time.Duration and
// strings or numbers.
//
// Supported conversions (pointer variants on either side included):
//   - time.Duration -> string kind: calls d.String(), such as "1h30m0s"
//   - string kind -> time.Duration: calls time.ParseDuration
//   - time.Duration -> integer kind: d / unit, truncated toward zero
//   - time.Duration -> float kind: d / unit with a fraction
//   - integer or float kind -> time.Duration: n * unit, floats rounded to the nanosecond
//
// The unit is time.Second unless [WithUnit] sets another. An empty string
// sets the destination to its zero value (nil for pointer destinations); a
// nil source pointer leaves the destination unchanged. Parse errors, values
// that overflow the destination and non-finite floats are returned as an
// [*Error].
//
// Example:
//
//	codec := durationcodec.New()
//	codec := durationcodec.New(durationcodec.WithUnit(time.Millisecond))
func New(opts ...Option) convcodec.CodecE {
	o := newOptions(opts...)

	return func(fieldPath string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
		if handled, err := tryConvert(src, dst, o.unit); handled {
			if err != nil {
				return &Error{Path: fieldPath, Value: sourceValue(src), Err: err}
			}
			return nil
		}

		return next(fieldPath, src, dst)
	}
}

// tryConvert converts between time.Duration and strings or numbers. It
// reports whether the conversion was handled (including a nil source).
func tryConvert(src, dst reflect.Value, unit time.Duration) (bool, error) {
	srcType := reflectutils.IndirectType(src.Type())
	dstType := reflectutils.IndirectType(dst.Type())

	switch {
	case srcType == durationType && dstType != durationType:
		if !isString(dstType.Kind()) && !isNumber(dstType.Kind()) {
			return false, nil
		}
		srcValue := reflectutils.IndirectValue(src)
		if !srcValue.IsValid() {
			return true, nil
		}
		return true, fromDuration(time.Duration(srcValue.Int()), dst, dstType, unit)

	case dstType == durationType && srcType != durationType:
		if !isString(srcType.Kind()) && !isNumber(srcType.Kind()) {
			return false, nil
		}
		srcValue := reflectutils.IndirectValue(src)
		if !srcValue.IsValid() {
			return true, nil
		}
		if srcValue.Kind() == reflect.String && srcValue.Len() == 0 {
			dst.SetZero()
			return true, nil
		}
		d, err := toDuration(srcValue, unit)
		if err != nil {
			return true, err
		}
		reflectutils.MakeDst(&dst, dstType)
		dst.SetInt(int64(d))
		return true, nil

	default:
		return false, nil
	}
}

// fromDuration stores d into a string or numeric destination.
func fromDuration(d time.Duration, dst reflect.Value, dstType reflect.Type, unit time.Duration) error {
	if dstType.Kind() == reflect.String {
		reflectutils.MakeDst(&dst, dstType)
		dst.SetString(d.String())
		return nil
	}

	n := int64(d / unit)
	switch dstType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if reflect.Zero(dstType).OverflowInt(n) {
			return overflow(d, dstType)
		}
		reflectutils.MakeDst(&dst, dstType)
		dst.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if n < 0 || reflect.Zero(dstType).OverflowUint(uint64(n)) {
			return overflow(d, dstType)
		}
		reflectutils.MakeDst(&dst, dstType)
		dst.SetUint(uint64(n))
	default: // float kinds; a float32 holds every int64 magnitude
		reflectutils.MakeDst(&dst, dstType)
		dst.SetFloat(float64(d) / float64(unit))
	}

	return nil
}

// toDuration parses a string or scales a number into a time.Duration.
func toDuration(v reflect.Value, unit time.Duration) (time.Duration, error) {
	switch v.Kind() {
	case reflect.String:
		return time.ParseDuration(v.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n := v.Int()
		if n > math.MaxInt64/int64(unit) || n < math.MinInt64/int64(unit) {
			return 0, fmt.Errorf("%d overflows time.Duration: %w", n, strconv.ErrRange)
		}
		return time.Duration(n) * unit, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		n := v.Uint()
		if n > uint64(math.MaxInt64/int64(unit)) {
			return 0, fmt.Errorf("%d overflows time.Duration: %w", n, strconv.ErrRange)
		}
		return time.Duration(n) * unit, nil
	default: // float kinds
		f := v.Float() * float64(unit)
		// float64(math.MaxInt64) rounds up to 2^63, the first value out of range.
		if math.IsNaN(f) || f >= float64(math.MaxInt64) || f < float64(math.MinInt64) {
			return 0, fmt.Errorf("%v overflows time.Duration: %w", v.Float(), strconv.ErrRange)
		}
		return time.Duration(math.Round(f)), nil
	}
}

// sourceValue returns the dereferenced source for an [Error], or nil when it
// cannot be exposed.
func sourceValue(src reflect.Value) any {
	v := reflectutils.IndirectValue(src)
	if !v.IsValid() || !v.CanInterface() {
		return nil
	}
	return v.Interface()
}

func overflow(d time.Duration, t reflect.Type) error {
	return fmt.Errorf("%s overflows %s: %w", d, t, strconv.ErrRange)
}

func isString(k reflect.Kind) bool { return k == reflect.String }

func isNumber(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}
