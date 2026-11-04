// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package googletype_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-convert/googletype"

	"google.golang.org/genproto/googleapis/type/date"
)

// ymd is the comparable content of a date.Date.
type ymd struct{ year, month, day int32 }

func ymdOf(d *date.Date) ymd { return ymd{d.GetYear(), d.GetMonth(), d.GetDay()} }

func dateValue(v ymd) reflect.Value {
	return reflect.ValueOf(&date.Date{Year: v.year, Month: v.month, Day: v.day}).Elem()
}

func TestNewDate_ToTime(t *testing.T) {
	t.Parallel()

	partial := []googletype.Option{googletype.WithAllowPartialDates()}
	tests := []struct {
		name string
		opts []googletype.Option
		src  ymd
		want time.Time
		kind googletype.ErrorKind
	}{
		{name: "full", src: ymd{2025, 6, 15}, want: time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)},
		{name: "leap day", src: ymd{2024, 2, 29}, want: time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC)},
		{name: "first", src: ymd{1, 1, 1}, want: time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)},
		{name: "last", src: ymd{9999, 12, 31}, want: time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)},
		{name: "year only", opts: partial, src: ymd{2025, 0, 0}, want: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		{name: "year and month", opts: partial, src: ymd{2025, 6, 0}, want: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)},
		{name: "month and day", opts: partial, src: ymd{0, 2, 29}, want: time.Date(0, 2, 29, 0, 0, 0, 0, time.UTC)},
		{name: "partial rejected", src: ymd{2025, 6, 0}, kind: googletype.ErrPartialDate},
		{name: "zero rejected", src: ymd{}, kind: googletype.ErrPartialDate},
		{name: "zero with partial option", opts: partial, src: ymd{}, kind: googletype.ErrPartialDate},
		{name: "day without month", opts: partial, src: ymd{2025, 0, 5}, kind: googletype.ErrPartialDate},
		{name: "month alone", opts: partial, src: ymd{0, 5, 0}, kind: googletype.ErrPartialDate},
		{name: "month 13", src: ymd{2025, 13, 1}, kind: googletype.ErrOutOfRange},
		{name: "february 29 non-leap", src: ymd{2025, 2, 29}, kind: googletype.ErrOutOfRange},
		{name: "day 32", src: ymd{2025, 1, 32}, kind: googletype.ErrOutOfRange},
		{name: "negative day", src: ymd{2025, 1, -1}, kind: googletype.ErrOutOfRange},
		{name: "year 10000", src: ymd{10000, 1, 1}, kind: googletype.ErrOutOfRange},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var dst time.Time
			err := run(googletype.NewDate(tt.opts...), dateValue(tt.src), &dst)
			if tt.kind != 0 {
				requireKind(t, err, tt.kind)
				require.True(t, dst.IsZero())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, dst)
		})
	}
}

func TestNewDate_FromTime(t *testing.T) {
	t.Parallel()

	tokyo := time.FixedZone("JST", 9*60*60)
	tests := []struct {
		name string
		src  time.Time
		want ymd
		kind googletype.ErrorKind
	}{
		{name: "midnight", src: time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC), want: ymd{2025, 6, 15}},
		{name: "clock dropped", src: time.Date(2025, 6, 15, 23, 59, 59, 999, time.UTC), want: ymd{2025, 6, 15}},
		{name: "own location", src: time.Date(2025, 6, 16, 1, 0, 0, 0, tokyo), want: ymd{2025, 6, 16}},
		{name: "zero time", src: time.Time{}, want: ymd{1, 1, 1}},
		{name: "year 10000", src: time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC), kind: googletype.ErrOutOfRange},
		{name: "year 0", src: time.Date(0, 12, 31, 0, 0, 0, 0, time.UTC), kind: googletype.ErrOutOfRange},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var dst *date.Date
			err := run(googletype.NewDate(), reflect.ValueOf(tt.src), &dst)
			if tt.kind != 0 {
				requireKind(t, err, tt.kind)
				require.Nil(t, dst)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, ymdOf(dst))
		})
	}
}

func TestNewDate_String(t *testing.T) {
	t.Parallel()

	partial := []googletype.Option{googletype.WithAllowPartialDates()}
	tests := []struct {
		name string
		opts []googletype.Option
		s    string
		d    ymd
	}{
		{name: "full", s: "2025-06-15", d: ymd{2025, 6, 15}},
		{name: "first", s: "0001-01-01", d: ymd{1, 1, 1}},
		{name: "last", s: "9999-12-31", d: ymd{9999, 12, 31}},
		{name: "month and day", opts: partial, s: "0000-12-25", d: ymd{0, 12, 25}},
		{name: "year only", opts: partial, s: "2025-00-00", d: ymd{2025, 0, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			codec := googletype.NewDate(tt.opts...)

			var s string
			require.NoError(t, run(codec, dateValue(tt.d), &s))
			require.Equal(t, tt.s, s)

			var d date.Date
			require.NoError(t, run(codec, reflect.ValueOf(tt.s), &d))
			require.Equal(t, tt.d, ymdOf(&d))
		})
	}
}

func TestNewDate_StringErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  string
		kind googletype.ErrorKind
	}{
		{name: "empty", src: "", kind: googletype.ErrSyntax},
		{name: "slashes", src: "2025/06/15", kind: googletype.ErrSyntax},
		{name: "short year", src: "25-06-15", kind: googletype.ErrSyntax},
		{name: "timestamp", src: "2025-06-15T00:00:00Z", kind: googletype.ErrSyntax},
		{name: "letters", src: "2025-0a-15", kind: googletype.ErrSyntax},
		{name: "signed", src: "+025-06-15", kind: googletype.ErrSyntax},
		{name: "month 13", src: "2025-13-01", kind: googletype.ErrOutOfRange},
		{name: "april 31", src: "2025-04-31", kind: googletype.ErrOutOfRange},
		{name: "partial", src: "2025-06-00", kind: googletype.ErrPartialDate},
		{name: "zero", src: "0000-00-00", kind: googletype.ErrPartialDate},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var dst *date.Date
			requireKind(t, run(googletype.NewDate(), reflect.ValueOf(tt.src), &dst), tt.kind)
			require.Nil(t, dst)
		})
	}
}

func TestNewDate_NilZeroAndDelegation(t *testing.T) {
	t.Parallel()

	codec := googletype.NewDate()
	sentinel := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

	dst := sentinel
	require.NoError(t, run(codec, reflect.ValueOf((*date.Date)(nil)), &dst))
	require.Equal(t, sentinel, dst)

	var d *date.Date
	require.NoError(t, run(codec, reflect.ValueOf((*string)(nil)), &d))
	require.Nil(t, d)

	ignoreZero := googletype.NewDate(googletype.WithIgnoreZero())
	require.NoError(t, run(ignoreZero, dateValue(ymd{}), &dst))
	require.Equal(t, sentinel, dst)
	require.NoError(t, run(ignoreZero, reflect.ValueOf(""), &d))
	require.NoError(t, run(ignoreZero, reflect.ValueOf(time.Time{}), &d))
	require.Nil(t, d)

	var i int64
	require.ErrorIs(t, run(codec, dateValue(ymd{2025, 1, 1}), &i), errNext)
	require.ErrorIs(t, run(codec, reflect.ValueOf("2025-01-01"), &dst), errNext)
}
