// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package tspb

import (
	"reflect"
	"time"
	"unsafe"

	"google.golang.org/protobuf/types/known/timestamppb"

	convcodec "github.com/altessa-s/go-convert/codec"
	reflectutils "github.com/altessa-s/go-convert/internal/reflect"
)

var (
	timestampType = reflect.TypeFor[timestamppb.Timestamp]()
	timeType      = reflect.TypeFor[time.Time]()
	int64Type     = reflect.TypeFor[int64]()

	// Pre-computed field offsets for zero-alloc Timestamp read/write via unsafe.
	tsSecondsOffset = func() uintptr {
		f, ok := timestampType.FieldByName("Seconds")
		if !ok {
			panic("tspb: timestamppb.Timestamp has no Seconds field")
		}
		return f.Offset
	}()
	tsNanosOffset = func() uintptr {
		f, ok := timestampType.FieldByName("Nanos")
		if !ok {
			panic("tspb: timestamppb.Timestamp has no Nanos field")
		}
		return f.Offset
	}()
)

// New creates a Codec for bidirectional conversion between timestamppb.Timestamp and time.Time / int64.
//
// Supported conversions:
//   - timestamppb.Timestamp <-> time.Time
//   - timestamppb.Timestamp <-> int64 (Unix seconds, or millis with WithMilliseconds)
//   - Pointer variants (*timestamppb.Timestamp <-> *time.Time, etc.)
//
// Example:
//
//	codec := tspb.New()
//	codec := tspb.New(tspb.WithIgnoreZero(), tspb.WithMilliseconds())
func New(opts ...Option) convcodec.Codec {
	o := newOptions(opts...)

	return func(fieldName string, src, dst reflect.Value, next convcodec.CodecHandler) {
		if tryConvert(src, dst, o) {
			return
		}

		next(fieldName, src, dst)
	}
}

// tryConvert attempts to convert between timestamppb.Timestamp and time.Time or int64.
// Returns true if conversion was handled (including skipped due to options), false otherwise.
func tryConvert(src, dst reflect.Value, opts *options) bool {
	srcType := reflectutils.IndirectType(src.Type())
	dstType := reflectutils.IndirectType(dst.Type())

	srcValue := reflectutils.IndirectValue(src)

	switch {
	case srcType == timestampType && dstType == timeType:
		return convertTimestampToTime(srcValue, dst, dstType, opts)

	case srcType == timeType && dstType == timestampType:
		return convertTimeToTimestamp(srcValue, dst, dstType, opts)

	case srcType == timestampType && dstType == int64Type:
		return convertTimestampToInt64(srcValue, dst, dstType, opts)

	case srcType == int64Type && dstType == timestampType:
		return convertInt64ToTimestamp(srcValue, dst, dstType, opts)

	default:
		return false
	}
}

// convertTimestampToTime handles timestamppb.Timestamp -> time.Time conversion.
func convertTimestampToTime(srcValue reflect.Value, dst reflect.Value, dstType reflect.Type, opts *options) bool {
	if !srcValue.IsValid() {
		return true
	}

	sec, nanos, ok := readTimestamp(srcValue)
	if !ok {
		return false
	}

	if opts.ignoreZero && sec == 0 && nanos == 0 {
		return true
	}

	t := time.Unix(sec, int64(nanos))

	reflectutils.MakeDst(&dst, dstType)
	writeTime(dst, t)

	return true
}

// convertTimeToTimestamp handles time.Time -> timestamppb.Timestamp conversion.
func convertTimeToTimestamp(srcValue reflect.Value, dst reflect.Value, dstType reflect.Type, opts *options) bool {
	if !srcValue.IsValid() {
		return true
	}

	t, ok := readTime(srcValue)
	if !ok {
		return false
	}

	if opts.ignoreZero && t.IsZero() {
		return true
	}

	reflectutils.MakeDst(&dst, dstType)
	writeTimestamp(dst, t.Unix(), int32(t.Nanosecond()))

	return true
}

