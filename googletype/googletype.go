// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package googletype

import (
	"maps"
	"reflect"
	"sync"
	"time"

	"github.com/altessa-s/go-convert/reflectx"

	convcodec "github.com/altessa-s/go-convert/codec"
)

var (
	stringType   = reflect.TypeFor[string]()
	int64Type    = reflect.TypeFor[int64]()
	timeType     = reflect.TypeFor[time.Time]()
	durationType = reflect.TypeFor[time.Duration]()
	latLngPair   = reflect.TypeFor[[2]float64]()
)

// pair is one conversion request. src is the source with every pointer layer
// followed (invalid when a pointer in the chain is nil); dst is the
// destination as received; srcType and dstType have their pointers removed.
type pair struct {
	path    string
	src     reflect.Value
	dst     reflect.Value
	srcType reflect.Type
	dstType reflect.Type
}

// convertFunc converts p when it recognizes the pair of types. It reports
// whether it handled the pair; err is only meaningful when handled is true.
type convertFunc func(p pair, o *options) (handled bool, err error)

// New creates a CodecE that combines [NewMoney], [NewDate], [NewLatLng] and
// [NewTimeOfDay] with one set of options. Pairs that involve none of their
// messages are passed to next.
//
// Example:
//
//	codec := googletype.New(googletype.WithCurrency("EUR"))
func New(opts ...Option) convcodec.CodecE {
	return newCodec(buildOptions(opts), convertMoney, convertDate, convertLatLng, convertTimeOfDay)
}

// buildOptions applies opts and copies the caller's map, so that changing it
// later cannot race with conversions, and gives the codec its own cache of
// Money struct shapes.
func buildOptions(opts []Option) *options {
	o := newOptions(opts...)
	o.currencyScales = maps.Clone(o.currencyScales)
	o.moneyShapes = new(sync.Map)

	return o
}

// newCodec builds a CodecE that tries fns in order and delegates the pairs
// none of them handles to next.
func newCodec(o *options, fns ...convertFunc) convcodec.CodecE {
	return func(fieldPath string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
		p := pair{
			path:    fieldPath,
			src:     reflectx.IndirectValue(src),
			dst:     dst,
			srcType: reflectx.IndirectType(src.Type()),
			dstType: reflectx.IndirectType(dst.Type()),
		}
		for _, fn := range fns {
			if handled, err := fn(p, o); handled {
				return err
			}
		}

		return next(fieldPath, src, dst)
	}
}

// clearNil zeroes the destination of a nil source unless WithIgnoreZero
// keeps it.
func clearNil(p pair, o *options) {
	if !o.ignoreZero {
		reflectx.SetZero(p.dst)
	}
}

// messagePtr returns a pointer to the message held by v. An addressable v is
// read in place; otherwise v is copied once.
func messagePtr[T any](v reflect.Value) *T {
	if v.CanAddr() {
		if ptr, ok := reflect.TypeAssert[*T](v.Addr()); ok {
			return ptr
		}
	}

	ptr := reflect.New(v.Type())
	ptr.Elem().Set(v)
	msg, _ := reflect.TypeAssert[*T](ptr)

	return msg
}

// messageDst allocates every nil pointer layer of dst and returns a pointer to
// the destination message, which the caller fills field by field.
func messageDst[T any](dst reflect.Value, dstType reflect.Type) *T {
	reflectx.MakeDst(&dst, dstType)
	msg, _ := reflect.TypeAssert[*T](dst.Addr())

	return msg
}

// readValue returns the value held by v without boxing it.
func readValue[T any](v reflect.Value) T {
	value, _ := reflect.TypeAssert[T](v)
	return value
}

// setValue allocates every nil pointer layer of dst and stores value in it.
func setValue[T any](dst reflect.Value, dstType reflect.Type, value T) {
	reflectx.MakeDst(&dst, dstType)

	if dst.CanAddr() {
		if ptr, ok := reflect.TypeAssert[*T](dst.Addr()); ok {
			*ptr = value
			return
		}
	}

	dst.Set(reflect.ValueOf(value))
}

// appendPadded appends v in decimal, left-padded with zeros to width digits.
func appendPadded(buf []byte, v, width int) []byte {
	var digits [maxFractionDigits]byte
	for i := width - 1; i >= 0; i-- {
		digits[i] = byte('0' + v%decimalBase)
		v /= decimalBase
	}

	return append(buf, digits[:width]...)
}

// parseDigits parses s, which must be non-empty and hold only ASCII digits.
func parseDigits(s string) (int, bool) {
	if s == "" {
		return 0, false
	}

	n := 0

	for i := range len(s) {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, false
		}

		n = n*decimalBase + int(c-'0')
	}

	return n, true
}

// parseFraction parses 1-9 fraction digits into nanoseconds.
func parseFraction(s string) (int32, bool) {
	if len(s) > maxFractionDigits {
		return 0, false
	}

	n, ok := parseDigits(s)
	if !ok {
		return 0, false
	}

	for range maxFractionDigits - len(s) {
		n *= decimalBase
	}

	return int32(n), true
}

// appendFraction appends nanos (0-999,999,999) as a fraction: at least
// minDigits digits, extended as needed to keep every non-zero digit, and no
// dot at all when that leaves no digits.
func appendFraction(buf []byte, nanos int32, minDigits int) []byte {
	var digits [maxFractionDigits]byte
	appendPadded(digits[:0], int(nanos), maxFractionDigits)

	n := maxFractionDigits
	for n > minDigits && digits[n-1] == '0' {
		n--
	}

	if n == 0 {
		return buf
	}

	buf = append(buf, '.')

	return append(buf, digits[:n]...)
}
