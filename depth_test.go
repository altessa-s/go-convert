// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package convert_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	converter "github.com/altessa-s/go-convert"
)

type listNode struct {
	Value int
	Next  *listNode
}

type listNodeDTO struct {
	Value int
	Next  *listNodeDTO
}

type anyNode struct{ Child any }

func chain(n int) *listNode {
	var head *listNode
	for i := range n {
		head = &listNode{Value: i, Next: head}
	}
	return head
}

func TestCyclicGraphFailsWithDepthError(t *testing.T) {
	t.Parallel()

	node := &listNode{Value: 1}
	node.Next = node

	var dst listNodeDTO
	_, err := converter.ConvertE(node, &dst)

	var depthErr converter.DepthError
	require.ErrorAs(t, err, &depthErr)
	require.Equal(t, converter.DefaultMaxDepth, depthErr.Limit)
}

func TestCyclicGraphPanicsWithDepthError(t *testing.T) {
	t.Parallel()

	node := &listNode{Value: 1}
	node.Next = node

	require.PanicsWithError(t, converter.DepthError{Limit: 8}.Error(), func() {
		converter.Convert(node, &listNodeDTO{}, converter.WithMaxDepth(8))
	})
}

func TestMaxDepthAllowsNestingUpToTheLimit(t *testing.T) {
	t.Parallel()

	var dst listNodeDTO
	_, err := converter.ConvertE(chain(10), &dst, converter.WithMaxDepth(64))
	require.NoError(t, err)

	got := 0
	for n := &dst; n != nil; n = n.Next {
		got++
	}
	require.Equal(t, 10, got)

	_, err = converter.ConvertE(chain(100), &listNodeDTO{}, converter.WithMaxDepth(64))
	require.ErrorAs(t, err, new(converter.DepthError))
}

func TestNonPositiveMaxDepthUsesDefault(t *testing.T) {
	t.Parallel()

	_, err := converter.ConvertE(chain(50), &listNodeDTO{}, converter.WithMaxDepth(0))
	require.NoError(t, err)
}

func TestDynamicCycleWithIgnorePathTerminates(t *testing.T) {
	t.Parallel()

	inner := map[string]any{}
	inner["self"] = anyNode{Child: inner}
	src := anyNode{Child: inner}

	_, err := converter.ConvertE(src, &anyNode{}, converter.WithIgnorePaths("child.secret"), converter.WithMaxDepth(32))
	if err != nil {
		require.ErrorAs(t, err, new(converter.DepthError))
	}
}

func TestUnsupportedTypeErrorUnwrapsCause(t *testing.T) {
	t.Parallel()

	cause := errors.New("boom")
	err := converter.UnsupportedTypeError{Source: 1, Destination: "", Reason: cause.Error(), Err: cause}

	require.ErrorIs(t, err, cause)
	require.Contains(t, err.Error(), "from int to string")
	require.NoError(t, converter.UnsupportedTypeError{}.Unwrap())
}

func TestConvertEReportsUnsupportedShapes(t *testing.T) {
	t.Parallel()

	type item struct{ Name string }

	tests := []struct {
		name string
		run  func() error
	}{
		{"scalar source", func() error { _, err := converter.ConvertE(42, new(int)); return err }},
		{"non-pointer destination", func() error { _, err := converter.ConvertE(item{}, item{}); return err }},
		{"nil destination", func() error { _, err := converter.ConvertE(item{}, (*item)(nil)); return err }},
		{"kind mismatch", func() error { _, err := converter.ConvertE(item{}, &[]item{}); return err }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.ErrorAs(t, tc.run(), new(converter.UnsupportedTypeError))
		})
	}
}

func TestOverflowErrorNamesTheField(t *testing.T) {
	t.Parallel()

	type wide struct{ Count int64 }
	type narrow struct{ Count int8 }

	_, err := converter.ConvertE(wide{Count: 1 << 10}, &narrow{}, converter.WithOverflowCheck())

	var overflow converter.OverflowError
	require.ErrorAs(t, err, &overflow)
	require.Equal(t, "count", overflow.Field)
	require.Contains(t, overflow.Error(), `field "count"`)
}