// convertTimestampToInt64 handles timestamppb.Timestamp -> int64 conversion.
func convertTimestampToInt64(srcValue reflect.Value, dst reflect.Value, dstType reflect.Type, opts *options) bool {
	if !srcValue.IsValid() {
		return true
	}

	sec, nanos, ok := readTimestamp(srcValue)
	if !ok {
		return false
	}

	if opts.ignoreZero && sec == 0 && nanos == 0 {
		return true
	}

	var v int64
	if opts.milliseconds {
		v = time.Unix(sec, int64(nanos)).UnixMilli()
	} else {
		v = sec
	}

	reflectutils.MakeDst(&dst, dstType)
	dst.SetInt(v)

	return true
}

// convertInt64ToTimestamp handles int64 -> timestamppb.Timestamp conversion.
func convertInt64ToTimestamp(srcValue reflect.Value, dst reflect.Value, dstType reflect.Type, opts *options) bool {
	if !srcValue.IsValid() {
		return true
	}

	v := srcValue.Int()

	if opts.ignoreZero && v == 0 {
		return true
	}

	var sec int64
	var nanos int32

	if opts.milliseconds {
		t := time.UnixMilli(v)
		sec = t.Unix()
		nanos = int32(t.Nanosecond())
	} else {
		sec = v
	}

	reflectutils.MakeDst(&dst, dstType)
	writeTimestamp(dst, sec, nanos)

	return true
}

// readTimestamp extracts Seconds and Nanos from a Timestamp reflect.Value
// without allocating. Uses unsafe pointer arithmetic when the value is addressable,
// falls back to reflect field access otherwise.
//
// #nosec G103 -- intentional unsafe for zero-alloc Timestamp read with verified safety
func readTimestamp(v reflect.Value) (seconds int64, nanos int32, ok bool) {
	if !v.IsValid() || v.Type() != timestampType {
		return 0, 0, false
	}
	if v.CanAddr() {
		base := unsafe.Pointer(v.UnsafeAddr())
		seconds = *(*int64)(unsafe.Add(base, tsSecondsOffset))
		nanos = *(*int32)(unsafe.Add(base, tsNanosOffset))

		return seconds, nanos, true
	}

	// Non-addressable fallback: use reflect field access
	secField := v.FieldByName("Seconds")
	nanField := v.FieldByName("Nanos")
	if !secField.IsValid() || !nanField.IsValid() {
		return 0, 0, false
	}

	return secField.Int(), int32(nanField.Int()), true
}

// writeTimestamp writes Seconds and Nanos directly into a Timestamp reflect.Value
// without allocating via timestamppb.New() + reflect.ValueOf().
// Uses unsafe pointer arithmetic when the value is addressable,
// falls back to timestamppb.New() otherwise.
//
// #nosec G103 -- intentional unsafe for zero-alloc Timestamp write with verified safety
func writeTimestamp(dst reflect.Value, seconds int64, nanos int32) {
	if dst.CanAddr() {
		base := unsafe.Pointer(dst.UnsafeAddr())
		*(*int64)(unsafe.Add(base, tsSecondsOffset)) = seconds
		*(*int32)(unsafe.Add(base, tsNanosOffset)) = nanos

		return
	}

	ts := &timestamppb.Timestamp{Seconds: seconds, Nanos: nanos}
	dst.Set(reflect.ValueOf(ts).Elem())
}

// readTime extracts time.Time from a reflect.Value without heap-allocating
// via Interface() boxing. Falls back to Interface() for non-addressable values.
//
// #nosec G103 -- intentional unsafe for zero-alloc time.Time read with verified safety
func readTime(v reflect.Value) (time.Time, bool) {
	if !v.IsValid() || v.Type() != timeType {
		return time.Time{}, false
	}
	if v.CanAddr() {
		return *(*time.Time)(unsafe.Pointer(v.UnsafeAddr())), true
	}

	t, ok := reflect.TypeAssert[time.Time](v)
	return t, ok
}

// writeTime sets a time.Time into a reflect.Value without heap-allocating
// via reflect.ValueOf(). Falls back to reflect.ValueOf for non-addressable values.
//
// #nosec G103 -- intentional unsafe for zero-alloc time.Time write with verified safety
func writeTime(dst reflect.Value, t time.Time) {
	if dst.CanAddr() {
		*(*time.Time)(unsafe.Pointer(dst.UnsafeAddr())) = t
		return
	}

	dst.Set(reflect.ValueOf(t))
}
