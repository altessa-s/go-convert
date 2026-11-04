// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package durationcodec_test

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	converter "github.com/altessa-s/go-convert"
	convcodec "github.com/altessa-s/go-convert/codec"
	"github.com/altessa-s/go-convert/codec/durationcodec"
)

var errNext = errors.New("next called")

func failNext(string, reflect.Value, reflect.Value) error { return errNext }

type label string

func run(t *testing.T, codec convcodec.CodecE, src any, dstPtr any) error {
	t.Helper()
	return codec("field", reflect.ValueOf(src), reflect.ValueOf(dstPtr).Elem(), failNext)
}

const ninetyMinutes = 90 * time.Minute

func TestNew_FromDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		unit time.Duration
		src  any
		dst  any
		want any
	}{
		{"string", 0, ninetyMinutes, new(string), "1h30m0s"},
		{"named string pointer", 0, new(1500 * time.Millisecond), new(*label), new(label("1.5s"))},
		{"int64 seconds", 0, ninetyMinutes, new(int64), int64(5400)},
		{"int seconds truncates", 0, 1999 * time.Millisecond, new(int), 1},
		{"negative int seconds truncates", 0, -1999 * time.Millisecond, new(int32), int32(-1)},
		{"uint16 seconds", 0, time.Minute, new(uint16), uint16(60)},
		{"float64 seconds", 0, 1500 * time.Millisecond, new(float64), 1.5},
		{"float32 seconds pointer", 0, 250 * time.Millisecond, new(*float32), new(float32(0.25))},
		{"int64 milliseconds", time.Millisecond, ninetyMinutes, new(int64), int64(5_400_000)},
		{"float64 minutes", time.Minute, ninetyMinutes, new(float64), 90.0},
		{"double pointer source", 0, new(new(time.Second)), new(**int), new(new(1))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, run(t, durationcodec.New(durationcodec.WithUnit(tt.unit)), tt.src, tt.dst))
			require.Equal(t, tt.want, reflect.ValueOf(tt.dst).Elem().Interface())
		})
	}
}

func TestNew_ToDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		unit time.Duration
		src  any
		dst  any
		want any
	}{
		{"string", 0, "1h30m", new(time.Duration), ninetyMinutes},
		{"string pointer to pointer", 0, new("-250ms"), new(*time.Duration), new(-250 * time.Millisecond)},
		{"named string", 0, label("2m"), new(time.Duration), 2 * time.Minute},
		{"int seconds", 0, 90, new(time.Duration), 90 * time.Second},
		{"uint8 seconds", 0, uint8(3), new(time.Duration), 3 * time.Second},
		{"float seconds", 0, 1.5, new(time.Duration), 1500 * time.Millisecond},
		{"float seconds rounds to nanosecond", 0, 1e-10, new(time.Duration), time.Duration(0)},
		{"float32 seconds", 0, float32(0.5), new(*time.Duration), new(500 * time.Millisecond)},
		{"int64 milliseconds", time.Millisecond, int64(5_400_000), new(time.Duration), ninetyMinutes},
		{"negative milliseconds", time.Millisecond, -20, new(time.Duration), -20 * time.Millisecond},
		{"nanosecond unit max", time.Nanosecond, int64(math.MaxInt64), new(time.Duration), time.Duration(math.MaxInt64)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, run(t, durationcodec.New(durationcodec.WithUnit(tt.unit)), tt.src, tt.dst))
			require.Equal(t, tt.want, reflect.ValueOf(tt.dst).Elem().Interface())
		})
	}
}

func TestNew_EmptyAndNil(t *testing.T) {
	t.Parallel()
	codec := durationcodec.New()

	d := time.Hour
	require.NoError(t, run(t, codec, "", &d))
	require.Zero(t, d, "empty string sets the zero duration")

	p := new(time.Hour)
	require.NoError(t, run(t, codec, "", &p))
	require.Nil(t, p, "empty string sets a nil pointer")

	preset := new(time.Hour)
	require.NoError(t, run(t, codec, (*string)(nil), &preset))
	require.Equal(t, time.Hour, *preset, "nil source leaves the destination unchanged")

	var s *string
	require.NoError(t, run(t, codec, (*time.Duration)(nil), &s))
	require.Nil(t, s)
}

