// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package textcodec

import (
	"encoding"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"

	convcodec "github.com/altessa-s/go-convert/codec"
	reflectutils "github.com/altessa-s/go-convert/internal/reflect"
)

var (
	marshalerType   = reflect.TypeFor[encoding.TextMarshaler]()
	unmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()
)

// maxClaims bounds the number of cached type pairs. Pairs beyond the bound,
// as produced by programs that synthesize types at run time, are decided on
// every call instead of being cached.
const maxClaims = 4096

// claims caches the claim decision of up to [maxClaims] source and
// destination type pairs, so the Implements checks run once per pair rather
// than per field: typePair -> *claim.
var (
	claims     sync.Map
	claimCount atomic.Int64
)

// typePair is a source and destination type, the key of [claims].
type typePair struct{ src, dst reflect.Type }

// claim is the cached decision for a type pair.
type claim struct {
	// srcType and dstType are the pair with pointers removed.
	srcType, dstType reflect.Type
	// handled reports whether the codec converts the pair.
	handled bool
	// srcMarshaler reports whether srcType or *srcType implements
	// encoding.TextMarshaler.
	srcMarshaler bool
	// dstUnmarshaler reports whether *dstType implements
	// encoding.TextUnmarshaler.
	dstUnmarshaler bool
}

// Error reports a failed MarshalText or UnmarshalText call.
type Error struct {
	// Path is the full source field path; it is empty at the root.
	Path string
	// Type is the type whose MarshalText or UnmarshalText failed.
	Type reflect.Type
	// Err is the error returned by MarshalText or UnmarshalText.
	Err error
}

// Error reports the field path, the type and the cause.
func (e *Error) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("textcodec: %s: %v", e.Type, e.Err)
	}

	return fmt.Sprintf("textcodec: field %q: %s: %v", e.Path, e.Type, e.Err)
}

// Unwrap returns the error from MarshalText or UnmarshalText.
func (e *Error) Unwrap() error { return e.Err }

// New creates an error codec converting through encoding.TextMarshaler and
// encoding.TextUnmarshaler.
//
// Supported conversions (pointer variants on either side included):
//   - TextMarshaler -> string or []byte kind: calls MarshalText
//   - string or []byte kind -> TextUnmarshaler: calls UnmarshalText
//   - TextMarshaler -> TextUnmarshaler of another type: MarshalText, then UnmarshalText
//
// A method on either the value or the pointer receiver qualifies. Identical
// source and destination types, and interface types, are passed to next.
// Failures are returned as an [*Error]. Empty text sets the destination to its
// zero value unless [WithParseEmpty] is set, and so does a nil source pointer.
// MarshalText with a pointer receiver is called on a copy of the source, so
// it cannot modify the source value.
//
// Example:
//
//	codec := textcodec.New()
//	codec := textcodec.New(textcodec.WithParseEmpty())
func New(opts ...Option) convcodec.CodecE {
	o := newOptions(opts...)

	return func(fieldPath string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
		if handled, err := tryConvert(fieldPath, src, dst, o); handled {
			return err
		}

		return next(fieldPath, src, dst)
	}
}

// tryConvert converts src into dst when one side is a text type. It reports
// whether the conversion was handled; a nil source sets dst to its zero
// value.
func tryConvert(fieldPath string, src, dst reflect.Value, opts *options) (bool, error) {
	c := claimOf(src.Type(), dst.Type())
	if !c.handled {
		return false, nil
	}

	srcType, dstType := c.srcType, c.dstType
	srcMarshaler, dstUnmarshaler := c.srcMarshaler, c.dstUnmarshaler

	srcValue := reflectutils.IndirectValue(src)
	if !srcValue.IsValid() {
		dst.SetZero()
		return true, nil
	}

	if !srcValue.CanInterface() {
		return false, nil
	}

	var text []byte

	if srcMarshaler {
		var err error
		if text, err = marshal(srcValue); err != nil {
			return true, &Error{Path: fieldPath, Type: srcType, Err: err}
		}
	} else {
		text = readText(srcValue)
	}

	if len(text) == 0 && (!dstUnmarshaler || !opts.parseEmpty) {
		dst.SetZero()
		return true, nil
	}

	if !dstUnmarshaler {
		reflectutils.MakeDst(&dst, dstType)
		writeText(dst, text)

		return true, nil
	}

	value, err := unmarshal(dstType, text)
	if err != nil {
		return true, &Error{Path: fieldPath, Type: dstType, Err: err}
	}

	reflectutils.MakeDst(&dst, dstType)
	dst.Set(value)

	return true, nil
}

