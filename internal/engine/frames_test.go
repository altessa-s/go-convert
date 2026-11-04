// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package engine

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	convcodec "github.com/altessa-s/go-convert/codec"
)

type frameInner struct{ Y int32 }

type frameInnerDst struct{ Y int64 }

type frameSrc struct {
	A  int32
	In frameInner
}

type frameDst struct {
	A  int64
	In frameInnerDst
}

func TestErrorFramesSurviveRepeatedAndRecoveredCalls(t *testing.T) {
	t.Parallel()

	var names []string

	record := func(name string, src, dst reflect.Value, next convcodec.CodecHandler) {
		names = append(names, name)
		next(name, src, dst)
	}
	// The codec runs its continuation twice, the first time recovering a
	// panic raised below it, so the terminal of every level must keep its
	// own value name across the nested chains.
	twice := func(path string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
		if path == "in.y" {
			if src.Int() < 0 {
				panic("negative")
			}

			return next(path, src, dst)
		}

		func() {
			defer func() { _ = recover() }()

			_ = next(path, reflect.ValueOf(frameInner{Y: -1}), dst)
		}()

		return next(path, src, dst)
	}

	e := New(Config{Codecs: convcodec.NewCodecsSet(record).AddError(twice)})

	var out frameDst
	require.NoError(t, e.Convert(&frameSrc{A: 1, In: frameInner{Y: 2}}, &out))
	require.Equal(t, frameDst{A: 1, In: frameInnerDst{Y: 2}}, out)
	require.Equal(t, []string{"a", "a", "in", "in", "y"}, names)
}

func TestIdleErrorFramesAreBounded(t *testing.T) {
	t.Parallel()

	w := New(Config{}).walker()
	defer w.release()

	frames := make([]*errorFrame, maxIdleFrames+8)
	for i := range frames {
		frames[i] = w.errorFrame("f")
		require.Equal(t, "f", frames[i].name)
	}

	for _, f := range frames {
		w.releaseFrame(f)
	}

	require.Len(t, w.frames, maxIdleFrames)
	require.Same(t, frames[maxIdleFrames-1], w.errorFrame("g"), "the last idle frame is reused")
}

func TestPathMemoKeepsTheFirstParent(t *testing.T) {
	t.Parallel()

	var m pathMemo

	require.Equal(t, "a.b", m.join("a", "b"))
	require.Equal(t, "c.b", m.join("c", "b"), "other parents are joined")
	require.Equal(t, "a.b", m.first.Load().path, "the first parent is remembered")
	require.Equal(t, "a.b", m.join("a", "b"))
	require.Equal(t, "b", (&pathMemo{}).join("", "b"))
}
