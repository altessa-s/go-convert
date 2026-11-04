// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package textcodec_test

import (
	"errors"
	"log/slog"
	"math/big"
	"net"
	"net/netip"
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"

	converter "github.com/altessa-s/go-convert"
	convcodec "github.com/altessa-s/go-convert/codec"
	"github.com/altessa-s/go-convert/codec/textcodec"
)

var errNext = errors.New("next called")

func failNext(string, reflect.Value, reflect.Value) error { return errNext }

// color is a user type with value-receiver MarshalText and pointer-receiver
// UnmarshalText.
type color int

func (c color) MarshalText() ([]byte, error) {
	switch c {
	case 1:
		return []byte("red"), nil
	case 2:
		return []byte("green"), nil
	default:
		return nil, errors.New("unknown color")
	}
}

func (c *color) UnmarshalText(text []byte) error {
	switch string(text) {
	case "red":
		*c = 1
	case "green":
		*c = 2
	default:
		return errors.New("unknown color")
	}
	return nil
}

type label string

func run(t *testing.T, codec convcodec.CodecE, src any, dstPtr any) error {
	t.Helper()
	return codec("field", reflect.ValueOf(src), reflect.ValueOf(dstPtr).Elem(), failNext)
}

func TestNew_Encode(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	ts := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		src  any
		dst  any
		want any
	}{
		{"uuid to string", id, new(string), id.String()},
		{"addr to string", netip.MustParseAddr("10.0.0.1"), new(string), "10.0.0.1"},
		{"prefix pointer to string pointer", new(netip.MustParsePrefix("10.0.0.0/8")), new(*string), new("10.0.0.0/8")},
		{"big int pointer to string", big.NewInt(42), new(string), "42"},
		{"big int value to string", *big.NewInt(-7), new(string), "-7"},
		{"time to bytes", ts, new([]byte), []byte("2025-06-15T12:00:00Z")},
		{"slog level to string", slog.LevelWarn, new(string), "WARN"},
		{"user type to named string", color(2), new(label), label("green")},
		{"double pointer source", new(new(color(1))), new(**string), new(new("red"))},
	}
	codec := textcodec.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, run(t, codec, tt.src, tt.dst))
			require.Equal(t, tt.want, reflect.ValueOf(tt.dst).Elem().Interface())
		})
	}
}

func TestNew_Decode(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

	tests := []struct {
		name string
		src  any
		dst  any
		want any
	}{
		{"string to uuid", id.String(), new(uuid.UUID), id},
		{"string to addr pointer", "::1", new(*netip.Addr), new(netip.IPv6Loopback())},
		{"bytes to prefix", []byte("192.168.0.0/16"), new(netip.Prefix), netip.MustParsePrefix("192.168.0.0/16")},
		{"string to big int pointer", "123456789012345678901234567890", new(*big.Int), bigInt(t, "123456789012345678901234567890")},
		{"string to time", "2025-06-15T12:00:00Z", new(time.Time), time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)},
		{"string to slog level", "ERROR", new(slog.Level), slog.LevelError},
		{"string pointer to user type", new("red"), new(color), color(1)},
		{"named string to user type pointer", label("green"), new(*color), new(color(2))},
	}
	codec := textcodec.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, run(t, codec, tt.src, tt.dst))
			require.Equal(t, tt.want, reflect.ValueOf(tt.dst).Elem().Interface())
		})
	}
}

func bigInt(t *testing.T, s string) *big.Int {
	t.Helper()
	n, ok := new(big.Int).SetString(s, 10)
	require.True(t, ok)
	return n
}

func TestNew_MarshalerToUnmarshaler(t *testing.T) {
	t.Parallel()
	codec := textcodec.New()

	var ip net.IP
	require.NoError(t, run(t, codec, netip.MustParseAddr("192.0.2.1"), &ip))
	require.Equal(t, "192.0.2.1", ip.String())

	var addr netip.Addr
	require.NoError(t, run(t, codec, net.ParseIP("192.0.2.2"), &addr))
	require.Equal(t, netip.MustParseAddr("192.0.2.2"), addr.Unmap())
}

func TestNew_NilSource(t *testing.T) {
	t.Parallel()
	codec := textcodec.New()

	var dst *string
	require.NoError(t, run(t, codec, (*uuid.UUID)(nil), &dst))
	require.Nil(t, dst)

	addr := netip.MustParseAddr("10.0.0.1")
	preset := &addr
	require.NoError(t, run(t, codec, (*string)(nil), &preset))
	require.Equal(t, netip.MustParseAddr("10.0.0.1"), *preset, "nil source leaves the destination unchanged")
}

