// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

//go:build integration

package tspb_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-convert/proto/codec/tspb"

	"google.golang.org/protobuf/types/known/timestamppb"

	converter "github.com/altessa-s/go-convert"
)

func TestTimestampRoundTrip(t *testing.T) {
	t.Parallel()

	type domain struct{ Created time.Time }

	instant := time.Date(2026, 11, 2, 10, 30, 0, 0, time.UTC)

	var message struct{ Created *timestamppb.Timestamp }

	_, err := converter.ConvertE(domain{Created: instant}, &message, converter.WithCodecs(tspb.New()))
	require.NoError(t, err)

	var result domain

	_, err = converter.ConvertE(message, &result, converter.WithCodecs(tspb.New()))
	require.NoError(t, err)
	require.True(t, instant.Equal(result.Created))
}