func TestNew_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		unit      time.Duration
		src       any
		dst       any
		wantRange bool
	}{
		{"invalid string", 0, "soon", new(time.Duration), false},
		{"missing unit", 0, new("15"), new(*time.Duration), false},
		{"int overflows duration", 0, int64(math.MaxInt64 / 1000), new(time.Duration), true},
		{"negative int overflows duration", 0, int64(math.MinInt64 / 1000), new(time.Duration), true},
		{"uint overflows duration", time.Millisecond, uint64(math.MaxUint64), new(time.Duration), true},
		{"float overflows duration", 0, 1e19, new(time.Duration), true},
		{"NaN", 0, math.NaN(), new(time.Duration), true},
		{"infinity", 0, math.Inf(-1), new(time.Duration), true},
		{"duration overflows int8", 0, 200 * time.Second, new(int8), true},
		{"negative duration into uint", 0, -time.Second, new(uint), true},
		{"duration overflows uint8", time.Nanosecond, time.Second, new(*uint8), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			before := reflect.ValueOf(tt.dst).Elem().Interface()
			err := run(t, durationcodec.New(durationcodec.WithUnit(tt.unit)), tt.src, tt.dst)
			var durErr *durationcodec.Error
			require.ErrorAs(t, err, &durErr)
			require.Equal(t, "field", durErr.Path)
			require.Equal(t, fmt.Sprint(reflect.Indirect(reflect.ValueOf(tt.src)).Interface()), fmt.Sprint(durErr.Value))
			require.Equal(t, tt.wantRange, errors.Is(err, strconv.ErrRange), "%v", err)
			require.Contains(t, err.Error(), `durationcodec: field "field": `)
			require.Equal(t, before, reflect.ValueOf(tt.dst).Elem().Interface(), "failed conversion leaves dst unchanged")
		})
	}
}

func TestError_RootPath(t *testing.T) {
	t.Parallel()
	err := &durationcodec.Error{Value: "x", Err: errors.New("bad")}
	require.Equal(t, "durationcodec: bad", err.Error())
}

func TestNew_PassThrough(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  any
		dst  any
	}{
		{"unrelated types", 1, new(string)},
		{"duration to duration", time.Second, new(time.Duration)},
		{"duration pointers", new(time.Second), new(*time.Duration)},
		{"duration to bool", time.Second, new(bool)},
		{"bool to duration", true, new(time.Duration)},
		{"duration to interface", time.Second, new(any)},
	}
	codec := durationcodec.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, run(t, codec, tt.src, tt.dst), errNext)
		})
	}
}

type jobConfig struct {
	Timeout time.Duration
	Retry   *time.Duration
	TTL     time.Duration
}

type jobDTO struct {
	Timeout string
	Retry   *string
	TTL     float64
}

func TestConvertE_RoundTrip(t *testing.T) {
	t.Parallel()
	opt := converter.WithErrorCodecs(durationcodec.New())
	src := jobConfig{Timeout: ninetyMinutes, Retry: new(250 * time.Millisecond), TTL: 2500 * time.Millisecond}

	dto, err := converter.ConvertE(src, &jobDTO{}, opt)
	require.NoError(t, err)
	require.Equal(t, &jobDTO{Timeout: "1h30m0s", Retry: new("250ms"), TTL: 2.5}, dto)

	back, err := converter.ConvertE(dto, &jobConfig{}, opt)
	require.NoError(t, err)
	require.Equal(t, &src, back)
}

func TestConvertE_ReturnsCodecErrorUnchanged(t *testing.T) {
	t.Parallel()
	codec := durationcodec.New()
	var returned error
	recording := func(path string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
		returned = codec(path, src, dst, next)
		return returned
	}

	_, err := converter.ConvertE(jobDTO{Timeout: "forever"}, &jobConfig{}, converter.WithErrorCodecs(recording))
	require.Error(t, err)
	require.Same(t, returned, err)

	var durErr *durationcodec.Error
	require.ErrorAs(t, err, &durErr)
	require.Equal(t, "forever", durErr.Value)
}
