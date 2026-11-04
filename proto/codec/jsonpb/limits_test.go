// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package jsonpb_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/altessa-s/go-convert/proto/codec/jsonpb"
)

func TestNewE_RawNull(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{`null`, " null\n", ``} {
		got, err := runE[*structpb.Struct](t, jsonpb.NewE(), json.RawMessage(raw))
		require.NoError(t, err, raw)
		require.Nil(t, got, raw)
	}

	got, err := runE[*structpb.Struct](t, jsonpb.NewE(), json.RawMessage(`{}`))
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Empty(t, got.GetFields())

	for _, raw := range []string{`1`, `true`, `"x"`, `[]`} {
		_, err := runE[*structpb.Struct](t, jsonpb.NewE(), json.RawMessage(raw))

		var jsonErr jsonpb.Error
		require.ErrorAs(t, err, &jsonErr, raw)
	}
}

func TestNewE_RawStrictJSON(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{`{"a":1,"a":2}`, "{\"a\":\"\xff\"}"} {
		_, err := runE[*structpb.Struct](t, jsonpb.NewE(), json.RawMessage(raw))

		var jsonErr jsonpb.Error
		require.ErrorAs(t, err, &jsonErr, raw)
	}
}

func TestNewE_CyclicContainers(t *testing.T) {
	t.Parallel()

	m := map[string]any{}
	m["self"] = m

	_, err := runE[*structpb.Struct](t, jsonpb.NewE(), m)

	var valueErr jsonpb.ValueError
	require.ErrorAs(t, err, &valueErr)
	require.Equal(t, "cyclic value", valueErr.Reason)
	require.NotEmpty(t, err.Error())

	s := []any{nil}
	s[0] = s

	_, err = runE[*structpb.ListValue](t, jsonpb.NewE(), s)
	require.ErrorAs(t, err, &valueErr)
	require.Equal(t, "cyclic value", valueErr.Reason)

	// Shared, acyclic containers are fine.
	shared := map[string]any{"k": "v"}
	_, err = runE[*structpb.Struct](t, jsonpb.NewE(), map[string]any{"a": shared, "b": shared})
	require.NoError(t, err)
}

func TestNewE_MaxDepth(t *testing.T) {
	t.Parallel()

	deep := structpb.NewStringValue("leaf")
	for range 20 {
		deep = structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{deep}})
	}

	_, err := runE[any](t, jsonpb.NewE(jsonpb.WithMaxDepth(10)), deep)

	var valueErr jsonpb.ValueError
	require.ErrorAs(t, err, &valueErr)
	require.Equal(t, "nesting exceeds 10 levels", valueErr.Reason)

	got, err := runE[any](t, jsonpb.NewE(jsonpb.WithMaxDepth(30)), deep)
	require.NoError(t, err)
	require.NotNil(t, got)

	// A cyclic Value is rejected instead of overflowing the stack.
	cyclic := &structpb.Value{}
	cyclic.Kind = &structpb.Value_ListValue{ListValue: &structpb.ListValue{Values: []*structpb.Value{cyclic}}}

	_, err = runE[any](t, jsonpb.NewE(), cyclic)
	require.ErrorAs(t, err, &valueErr)

	// Deep nesting in the other direction is limited too.
	var nested any = "leaf"
	for range 20 {
		nested = []any{nested}
	}

	_, err = runE[*structpb.Value](t, jsonpb.NewE(jsonpb.WithMaxDepth(10)), nested)
	require.ErrorAs(t, err, &valueErr)
	require.Equal(t, "nesting exceeds 10 levels", valueErr.Reason)
}
