// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package convert_test

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-convert/codec/mapslice"

	converter "github.com/altessa-s/go-convert"
	convcodec "github.com/altessa-s/go-convert/codec"
)

func TestWithCodecsAcceptsNoCodecs(t *testing.T) {
	conv := converter.New[*SrcUser, *DstUser](converter.WithCodecs())
	require.NotNil(t, conv)
}

// A registered codec is consulted before the built-in struct-to-struct field
// copy, so a codec-handled type whose Go kind is struct is not shadowed by a
// field-by-field copy (which, for differing field names or unexported-only
// fields, would yield a zero value).
func TestCodecsInterceptStructKind(t *testing.T) {
	type legacyName struct{ Full string }
	type displayName struct{ Value string }
	type src struct {
		ID   string
		Name legacyName
	}
	type dst struct {
		ID   string
		Name displayName
	}

	legacyType := reflect.TypeFor[legacyName]()
	displayType := reflect.TypeFor[displayName]()

	// Maps legacyName -> displayName. Field names differ, so the default struct
	// copy would leave Value empty; only the codec can populate it.
	codec := func(field string, s, d reflect.Value, next convcodec.CodecHandler) {
		si := reflect.Indirect(s)
		di := reflect.Indirect(d)
		if si.IsValid() && di.IsValid() && si.Type() == legacyType && di.Type() == displayType {
			di.FieldByName("Value").SetString(si.FieldByName("Full").String())
			return
		}
		next(field, s, d)
	}

	var out dst
	converter.Convert(src{ID: "u1", Name: legacyName{Full: "Ada"}}, &out, converter.WithCodecs(codec))

	require.Equal(t, "u1", out.ID, "plain assignable field must still convert with a codec registered")
	require.Equal(t, "Ada", out.Name.Value, "codec must intercept the struct-kind field")
}

// The built-in dispatch is the codec chain's terminal handler: a codec that
// always delegates still lets the convertible-scalar fallback run (here a
// defined type to its underlying type).
func TestCodecsDelegateToConvertibleFallback(t *testing.T) {
	type celsius float64
	type src struct{ Temp celsius }
	type dst struct{ Temp float64 }

	// Never handles anything; always delegates to the rest of the chain.
	passthrough := func(field string, s, d reflect.Value, next convcodec.CodecHandler) {
		next(field, s, d)
	}

	var out dst
	converter.Convert(src{Temp: celsius(36.6)}, &out, converter.WithCodecs(passthrough))

	require.InEpsilon(t, 36.6, out.Temp, 1e-9, "convertible scalar fallback must run after a delegating codec")
}

// The codec hook fires for every element during slice and map traversal, so
// codec-handled element types are converted across the whole collection.
func TestCodecsInterceptSliceAndMapElements(t *testing.T) {
	type legacyName struct{ Full string }
	type displayName struct{ Value string }
	type src struct {
		Names    []legacyName
		ByRegion map[string]legacyName
	}
	type dst struct {
		Names    []displayName
		ByRegion map[string]displayName
	}

	legacyType := reflect.TypeFor[legacyName]()
	displayType := reflect.TypeFor[displayName]()

	codec := func(field string, s, d reflect.Value, next convcodec.CodecHandler) {
		si := reflect.Indirect(s)
		di := reflect.Indirect(d)
		if si.IsValid() && di.IsValid() && si.Type() == legacyType && di.Type() == displayType {
			di.FieldByName("Value").SetString(si.FieldByName("Full").String())
			return
		}
		next(field, s, d)
	}

	in := src{
		Names: []legacyName{{Full: "Ada"}, {Full: "Bob"}},
		ByRegion: map[string]legacyName{
			"eu": {Full: "Carol"},
			"us": {Full: "Dan"},
		},
	}
	var out dst
	converter.Convert(in, &out, converter.WithCodecs(codec))

	require.Len(t, out.Names, 2)
	require.Equal(t, "Ada", out.Names[0].Value)
	require.Equal(t, "Bob", out.Names[1].Value)
	require.Equal(t, "Carol", out.ByRegion["eu"].Value)
	require.Equal(t, "Dan", out.ByRegion["us"].Value)
}

