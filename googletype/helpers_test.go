// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package googletype_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-convert/googletype"

	convcodec "github.com/altessa-s/go-convert/codec"
)

// errNext is returned by the next handler, so a test can tell that the codec
// delegated the pair.
var errNext = errors.New("next called")

func nextHandler(string, reflect.Value, reflect.Value) error { return errNext }

// run calls codec with src and the value dst points to.
func run(codec convcodec.CodecE, src reflect.Value, dst any) error {
	return codec("Field", src, reflect.ValueOf(dst).Elem(), nextHandler)
}

// requireKind asserts that err is a googletype.Error of the given kind for
// the path "Field".
func requireKind(t *testing.T, err error, kind googletype.ErrorKind) {
	t.Helper()
	require.ErrorIs(t, err, kind)

	var gtErr googletype.Error
	require.ErrorAs(t, err, &gtErr)
	require.Equal(t, "Field", gtErr.Path)
	require.Equal(t, kind, gtErr.Kind)
}
