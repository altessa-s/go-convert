// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package convert_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	converter "github.com/altessa-s/go-convert"
	convcodec "github.com/altessa-s/go-convert/codec"
)

func TestConvertDoublePointerRoots(t *testing.T) {
	t.Parallel()

	type source struct{ Name string }
	type destination struct{ Name string }

	srcValue := &source{Name: "nested"}
	src := &srcValue
	var dstValue *destination
	dst := &dstValue
	converter.Convert(src, dst)
	require.NotNil(t, dstValue)
	require.Equal(t, "nested", dstValue.Name)

	sliceValue := []source{{Name: "first"}, {Name: "second"}}
	sliceSrcValue := &sliceValue
	sliceSrc := &sliceSrcValue
	var sliceDstValue *[]destination
	sliceDst := &sliceDstValue
	converter.Convert(sliceSrc, sliceDst)
	require.NotNil(t, sliceDstValue)
	require.Equal(t, []destination{{Name: "first"}, {Name: "second"}}, *sliceDstValue)
}

func TestConvertDoublePointerStructFields(t *testing.T) {
	t.Parallel()
	type sourceValue struct{ Name string }
	type destinationValue struct{ Name string }
	type source struct{ Value **sourceValue }
	type destination struct{ Value **destinationValue }

	value := &sourceValue{Name: "nested"}
	valuePtr := &value
	src := source{Value: valuePtr}
	var dst destination
	converter.Convert(src, &dst)
	require.NotNil(t, dst.Value)
	require.NotNil(t, *dst.Value)
	require.Equal(t, "nested", (**dst.Value).Name)

	var nilInner *sourceValue
	src.Value = &nilInner
	dstValue := &destinationValue{Name: "preserved"}
	dst.Value = &dstValue
	converter.Convert(src, &dst)
	require.Nil(t, dst.Value, "a nil inner source pointer preserves nil semantics")
}

func TestConvertNilInnerPointerRootsAreNoOps(t *testing.T) {
	t.Parallel()

	type source struct{ Name string }
	type destination struct{ Name string }

	var srcValue *source
	src := &srcValue
	dstValue := &destination{Name: "preserved"}
	dst := &dstValue
	converter.Convert(src, dst)
	require.Equal(t, "preserved", dstValue.Name)

	var sliceValue *[]source
	sliceSrc := &sliceValue
	sliceDstValue := &[]destination{{Name: "preserved"}}
	sliceDst := &sliceDstValue
	converter.Convert(sliceSrc, sliceDst)
	require.Equal(t, []destination{{Name: "preserved"}}, *sliceDstValue)
}

type sealedValue struct {
	Ciphertext []byte
}

type secretProvider struct {
	Headers sealedValue
}

type secretResponse struct {
	Headers *string
}

// A struct field converted into a pointer of an incompatible type leaves the
// destination nil instead of allocating a pointer to the zero value.
func TestConvertLeavesIncompatiblePointerNil(t *testing.T) {
	t.Parallel()

	var dst secretResponse
	converter.Convert(secretProvider{Headers: sealedValue{Ciphertext: []byte("secret")}}, &dst)

	require.Nil(t, dst.Headers)
}

func TestConvertLeavesIncompatiblePointerNilWithCodecs(t *testing.T) {
	t.Parallel()
	noop := func(fieldName string, src, dst reflect.Value, next convcodec.CodecHandler) {
		next(fieldName, src, dst)
	}
	var dst secretResponse
	converter.Convert(secretProvider{Headers: sealedValue{Ciphertext: []byte("secret")}}, &dst,
		converter.WithCodecs(noop))
	require.Nil(t, dst.Headers)
}

type sealedSliceSource struct {
	Items []sealedValue
}

type stringPointerSliceDest struct {
	Items []*string
}

func TestConvertLeavesIncompatiblePointerElementsNil(t *testing.T) {
	t.Parallel()
	src := sealedSliceSource{Items: []sealedValue{{Ciphertext: []byte("secret")}}}
	var dst stringPointerSliceDest
	converter.Convert(src, &dst)
	require.Len(t, dst.Items, 1)
	require.Nil(t, dst.Items[0])
}

func TestConvertLeavesIncompatiblePointerElementsNilWithCodecs(t *testing.T) {
	t.Parallel()
	noop := func(fieldName string, src, dst reflect.Value, next convcodec.CodecHandler) {
		next(fieldName, src, dst)
	}
	src := sealedSliceSource{Items: []sealedValue{{Ciphertext: []byte("secret")}}}
	var dst stringPointerSliceDest
	converter.Convert(src, &dst, converter.WithCodecs(noop))
	require.Len(t, dst.Items, 1)
	require.Nil(t, dst.Items[0])
}

func TestConvertNilPointerElementIntoValueSlice(t *testing.T) {
	t.Parallel()

	type a struct{ X int }
	type b struct{ X int }
	type from struct{ Items []*a }
	type to struct{ Items []b }
	var dst to
	converter.Convert(from{Items: []*a{nil, {X: 1}}}, &dst)
	require.Equal(t, []b{{}, {X: 1}}, dst.Items)
}
