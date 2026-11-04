// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package convert_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	converter "github.com/altessa-s/go-convert"
	convcodec "github.com/altessa-s/go-convert/codec"
)

func TestIgnorePathsDynamicValues(t *testing.T) {
	t.Parallel()

	type root struct{ Data any }

	type object struct {
		Secret string
		Name   string
	}

	var typedNil *object
	for _, tc := range []struct {
		name           string
		src, dst, want any
	}{
		{"struct", object{"secret", "new"}, object{"keep", "old"}, object{"keep", "new"}},
		{"pointer", &object{"secret", "new"}, &object{"keep", "old"}, &object{"keep", "new"}},
		{"map", map[string]any{"secret": "secret", "name": "new"}, map[string]any{"secret": "keep"}, map[string]any{"secret": "keep", "name": "new"}},
		{"map absent", map[string]any{"secret": "secret", "name": "new"}, nil, map[string]any{"name": "new"}},
		{"slice", []object{{"secret", "new"}}, nil, []object{{"", "new"}}},
		{"array", [1]object{{"secret", "new"}}, nil, [1]object{{"", "new"}}},
		{"scalar", 42, 1, 42},
		{"nil", nil, object{}, nil},
		{"typed nil", typedNil, nil, typedNil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			src, dst := root{tc.src}, root{tc.dst}
			converter.Convert(src, &dst, converter.WithIgnorePaths("data.secret"))
			require.Equal(t, tc.want, dst.Data)

			if tc.name == "typed nil" {
				require.NotNil(t, reflect.TypeOf(dst.Data))
				require.True(t, reflect.ValueOf(dst.Data).IsNil())
			}

			if p, ok := src.Data.(*object); ok && p != nil {
				require.Equal(t, "secret", p.Secret)
			}

			if m, ok := src.Data.(map[string]any); ok {
				require.Equal(t, "secret", m["secret"])
			}
		})
	}
}

func TestDynamicStructAndMapApplySameOptions(t *testing.T) {
	type object struct{ Secret, Alias, Name, Skip string }

	type root struct{ Data any }

	for _, data := range []any{object{Secret: "hidden", Name: "new", Skip: "ignored"}, map[string]any{"Secret": "hidden", "Name": "new", "Skip": "ignored"}} {
		var paths []string

		dst := root{}

		switch data.(type) {
		case object:
			dst.Data = object{Secret: "keep", Skip: "old"}
		default:
			dst.Data = map[string]any{"Secret": "keep", "Missing": "keep missing", "Skip": "old"}
		}

		_, err := converter.ConvertE(root{data}, &dst,
			converter.WithIgnorePaths("data.secret", "data.missing"),
			converter.WithIgnoreFields("skip"),
			converter.WithFieldMappings(map[string]string{"name": "alias"}),
			converter.WithErrorCodecs(func(p string, s, d reflect.Value, next convcodec.CodecHandlerE) error {
				paths = append(paths, p)
				return next(p, s, d)
			}))
		if err != nil {
			t.Fatal(err)
		}

		switch got := dst.Data.(type) {
		case object:
			if got != (object{Secret: "keep", Alias: "new", Skip: "old"}) {
				t.Fatalf("struct = %+v", got)
			}
		case map[string]any:
			want := map[string]any{"Secret": "keep", "Missing": "keep missing", "Name": "new", "Skip": "old"}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("map = %+v", got)
			}
		}

		found := false

		for _, p := range paths {
			if p == "data.name" {
				found = true
			}

			if p == "data.secret" || p == "data.skip" {
				t.Fatalf("ignored path reached codec: %q", p)
			}
		}

		if !found {
			t.Fatalf("source name path absent: %v", paths)
		}
	}
}

func TestDynamicSliceMatchesStaticSlice(t *testing.T) {
	type item struct{ Secret, Name string }

	src := []item{{"source", "new"}}
	old := []item{{"keep", "old"}}
	static := struct{ Data []item }{old}
	dynamic := struct{ Data any }{old}
	opts := []converter.Option{converter.WithIgnorePaths("data.secret")}
	converter.Convert(struct{ Data []item }{src}, &static, opts...)
	converter.Convert(struct{ Data any }{src}, &dynamic, opts...)

	if static.Data[0].Secret != "" || !reflect.DeepEqual(dynamic.Data, static.Data) {
		t.Fatalf("dynamic = %v, static = %v", dynamic.Data, static.Data)
	}

	array := struct{ Data any }{[1]item{old[0]}}
	converter.Convert(struct{ Data any }{[1]item{src[0]}}, &array, opts...)

	if want := ([1]item{{"keep", "new"}}); array.Data != want {
		t.Fatalf("array = %v, want %v", array.Data, want)
	}
}

func TestDynamicStructKeepsDestinationPointerForNilSource(t *testing.T) {
	t.Parallel()

	type inner struct {
		P    *int
		Name string
	}

	type wrapper struct{ Data any }

	src := wrapper{Data: inner{Name: "new"}}
	dst := wrapper{Data: inner{P: new(5), Name: "old"}}
	converter.Convert(src, &dst, converter.WithIgnorePaths("data.name"))

	got, ok := dst.Data.(inner)
	require.True(t, ok)
	require.NotNil(t, got.P)
	require.Equal(t, 5, *got.P)
	require.Equal(t, "old", got.Name)
}