func TestNew_EmptyText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    []textcodec.Option
		src     any
		dst     any
		want    any
		wantErr bool
	}{
		{"empty string zeroes value", nil, "", new(netip.MustParseAddr("10.0.0.1")), netip.Addr{}, false},
		{"empty string sets nil pointer", nil, "", new(big.NewInt(5)), (*big.Int)(nil), false},
		{"nil bytes zero time", nil, []byte(nil), new(time.Now()), time.Time{}, false},
		{"zero marshaler to string pointer", nil, netip.Addr{}, new(new("x")), (*string)(nil), false},
		{"parse empty accepted", []textcodec.Option{textcodec.WithParseEmpty()}, "", new(netip.MustParseAddr("10.0.0.1")), netip.Addr{}, false},
		{"parse empty rejected", []textcodec.Option{textcodec.WithParseEmpty()}, "", new(time.Time), nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := run(t, textcodec.New(tt.opts...), tt.src, tt.dst)
			if tt.wantErr {
				var textErr *textcodec.Error
				require.ErrorAs(t, err, &textErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, reflect.ValueOf(tt.dst).Elem().Interface())
		})
	}
}

func TestNew_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		src      any
		dst      any
		wantType reflect.Type
	}{
		{"invalid uuid", "not-a-uuid", new(uuid.UUID), reflect.TypeFor[uuid.UUID]()},
		{"invalid addr", "999.0.0.1", new(*netip.Addr), reflect.TypeFor[netip.Addr]()},
		{"invalid big int", []byte("12x"), new(*big.Int), reflect.TypeFor[big.Int]()},
		{"invalid level", "LOUD", new(slog.Level), reflect.TypeFor[slog.Level]()},
		{"marshal failure", color(9), new(string), reflect.TypeFor[color]()},
		{"bridge unmarshal failure", color(1), new(netip.Addr), reflect.TypeFor[netip.Addr]()},
	}
	codec := textcodec.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			before := reflect.ValueOf(tt.dst).Elem().Interface()
			err := run(t, codec, tt.src, tt.dst)
			var textErr *textcodec.Error
			require.ErrorAs(t, err, &textErr)
			require.Equal(t, "field", textErr.Path)
			require.Equal(t, tt.wantType, textErr.Type)
			require.Error(t, errors.Unwrap(err))
			require.Contains(t, err.Error(), `textcodec: field "field"`)
			require.Equal(t, before, reflect.ValueOf(tt.dst).Elem().Interface(), "failed conversion leaves dst unchanged")
		})
	}
}

func TestError_RootPath(t *testing.T) {
	t.Parallel()
	err := &textcodec.Error{Type: reflect.TypeFor[uuid.UUID](), Err: errors.New("bad")}
	require.Equal(t, "textcodec: uuid.UUID: bad", err.Error())
}

func TestNew_PassThrough(t *testing.T) {
	t.Parallel()
	id := uuid.New()

	tests := []struct {
		name string
		src  any
		dst  any
	}{
		{"unrelated types", 1, new(string)},
		{"same type", id, new(uuid.UUID)},
		{"same type through pointers", &id, new(*uuid.UUID)},
		{"marshaler to int", slog.LevelInfo, new(int)},
		{"int to unmarshaler", 4, new(slog.Level)},
		{"string to string", "a", new(string)},
		{"interface destination", id, new(any)},
	}
	codec := textcodec.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, run(t, codec, tt.src, tt.dst), errNext)
		})
	}
}

type textSource struct {
	ID     uuid.UUID
	Addr   *netip.Addr
	Level  slog.Level
	Amount *big.Int
	Name   string
}

type textDTO struct {
	ID     string
	Addr   *string
	Level  string
	Amount string
	Name   string
}

func TestConvertE_RoundTrip(t *testing.T) {
	t.Parallel()
	addr := netip.MustParseAddr("2001:db8::1")
	src := textSource{ID: uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"), Addr: &addr, Level: slog.LevelDebug, Amount: big.NewInt(99), Name: "n"}
	opt := converter.WithErrorCodecs(textcodec.New())

	dto, err := converter.ConvertE(src, &textDTO{}, opt)
	require.NoError(t, err)
	require.Equal(t, &textDTO{ID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8", Addr: new("2001:db8::1"), Level: "DEBUG", Amount: "99", Name: "n"}, dto)

	back, err := converter.ConvertE(dto, &textSource{}, opt)
	require.NoError(t, err)
	require.Equal(t, src.ID, back.ID)
	require.Equal(t, addr, *back.Addr)
	require.Equal(t, src.Level, back.Level)
	require.Zero(t, src.Amount.Cmp(back.Amount))
	require.Equal(t, "n", back.Name)
}

func TestConvertE_ReturnsCodecErrorUnchanged(t *testing.T) {
	t.Parallel()
	codec := textcodec.New()
	var returned error
	recording := func(path string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
		returned = codec(path, src, dst, next)
		return returned
	}

	_, err := converter.ConvertE(textDTO{ID: "nope"}, &textSource{}, converter.WithErrorCodecs(recording))
	require.Error(t, err)
	require.Same(t, returned, err)

	var textErr *textcodec.Error
	require.ErrorAs(t, err, &textErr)
	require.Equal(t, reflect.TypeFor[uuid.UUID](), textErr.Type)
	require.True(t, strings.EqualFold(textErr.Path, "id"), "path %q", textErr.Path)
}
