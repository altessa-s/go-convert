// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package pbwrap_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-convert/proto/codec/pbwrap"

	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestNew_StringToWrapper(t *testing.T) {
	codec := pbwrap.New()

	src := reflect.ValueOf("hello")
	var dst *wrapperspb.StringValue
	dstVal := reflect.ValueOf(&dst).Elem()

	codec("Name", src, dstVal, func(_ string, _, _ reflect.Value) {
		require.Fail(t, "next should not be called")
	})

	require.NotNil(t, dst)
	assert.Equal(t, "hello", dst.Value)
}

func TestNew_WrapperToString(t *testing.T) {
	codec := pbwrap.New()

	src := reflect.ValueOf(&wrapperspb.StringValue{Value: "hello"})
	var dst string
	dstVal := reflect.ValueOf(&dst).Elem()

	codec("Name", src, dstVal, func(_ string, _, _ reflect.Value) {
		require.Fail(t, "next should not be called")
	})

	assert.Equal(t, "hello", dst)
}

func TestNew_PassThrough(t *testing.T) {
	codec := pbwrap.New()

	src := reflect.ValueOf("hello")
	var dst string
	dstVal := reflect.ValueOf(&dst).Elem()

	nextCalled := false
	codec("Name", src, dstVal, func(_ string, _, _ reflect.Value) {
		nextCalled = true
	})

	assert.True(t, nextCalled, "next should be called for non-wrapper types")
}

func TestNew_IgnoreZeroValues(t *testing.T) {
	codec := pbwrap.New(pbwrap.WithIgnoreZeroValues())

	src := reflect.ValueOf("")
	var dst *wrapperspb.StringValue
	dstVal := reflect.ValueOf(&dst).Elem()

	codec("Name", src, dstVal, func(_ string, _, _ reflect.Value) {
		require.Fail(t, "next should not be called")
	})

	assert.Nil(t, dst, "dst should remain nil when zero value is ignored")
}

func TestNew_IgnoreZeroWrappers(t *testing.T) {
	codec := pbwrap.New(pbwrap.WithIgnoreZeroWrappers())

	src := reflect.ValueOf(&wrapperspb.StringValue{Value: ""})
	var dst string
	dstVal := reflect.ValueOf(&dst).Elem()

	codec("Name", src, dstVal, func(_ string, _, _ reflect.Value) {
		require.Fail(t, "next should not be called")
	})

	assert.Equal(t, "", dst, "dst should remain zero when zero wrapper is ignored")
}

func TestNew_NilWrapperPointerToValue(t *testing.T) {
	for _, name := range []string{"default", "ignore zero values"} {
		t.Run(name, func(t *testing.T) {
			var src *wrapperspb.StringValue
			srcValue := reflect.ValueOf(src)
			dst := "preserved"
			dstValue := reflect.ValueOf(&dst).Elem()
			c := pbwrap.New()
			if name == "ignore zero values" {
				c = pbwrap.New(pbwrap.WithIgnoreZeroValues())
			}

			require.NotPanics(t, func() {
				c("Name", srcValue, dstValue, func(_ string, _, _ reflect.Value) {
					require.Fail(t, "nil wrapper should be handled")
				})
			})
			require.Equal(t, "preserved", dst)
		})
	}
}

func TestNew_IncompatibleWrapperPairsDelegate(t *testing.T) {
	tests := []struct {
		name string
		src  reflect.Value
		dst  reflect.Value
	}{
		{
			name: "string wrapper to integer",
			src:  reflect.ValueOf(wrapperspb.String("hello")),
			dst:  reflect.ValueOf(new(int)).Elem(),
		},
		{
			name: "string to integer wrapper",
			src:  reflect.ValueOf("hello"),
			dst:  reflect.ValueOf(new(*wrapperspb.Int32Value)).Elem(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled := false
			require.NotPanics(t, func() {
				pbwrap.New()("field", tt.src, tt.dst, func(_ string, _, _ reflect.Value) { nextCalled = true })
			})
			require.True(t, nextCalled)
		})
	}
}

func TestNew_DereferencesWrapperPointerLayers(t *testing.T) {
	t.Run("source", func(t *testing.T) {
		wrapper := wrapperspb.String("hello")
		src := reflect.ValueOf(&wrapper)
		var dst string
		dstValue := reflect.ValueOf(&dst).Elem()
		nextCalled := false
		pbwrap.New()("field", src, dstValue, func(_ string, _, _ reflect.Value) { nextCalled = true })
		require.False(t, nextCalled)
		require.Equal(t, "hello", dst)
	})
	t.Run("destination", func(t *testing.T) {
		src := reflect.ValueOf("hello")
		var dst *wrapperspb.StringValue
		dstValue := reflect.ValueOf(&dst).Elem()
		nextCalled := false
		pbwrap.New()("field", src, dstValue, func(_ string, _, _ reflect.Value) { nextCalled = true })
		require.False(t, nextCalled)
		require.NotNil(t, dst)
		require.Equal(t, "hello", dst.Value)
	})
}