func TestDynamicMapEntryIgnoredAfterCodecRename(t *testing.T) {
	t.Parallel()

	type wrapper struct{ Data any }

	rename := func(name string, src, dst reflect.Value, next convcodec.CodecHandler) {
		if name == "a" {
			name = "secret"
		}

		next(name, src, dst)
	}
	src := wrapper{Data: map[string]any{"a": 1, "b": 2}}

	var dst wrapper
	converter.Convert(src, &dst,
		converter.WithIgnorePaths("data.b"),
		converter.WithIgnoreFields("secret"),
		converter.WithCodecs(rename),
	)
	require.Equal(t, map[string]any{"a": nil}, dst.Data)
}

func TestDynamicMapEntryIgnoredAfterCodecPathRedirect(t *testing.T) {
	t.Parallel()

	type wrapper struct{ Data any }

	redirect := func(path string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
		if path == "data.a" {
			path = "data.b"
		}

		return next(path, src, dst)
	}
	src := wrapper{Data: map[string]any{"a": 1, "b": 2}}

	var dst wrapper
	converter.Convert(src, &dst,
		converter.WithIgnorePaths("data.b"),
		converter.WithErrorCodecs(redirect),
	)
	require.Equal(t, map[string]any{"a": nil}, dst.Data)
}

func TestDynamicMapKeyMatchingInternalSentinel(t *testing.T) {
	const key = "\x00key"

	for _, tc := range []struct {
		name   string
		value  any
		legacy bool
		want   any
	}{
		{name: "scalar", value: "source", want: "source"},
		{name: "nested", value: map[string]any{"value": "source"}, want: map[string]any{"value": "source"}},
		{name: "legacy", value: "source", legacy: true, want: "converted"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var paths []string

			errorCodec := func(path string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
				paths = append(paths, path)
				return next(path, src, dst)
			}

			opts := []converter.Option{converter.WithErrorCodecs(errorCodec), converter.WithIgnorePaths("data.unused")}
			if tc.legacy {
				opts = append(opts, converter.WithCodecs(func(name string, src, dst reflect.Value, next convcodec.CodecHandler) {
					if name == key {
						dst.Set(reflect.ValueOf("converted"))
						return
					}

					next(name, src, dst)
				}))
			}

			var dst any
			for value := range converter.ConvertSeq[any]([]any{map[string]any{key: tc.value}}, opts...) {
				dst = value
			}

			if len(paths) == 0 {
				t.Fatal("error codec was never called")
			}

			for _, path := range paths {
				if path == key {
					t.Errorf("sentinel leaked: %q", path)
				}
			}

			if want := (map[string]any{key: tc.want}); !reflect.DeepEqual(dst, want) {
				t.Fatalf("converted map = %#v, want %#v", dst, want)
			}
		})
	}
}

func TestIgnorePathsDynamicCycles(t *testing.T) {
	t.Parallel()

	type node struct {
		Next   any
		Secret string
		Name   string
	}

	p := &node{Secret: "secret", Name: "new"}
	p.Next = p
	m := map[string]any{"secret": "secret"}
	m["next"] = m
	s := make([]any, 1)

	s[0] = s
	for _, value := range []any{p, m, s} {
		src := struct{ Data any }{value}

		var dst struct{ Data any }

		require.NotPanics(t, func() {
			converter.Convert(src, &dst, converter.WithIgnorePaths("data.next.next.secret", "data.secret"))
		})
		require.NotNil(t, dst.Data)
	}

	require.Equal(t, "secret", p.Secret)
	require.Equal(t, "secret", m["secret"])
}

func TestIgnorePathsDynamicCycleFiltersBackEdge(t *testing.T) {
	srcMap := map[string]any{"secret": "source"}
	srcMap["next"] = srcMap

	for _, existing := range []bool{false, true} {
		t.Run(fmt.Sprint(existing), func(t *testing.T) {
			var dst struct{ Data any }

			old := map[string]any{"name": "keep"}
			if existing {
				dst.Data = map[string]any{"next": old}
			}

			converter.Convert(struct{ Data any }{srcMap}, &dst, converter.WithIgnorePaths("data.next.secret"))

			back := dst.Data.(map[string]any)["next"].(map[string]any)
			if _, found := back["secret"]; found {
				t.Fatal("ignored secret leaked through back edge")
			}

			if reflect.ValueOf(back).Pointer() == reflect.ValueOf(srcMap).Pointer() {
				t.Fatal("back edge aliases source")
			}

			if existing && reflect.ValueOf(back).Pointer() != reflect.ValueOf(old).Pointer() {
				t.Fatal("existing back edge changed")
			}
		})
	}

	var dst struct{ Data any }
	converter.Convert(struct{ Data any }{srcMap}, &dst, converter.WithIgnorePaths("data.secret"))

	back := dst.Data.(map[string]any)["next"].(map[string]any)
	if reflect.ValueOf(back).Pointer() != reflect.ValueOf(srcMap).Pointer() {
		t.Fatal("unfiltered back edge must retain reference")
	}
}