// claimOf returns the cached claim decision for a source and destination
// type, computing it on first use.
func claimOf(src, dst reflect.Type) *claim {
	key := typePair{src: src, dst: dst}
	if cached, ok := claims.Load(key); ok {
		c, _ := cached.(*claim)
		return c
	}

	c := &claim{srcType: reflectutils.IndirectType(src), dstType: reflectutils.IndirectType(dst)}
	if c.srcType != c.dstType && c.srcType.Kind() != reflect.Interface && c.dstType.Kind() != reflect.Interface {
		c.dstUnmarshaler = reflect.PointerTo(c.dstType).Implements(unmarshalerType)
		c.srcMarshaler = isMarshaler(c.srcType)
		c.handled = c.srcMarshaler && (c.dstUnmarshaler || isText(c.dstType)) ||
			c.dstUnmarshaler && isText(c.srcType)
	}

	if reserveClaim() {
		if _, loaded := claims.LoadOrStore(key, c); loaded {
			claimCount.Add(-1)
		}
	}

	return c
}

// reserveClaim takes one of the [maxClaims] cache slots, reporting false when
// none is left. The compare-and-swap keeps concurrent callers within the
// bound.
func reserveClaim() bool {
	for {
		n := claimCount.Load()
		if n >= maxClaims {
			return false
		}

		if claimCount.CompareAndSwap(n, n+1) {
			return true
		}
	}
}

// isMarshaler reports whether t or *t implements encoding.TextMarshaler.
func isMarshaler(t reflect.Type) bool {
	return t.Implements(marshalerType) || reflect.PointerTo(t).Implements(marshalerType)
}

// isText reports whether t is a string or a byte slice kind.
func isText(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.String:
		return true
	case reflect.Slice:
		return t.Elem().Kind() == reflect.Uint8
	default:
		return false
	}
}

// marshal calls MarshalText on v. When only the pointer type implements
// encoding.TextMarshaler, it is called on a pointer to a copy of v, never on
// v itself, so a method that caches into its receiver cannot modify the
// source.
func marshal(v reflect.Value) ([]byte, error) {
	if !v.Type().Implements(marshalerType) {
		ptr := reflect.New(v.Type())
		ptr.Elem().Set(v)
		v = ptr
	}

	m, ok := reflect.TypeAssert[encoding.TextMarshaler](v)
	if !ok {
		return nil, fmt.Errorf("%s does not implement encoding.TextMarshaler", v.Type())
	}

	return m.MarshalText()
}

// unmarshal returns a new value of type t holding text decoded by its
// UnmarshalText method. A fresh value keeps the destination unchanged when
// decoding fails.
func unmarshal(t reflect.Type, text []byte) (reflect.Value, error) {
	ptr := reflect.New(t)

	u, ok := reflect.TypeAssert[encoding.TextUnmarshaler](ptr)
	if !ok {
		return reflect.Value{}, fmt.Errorf("%s does not implement encoding.TextUnmarshaler", ptr.Type())
	}

	if err := u.UnmarshalText(text); err != nil {
		return reflect.Value{}, err
	}

	return ptr.Elem(), nil
}

// readText returns the text held by a string or byte slice value.
func readText(v reflect.Value) []byte {
	if v.Kind() == reflect.String {
		return []byte(v.String())
	}

	return v.Bytes()
}

// writeText stores text into a string or byte slice value.
func writeText(dst reflect.Value, text []byte) {
	if dst.Kind() == reflect.String {
		dst.SetString(string(text))
		return
	}

	dst.SetBytes(text)
}
