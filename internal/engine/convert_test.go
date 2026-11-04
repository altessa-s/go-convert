// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIgnoreAncestors(t *testing.T) {
	t.Parallel()

	require.Nil(t, ignoreAncestors(nil))

	got := ignoreAncestors(map[string]struct{}{"a.b.c": {}, "x": {}})
	require.Equal(t, map[string]struct{}{"": {}, "a": {}, "a.b": {}}, got)
}

func TestHasIgnorePathBelow(t *testing.T) {
	t.Parallel()

	w := &walker{Engine: New(Config{IgnorePaths: map[string]struct{}{"smtp.password": {}}})}

	require.True(t, w.hasIgnorePathBelow(""))
	require.True(t, w.hasIgnorePathBelow("smtp"))
	require.False(t, w.hasIgnorePathBelow("smtp.password"))
	require.False(t, w.hasIgnorePathBelow("sm"))
	require.False(t, w.hasIgnorePathBelow(noIgnorePath))
	require.False(t, (&walker{Engine: New(Config{})}).hasIgnorePathBelow(""))
}

func TestDescendStopsAtTheLimit(t *testing.T) {
	t.Parallel()

	w := &walker{Engine: New(Config{MaxDepth: 2})}
	w.descend("a")
	w.descend("a.b")
	require.PanicsWithValue(t, DepthError{Limit: 2, Path: "a.b.c"}, func() { w.descend("a.b.c") })
}