func TestDynamicIgnoreZeroPointer(t *testing.T) {
	type root struct{ Data any }

	zero, old := 0, 7
	src, dst := root{&zero}, root{&old}
	converter.Convert(src, &dst, converter.WithValueMode(converter.IgnoreZeroValues), converter.WithIgnorePaths("data.secret"))
	require.Equal(t, 0, *dst.Data.(*int))
}

func TestDynamicSparseMergeEmptyPointer(t *testing.T) {
	type object struct {
		Secret *string
		Name   *string
	}

	type root struct{ Data any }

	name := "old"
	old := &object{Name: &name}
	src, dst := root{&object{}}, root{old}
	converter.Convert(src, &dst, converter.WithValueMode(converter.SparseMerge), converter.WithIgnorePaths("data.secret"))
	got, ok := dst.Data.(*object)
	require.True(t, ok, "interface must retain its pointer type")
	require.Nil(t, got)
	require.False(t, dst.Data == nil, "cleared pointer must remain a typed nil")
	require.Equal(t, "old", *old.Name)
}

func TestDynamicNilInterfaceConcreteDestination(t *testing.T) {
	src := struct{ Data any }{}
	dst := struct{ Data int }{7}
	converter.Convert(src, &dst, converter.WithIgnorePaths("data.secret"))
	require.Equal(t, 7, dst.Data)
}

func TestDynamicNonEmptyInterface(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source any
		want   fmt.Stringer
	}{
		{"incompatible", 42, stringerValue("old")},
		{"assignable", stringerValue("new"), stringerValue("new")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := struct{ Data any }{tc.source}
			dst := struct{ Data fmt.Stringer }{stringerValue("old")}

			require.NotPanics(t, func() {
				converter.Convert(src, &dst, converter.WithIgnorePaths("data.secret"))
			})
			require.Equal(t, tc.want, dst.Data)
		})
	}
}

type stringerValue string

func (s stringerValue) String() string { return string(s) }

func TestDynamicLegacyCodecRoot(t *testing.T) {
	for _, handle := range []bool{false, true} {
		t.Run(fmt.Sprintf("handle=%v", handle), func(t *testing.T) {
			calls := 0
			codec := func(name string, src, dst reflect.Value, next convcodec.CodecHandler) {
				if name == "data" {
					calls++

					if handle {
						dst.Set(reflect.ValueOf("handled"))
						return
					}
				}

				next(name, src, dst)
			}
			src := struct{ Data any }{42}

			var dst struct{ Data any }
			converter.Convert(src, &dst, converter.WithCodecs(codec), converter.WithIgnorePaths("data.secret"))
			require.Equal(t, 1, calls)

			if handle {
				require.Equal(t, "handled", dst.Data)
			} else {
				require.Equal(t, 42, dst.Data)
			}
		})
	}
}

func TestDynamicNestedCodecs(t *testing.T) {
	type object struct{ Secret, Name string }

	for _, tc := range []struct {
		name  string
		value any
		path  string
		want  any
	}{
		{"struct", struct{ Child any }{object{"secret", "new"}}, "data.child.secret", struct{ Child any }{object{"", "new"}}},
		{"map", map[string]any{"child": object{"secret", "new"}}, "data.child.secret", map[string]any{"child": object{"", "new"}}},
		{"slice", []any{object{"secret", "new"}}, "data.secret", []any{object{"", "new"}}},
		{"array", [1]any{object{"secret", "new"}}, "data.secret", [1]any{object{"", "new"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, handle := range []bool{false, true} {
				calls := 0
				codec := func(name string, src, dst reflect.Value, next convcodec.CodecHandler) {
					if src.Kind() == reflect.Interface {
						calls++

						if handle && src.Elem().Type() == reflect.TypeFor[object]() {
							dst.Set(reflect.ValueOf(object{Name: "handled"}))
							return
						}
					}

					next(name, src, dst)
				}
				src := struct{ Data any }{tc.value}

				var dst struct{ Data any }
				converter.Convert(src, &dst, converter.WithCodecs(codec), converter.WithIgnorePaths(tc.path))
				require.Equal(t, 2, calls)

				if !handle {
					require.Equal(t, tc.want, dst.Data)
				} else {
					require.Contains(t, fmt.Sprint(dst.Data), "handled")
				}
			}
		})
	}
}

func TestDynamicNestedErrorCodec(t *testing.T) {
	for _, failure := range []error{fmt.Errorf("nested codec failure"), newFieldError("data.child", "invalid")} {
		var paths []string

		codec := func(path string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
			paths = append(paths, path)
			if path == "data.child" && src.Kind() == reflect.Interface {
				return failure
			}

			return next(path, src, dst)
		}
		src := struct{ Data any }{struct{ Child any }{42}}

		var dst struct{ Data any }

		_, err := converter.ConvertE(src, &dst, converter.WithErrorCodecs(codec), converter.WithIgnorePaths("data.child.secret"))
		require.True(t, err == failure, "got %v, want original error %v", err, failure)
		require.Equal(t, []string{"data", "data.child"}, paths)
	}
}
