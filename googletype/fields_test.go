// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package googletype_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-convert/googletype"

	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/genproto/googleapis/type/dayofweek"
	"google.golang.org/protobuf/types/known/durationpb"

	coreerrs "github.com/altessa-s/go-atlas-core/errors"
)

// requireFieldError fails t unless err is a FieldError on field with reason
// that also wraps cause (when cause is not nil).
func requireFieldError(t *testing.T, err error, field, reason string, cause error) {
	t.Helper()
	fe, ok := errors.AsType[*coreerrs.FieldError](err)
	require.True(t, ok, "err = %v, want a FieldError", err)
	require.Equal(t, field, fe.Field)
	require.Equal(t, reason, fe.Reason)
	if cause != nil {
		require.ErrorIs(t, err, cause)
	}
}

func TestDateFromProtoAt(t *testing.T) {
	t.Parallel()

	got, err := googletype.DateFromProtoAt("date", &date.Date{Year: 2026, Month: 10, Day: 17})
	require.NoError(t, err)
	require.Equal(t, 2026, got.Year)
	require.Equal(t, time.October, got.Month)
	require.Equal(t, 17, got.Day)

	zero, err := googletype.DateFromProtoAt("date", nil)
	require.NoError(t, err)
	require.True(t, zero.IsZero())

	_, err = googletype.DateFromProtoAt("date", &date.Date{Year: 2026, Month: 2, Day: 30})
	requireFieldError(t, err, "date", "impossible date 2026-02-30", googletype.ErrImpossibleDate)
}

func TestDurationFromProtoAt(t *testing.T) {
	t.Parallel()

	d, err := googletype.DurationFromProtoAt("timeout", durationpb.New(3*time.Second))
	require.NoError(t, err)
	require.Equal(t, 3*time.Second, d)

	_, err = googletype.DurationFromProtoAt("timeout", &durationpb.Duration{Seconds: 1, Nanos: -1})
	requireFieldError(t, err, "timeout", googletype.ReasonInvalidDuration, googletype.ErrInvalidDuration)
}

func TestWeekdaysFromProtoAt(t *testing.T) {
	t.Parallel()

	got, err := googletype.WeekdaysFromProtoAt("days", []dayofweek.DayOfWeek{dayofweek.DayOfWeek_MONDAY, dayofweek.DayOfWeek_SUNDAY})
	require.NoError(t, err)
	require.Equal(t, []time.Weekday{time.Monday, time.Sunday}, got)

	_, err = googletype.WeekdaysFromProtoAt("days", []dayofweek.DayOfWeek{dayofweek.DayOfWeek_MONDAY, 0})
	requireFieldError(t, err, "days[1]", googletype.ReasonUnknownWeekday, googletype.ErrUnknownWeekday)
	_, ok := errors.AsType[*googletype.IndexError](err)
	require.True(t, ok, "err = %v, want the IndexError in the chain", err)
}