// Codecs see codec-handled fields nested inside other structs, not just the
// outermost ones.
func TestCodecsInterceptNestedStructField(t *testing.T) {
	type legacyName struct{ Full string }
	type displayName struct{ Value string }
	type srcInner struct{ Name legacyName }
	type dstInner struct{ Name displayName }
	type src struct{ Inner srcInner }
	type dst struct{ Inner dstInner }

	legacyType := reflect.TypeFor[legacyName]()
	displayType := reflect.TypeFor[displayName]()

	codec := func(field string, s, d reflect.Value, next convcodec.CodecHandler) {
		si := reflect.Indirect(s)
		di := reflect.Indirect(d)
		if si.IsValid() && di.IsValid() && si.Type() == legacyType && di.Type() == displayType {
			di.FieldByName("Value").SetString(si.FieldByName("Full").String())
			return
		}
		next(field, s, d)
	}

	var out dst
	converter.Convert(src{Inner: srcInner{Name: legacyName{Full: "Ada"}}}, &out, converter.WithCodecs(codec))
	require.Equal(t, "Ada", out.Inner.Name.Value, "codec must intercept the codec-handled struct field even when wrapped in another struct")
}

func TestMapsliceValuesFromPointerMap(t *testing.T) {
	t.Parallel()
	values := map[string]int{"a": 1}
	type source struct{ Values *map[string]int }
	type destination struct{ Values []int }
	var dst destination
	converter.Convert(source{Values: &values}, &dst, converter.WithCodecs(mapslice.Values))
	require.Equal(t, []int{1}, dst.Values)
}

func TestMixedCodecsKeepLegacyLeafName(t *testing.T) {
	t.Parallel()
	type sourceItem struct{ Grade int }
	type targetItem struct{ Grade string }
	src := struct{ Attrs sourceItem }{sourceItem{1}}
	var dst struct{ Attrs targetItem }
	var seenPath, seenName string
	errorCodec := func(path string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
		if src.Kind() == reflect.Int {
			seenPath = path
		}
		return next(path, src, dst)
	}
	legacyCodec := func(name string, src, dst reflect.Value, next convcodec.CodecHandler) {
		if src.Kind() == reflect.Int {
			seenName = name
			if name == "grade" {
				dst.SetString("junior")
				return
			}
		}
		next(name, src, dst)
	}
	converter.Convert(src, &dst, converter.WithErrorCodecs(errorCodec), converter.WithCodecs(legacyCodec))
	require.Equal(t, "attrs.grade", seenPath)
	require.Equal(t, "grade", seenName)
	require.Equal(t, "junior", dst.Attrs.Grade)
}

func TestErrorCodecsSkipMapKeys(t *testing.T) {
	var paths []string
	codec := func(path string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
		paths = append(paths, path)
		return next(path, src, dst)
	}
	src := struct{ Data map[bool]int }{map[bool]int{true: 3}}
	var dst struct{ Data map[string]int }
	converter.Convert(src, &dst, converter.WithErrorCodecs(codec))
	require.NotEmpty(t, paths)
	for _, path := range paths {
		require.NotEqual(t, "\x00key", path)
	}
	require.Equal(t, map[string]int{"": 3}, dst.Data)
}

