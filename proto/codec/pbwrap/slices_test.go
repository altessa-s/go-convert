// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// LICENSE file.

package pbwrap_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-convert/proto/codec/pbwrap"

	"google.golang.org/protobuf/types/known/wrapperspb"

	converter "github.com/altessa-s/go-convert"
)

type wrapperSliceSource struct{ Items []*wrapperspb.StringValue }
type wrapperSliceDestination struct{ Items []*string }
type wrapperValueSliceDestination struct{ Items []string }

func TestConvert_CodecPreservesExplicitZeroPointer(t *testing.T) {
	t.Parallel()

	src := wrapperSliceSource{Items: []*wrapperspb.StringValue{wrapperspb.String("")}}

	var dst wrapperSliceDestination
	converter.Convert(src, &dst, converter.WithCodecs(pbwrap.New()))
	require.Len(t, dst.Items, 1)
	require.NotNil(t, dst.Items[0])
	require.Equal(t, "", *dst.Items[0])
}

func TestConvert_NilWrapperPointerInValueSlice(t *testing.T) {
	t.Parallel()

	src := wrapperSliceSource{Items: []*wrapperspb.StringValue{nil, wrapperspb.String("value")}}

	var dst wrapperValueSliceDestination

	converter.Convert(src, &dst, converter.WithCodecs(pbwrap.New(), pbwrap.New(pbwrap.WithIgnoreZeroValues())))

	require.Equal(t, []string{"", "value"}, dst.Items)
}
