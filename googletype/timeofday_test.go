// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package googletype_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"google.golang.org/genproto/googleapis/type/timeofday"

	"github.com/altessa-s/go-convert/googletype"
)

// hmsn is the comparable content of a timeofday.TimeOfDay.
type hmsn struct{ hours, minutes, seconds, nanos int32 }

func hmsnOf(t *timeofday.TimeOfDay) hmsn {
	return hmsn{t.GetHours(), t.GetMinutes(), t.GetSeconds(), t.GetNanos()}
}

func timeOfDayValue(v hmsn) reflect.Value {
	return reflect.ValueOf(&timeofday.TimeOfDay{Hours: v.hours, Minutes: v.minutes, Seconds: v.seconds, Nanos: v.nanos}).Elem()
}

func TestNewTimeOfDay_RoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tod  hmsn
		d    time.Duration
		s    string
	}{
		{name: "midnight", tod: hmsn{}, d: 0, s: "00:00:00"},
		{name: "morning", tod: hmsn{8, 30, 0, 0}, d: 8*time.Hour + 30*time.Minute, s: "08:30:00"},
		{name: "millis", tod: hmsn{8, 30, 5, 250_000_000}, d: 8*time.Hour + 30*time.Minute + 5250*time.Millisecond, s: "08:30:05.25"},
		{name: "nanos", tod: hmsn{23, 59, 59, 999_999_999}, d: 24*time.Hour - 1, s: "23:59:59.999999999"},
		{name: "end of day", tod: hmsn{24, 0, 0, 0}, d: 24 * time.Hour, s: "24:00:00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			codec := googletype.NewTimeOfDay()

			var d time.Duration
			require.NoError(t, run(codec, timeOfDayValue(tt.tod), &d))
			require.Equal(t, tt.d, d)

			var s string
			require.NoError(t, run(codec, timeOfDayValue(tt.tod), &s))
			require.Equal(t, tt.s, s)

			var fromDuration *timeofday.TimeOfDay
			require.NoError(t, run(codec, reflect.ValueOf(tt.d), &fromDuration))
			require.Equal(t, tt.tod, hmsnOf(fromDuration))

			var fromString timeofday.TimeOfDay
			require.NoError(t, run(codec, reflect.ValueOf(tt.s), &fromString))
			require.Equal(t, tt.tod, hmsnOf(&fromString))
		})
	}
}

func TestNewTimeOfDay_ParseFraction(t *testing.T) {
	t.Parallel()

	var tod timeofday.TimeOfDay
	require.NoError(t, run(googletype.NewTimeOfDay(), reflect.ValueOf("12:00:00.123"), &tod))
	require.Equal(t, hmsn{12, 0, 0, 123_000_000}, hmsnOf(&tod))
}

func TestNewTimeOfDay_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  reflect.Value
		kind googletype.ErrorKind
	}{
		{name: "hour 25", src: timeOfDayValue(hmsn{25, 0, 0, 0}), kind: googletype.ErrOutOfRange},
		{name: "past end of day", src: timeOfDayValue(hmsn{24, 0, 0, 1}), kind: googletype.ErrOutOfRange},
		{name: "minute 60", src: timeOfDayValue(hmsn{1, 60, 0, 0}), kind: googletype.ErrOutOfRange},
		{name: "leap second", src: timeOfDayValue(hmsn{23, 59, 60, 0}), kind: googletype.ErrOutOfRange},
		{name: "negative nanos", src: timeOfDayValue(hmsn{1, 0, 0, -1}), kind: googletype.ErrOutOfRange},
		{name: "nanos overflow", src: timeOfDayValue(hmsn{1, 0, 0, 1_000_000_000}), kind: googletype.ErrOutOfRange},
		{name: "negative hours", src: timeOfDayValue(hmsn{-1, 0, 0, 0}), kind: googletype.ErrOutOfRange},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			codec := googletype.NewTimeOfDay()
			d := time.Duration(-1)
			requireKind(t, run(codec, tt.src, &d), tt.kind)
			require.Equal(t, time.Duration(-1), d)

			var s string
			requireKind(t, run(codec, tt.src, &s), tt.kind)
		})
	}
}

func TestNewTimeOfDay_InputErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  any
		kind googletype.ErrorKind
	}{
		{name: "negative duration", src: -time.Second, kind: googletype.ErrOutOfRange},
		{name: "duration past end of day", src: 24*time.Hour + 1, kind: googletype.ErrOutOfRange},
		{name: "empty", src: "", kind: googletype.ErrSyntax},
		{name: "no seconds", src: "08:30", kind: googletype.ErrSyntax},
		{name: "single digit hour", src: "8:30:00", kind: googletype.ErrSyntax},
		{name: "trailing dot", src: "08:30:00.", kind: googletype.ErrSyntax},
		{name: "ten fraction digits", src: "08:30:00.1234567890", kind: googletype.ErrSyntax},
		{name: "suffix", src: "08:30:00Z", kind: googletype.ErrSyntax},
		{name: "letters", src: "aa:bb:cc", kind: googletype.ErrSyntax},
		{name: "minute 61", src: "08:61:00", kind: googletype.ErrOutOfRange},
		{name: "hour 24 with minutes", src: "24:00:01", kind: googletype.ErrOutOfRange},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var dst *timeofday.TimeOfDay
			requireKind(t, run(googletype.NewTimeOfDay(), reflect.ValueOf(tt.src), &dst), tt.kind)
			require.Nil(t, dst)
		})
	}
}

func TestNewTimeOfDay_NilZeroAndDelegation(t *testing.T) {
	t.Parallel()

	codec := googletype.NewTimeOfDay()

	cleared := time.Minute
	require.NoError(t, run(codec, reflect.ValueOf((*timeofday.TimeOfDay)(nil)), &cleared))
	require.Zero(t, cleared)

	d := time.Minute

	ignoreZero := googletype.NewTimeOfDay(googletype.WithIgnoreZero())
	require.NoError(t, run(ignoreZero, timeOfDayValue(hmsn{}), &d))
	require.Equal(t, time.Minute, d)

	var tod *timeofday.TimeOfDay
	require.NoError(t, run(ignoreZero, reflect.ValueOf(time.Duration(0)), &tod))
	require.Nil(t, tod)

	var i int64
	require.ErrorIs(t, run(codec, timeOfDayValue(hmsn{1, 0, 0, 0}), &i), errNext)
	require.ErrorIs(t, run(codec, reflect.ValueOf(int64(5)), &tod), errNext)
}