func TestCodecsSkipStructMapKeyFields(t *testing.T) {
	type protoGrade int32
	type domainGrade string
	type sourceKey struct{ Grade protoGrade }
	type targetKey struct{ Grade domainGrade }
	for _, ignored := range []bool{false, true} {
		t.Run(fmt.Sprint(ignored), func(t *testing.T) {
			var paths []string
			errorCodec := func(path string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
				paths = append(paths, path)
				if src.Type() == reflect.TypeFor[protoGrade]() {
					t.Errorf("map key reached error codec at %q", path)
				}
				return next(path, src, dst)
			}
			legacy := func(name string, src, dst reflect.Value, next convcodec.CodecHandler) {
				if src.Type() == reflect.TypeFor[protoGrade]() {
					dst.SetString("known")
					return
				}
				next(name, src, dst)
			}
			opts := []converter.Option{converter.WithErrorCodecs(errorCodec), converter.WithCodecs(legacy)}
			if ignored {
				opts = append(opts, converter.WithIgnorePaths("unused"))
			}
			src := struct{ Data map[sourceKey]int }{map[sourceKey]int{{1}: 42}}
			var dst struct{ Data map[targetKey]int }
			converter.Convert(src, &dst, opts...)
			for _, path := range paths {
				if strings.Contains(path, "\x00key") {
					t.Errorf("sentinel leaked: %q", path)
				}
			}
			if dst.Data[targetKey{"known"}] != 42 {
				t.Fatalf("converted map = %v", dst.Data)
			}
		})
	}
}

func TestConvertECodecErrorIdentity(t *testing.T) {
	sentinel := fmt.Errorf("codec failure")
	codec := func(_ string, _, _ reflect.Value, _ convcodec.CodecHandlerE) error { return sentinel }
	src, dst := struct{ Data int }{1}, struct{ Data int }{}
	_, err := converter.ConvertE(src, &dst, converter.WithErrorCodecs(codec))
	require.Same(t, sentinel, err)
	_, isTypeError := err.(converter.UnsupportedTypeError)
	require.False(t, isTypeError)
}

func TestConvertEReturnsWrappedFieldErrorUnchanged(t *testing.T) {
	t.Parallel()
	fieldErr := newFieldError("attrs.grade", "unknown value")
	wrapped := fmt.Errorf("enum conversion: %w", fieldErr)
	codec := func(_ string, _, _ reflect.Value, _ convcodec.CodecHandlerE) error {
		return wrapped
	}
	src, dst := struct{ Grade int }{1}, struct{ Grade int }{}
	_, err := converter.ConvertE(src, &dst, converter.WithErrorCodecs(codec))
	require.Same(t, wrapped, err)
}

func TestConvertEWrapsLegacyCodecPanic(t *testing.T) {
	original := fmt.Errorf("legacy: %w", newFieldError("grade", "unknown"))
	codec := func(_ string, _, _ reflect.Value, _ convcodec.CodecHandler) { panic(original) }
	var dst struct{ Grade string }
	_, err := converter.ConvertE(struct{ Grade int }{1}, &dst, converter.WithCodecs(codec))
	typeErr, ok := err.(converter.UnsupportedTypeError)
	if !ok || typeErr.Err != original {
		t.Fatalf("got %T: %v, want UnsupportedTypeError wrapping original", err, err)
	}
}

func TestConvertSeqPropagatesCodecErrorIdentity(t *testing.T) {
	sentinel := errors.New("codec failure")
	fieldErr := newFieldError("grade", "unknown")
	for _, original := range []error{sentinel, fieldErr} {
		t.Run(original.Error(), func(t *testing.T) {
			defer func() {
				recovered := recover()
				err, ok := recovered.(error)
				if !ok || !errors.Is(err, original) {
					t.Fatalf("recovered %v does not wrap original %v", recovered, original)
				}
				if original == fieldErr {
					var got *fieldError
					if !errors.As(err, &got) || got != fieldErr {
						t.Fatalf("FieldError identity lost: %v", err)
					}
				}
			}()
			codec := func(_ string, _, _ reflect.Value, _ convcodec.CodecHandlerE) error { return original }
			conv := converter.New[int, string](converter.WithErrorCodecs(codec))
			for range conv.ConvertSeq([]int{1}) {
			}
			t.Fatal("expected codec panic")
		})
	}
}

// fieldError is a pointer error type, as used by validation codecs, whose
// identity must survive conversion.
type fieldError struct{ field, reason string }

func newFieldError(field, reason string) *fieldError {
	return &fieldError{field: field, reason: reason}
}

func (e *fieldError) Error() string { return e.field + ": " + e.reason }
