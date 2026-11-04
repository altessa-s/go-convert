// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package googletype_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"google.golang.org/genproto/googleapis/type/money"

	"github.com/altessa-s/go-convert/googletype"
)

// amount is the comparable content of a money.Money.
type amount struct {
	currency string
	units    int64
	nanos    int32
}

func amountOf(m *money.Money) amount {
	return amount{m.GetCurrencyCode(), m.GetUnits(), m.GetNanos()}
}

func moneyValue(a amount) reflect.Value {
	return reflect.ValueOf(&money.Money{CurrencyCode: a.currency, Units: a.units, Nanos: a.nanos}).Elem()
}

func TestNewMoney_ToInt64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts []googletype.Option
		src  amount
		want int64
		kind googletype.ErrorKind
	}{
		{name: "positive", src: amount{"USD", 12, 340_000_000}, want: 1234},
		{name: "negative", src: amount{"USD", -1, -750_000_000}, want: -175},
		{name: "negative nanos only", src: amount{"USD", 0, -10_000_000}, want: -1},
		{name: "zero", src: amount{"USD", 0, 0}, want: 0},
		{name: "fixed scale", opts: []googletype.Option{googletype.WithScale(3)}, src: amount{"USD", 1, 5_000_000}, want: 1005},
		{
			name: "currency scale", opts: []googletype.Option{googletype.WithCurrencyScales(map[string]int{"JPY": 0})},
			src: amount{"JPY", 500, 0}, want: 500,
		},
		{name: "scale 9", opts: []googletype.Option{googletype.WithScale(9)}, src: amount{"USD", 1, 1}, want: 1_000_000_001},
		{name: "max", src: amount{"USD", math.MaxInt64 / 100, 70_000_000}, want: math.MaxInt64},
		{name: "min", src: amount{"USD", math.MinInt64 / 100, -80_000_000}, want: math.MinInt64},
		{name: "matching currency", opts: []googletype.Option{googletype.WithCurrency("USD")}, src: amount{"USD", 1, 0}, want: 100},
		{name: "precision loss", src: amount{"USD", 1, 5_000_000}, kind: googletype.ErrPrecision},
		{name: "units overflow", src: amount{"USD", math.MaxInt64, 0}, kind: googletype.ErrOverflow},
		{name: "fraction overflow", src: amount{"USD", math.MaxInt64 / 100, 80_000_000}, kind: googletype.ErrOverflow},
		{name: "negative overflow", src: amount{"USD", math.MinInt64 / 100, -90_000_000}, kind: googletype.ErrOverflow},
		{name: "lower-case currency", src: amount{"usd", 1, 0}, kind: googletype.ErrInvalidCurrency},
		{name: "empty currency", src: amount{"", 1, 0}, kind: googletype.ErrInvalidCurrency},
		{name: "long currency", src: amount{"USDT", 1, 0}, kind: googletype.ErrInvalidCurrency},
		{name: "positive units negative nanos", src: amount{"USD", 1, -1}, kind: googletype.ErrNanosSign},
		{name: "negative units positive nanos", src: amount{"USD", -1, 1}, kind: googletype.ErrNanosSign},
		{name: "nanos out of range", src: amount{"USD", 0, 1_000_000_000}, kind: googletype.ErrOutOfRange},
		{name: "currency mismatch", opts: []googletype.Option{googletype.WithCurrency("EUR")}, src: amount{"USD", 1, 0}, kind: googletype.ErrCurrencyMismatch},
		{name: "invalid scale", opts: []googletype.Option{googletype.WithScale(10)}, src: amount{"USD", 1, 0}, kind: googletype.ErrInvalidScale},
		{name: "negative scale", opts: []googletype.Option{googletype.WithScale(-1)}, src: amount{"USD", 1, 0}, kind: googletype.ErrInvalidScale},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dst := int64(-42)

			err := run(googletype.NewMoney(tt.opts...), moneyValue(tt.src), &dst)
			if tt.kind != 0 {
				requireKind(t, err, tt.kind)
				require.Equal(t, int64(-42), dst, "destination must stay unchanged on error")

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, dst)
		})
	}
}

func TestNewMoney_FromInt64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts []googletype.Option
		src  int64
		want amount
		kind googletype.ErrorKind
	}{
		{name: "positive", opts: []googletype.Option{googletype.WithCurrency("USD")}, src: 1234, want: amount{"USD", 12, 340_000_000}},
		{name: "negative", opts: []googletype.Option{googletype.WithCurrency("USD")}, src: -175, want: amount{"USD", -1, -750_000_000}},
		{name: "negative cents", opts: []googletype.Option{googletype.WithCurrency("USD")}, src: -5, want: amount{"USD", 0, -50_000_000}},
		{
			name: "currency scale",
			opts: []googletype.Option{googletype.WithCurrency("KWD"), googletype.WithCurrencyScales(map[string]int{"KWD": 3})},
			src:  1005, want: amount{"KWD", 1, 5_000_000},
		},
		{name: "min", opts: []googletype.Option{googletype.WithCurrency("USD")}, src: math.MinInt64, want: amount{"USD", math.MinInt64 / 100, -80_000_000}},
		{name: "no currency", src: 1, kind: googletype.ErrInvalidCurrency},
		{name: "invalid currency", opts: []googletype.Option{googletype.WithCurrency("US1")}, src: 1, kind: googletype.ErrInvalidCurrency},
		{name: "invalid scale", opts: []googletype.Option{googletype.WithCurrency("USD"), googletype.WithScale(12)}, src: 1, kind: googletype.ErrInvalidScale},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var dst *money.Money

			err := run(googletype.NewMoney(tt.opts...), reflect.ValueOf(tt.src), &dst)
			if tt.kind != 0 {
				requireKind(t, err, tt.kind)
				require.Nil(t, dst)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, dst)
			require.Equal(t, tt.want, amountOf(dst))
		})
	}
}

func TestNewMoney_ToString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts []googletype.Option
		src  amount
		want string
		kind googletype.ErrorKind
	}{
		{name: "cents", src: amount{"USD", 12, 340_000_000}, want: "12.34"},
		{name: "padded to scale", src: amount{"USD", 12, 300_000_000}, want: "12.30"},
		{name: "whole", src: amount{"USD", 7, 0}, want: "7.00"},
		{name: "negative fraction", src: amount{"USD", 0, -500_000_000}, want: "-0.50"},
		{name: "negative", src: amount{"USD", -1, -750_000_000}, want: "-1.75"},
		{name: "beyond scale", src: amount{"USD", 1, 5_000_000}, want: "1.005"},
		{name: "nanos", src: amount{"USD", 0, 1}, want: "0.000000001"},
		{name: "scale 0", opts: []googletype.Option{googletype.WithScale(0)}, src: amount{"JPY", 5, 0}, want: "5"},
		{name: "min units", src: amount{"USD", math.MinInt64, -1}, want: "-9223372036854775808.000000001"},
		{name: "max units", src: amount{"USD", math.MaxInt64, 999_999_999}, want: "9223372036854775807.999999999"},
		{name: "nanos sign", src: amount{"USD", -1, 5}, kind: googletype.ErrNanosSign},
		{name: "invalid currency", src: amount{"$", 1, 0}, kind: googletype.ErrInvalidCurrency},
		{name: "currency mismatch", opts: []googletype.Option{googletype.WithCurrency("EUR")}, src: amount{"USD", 1, 0}, kind: googletype.ErrCurrencyMismatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var dst string

			err := run(googletype.NewMoney(tt.opts...), moneyValue(tt.src), &dst)
			if tt.kind != 0 {
				requireKind(t, err, tt.kind)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, dst)
		})
	}
}

func TestNewMoney_FromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  string
		want amount
		kind googletype.ErrorKind
	}{
		{name: "cents", src: "12.34", want: amount{"USD", 12, 340_000_000}},
		{name: "whole", src: "12", want: amount{"USD", 12, 0}},
		{name: "plus sign", src: "+3.5", want: amount{"USD", 3, 500_000_000}},
		{name: "negative fraction", src: "-0.5", want: amount{"USD", 0, -500_000_000}},
		{name: "negative", src: "-1.75", want: amount{"USD", -1, -750_000_000}},
		{name: "nanos", src: "1.123456789", want: amount{"USD", 1, 123_456_789}},
		{name: "leading zeros", src: "007.10", want: amount{"USD", 7, 100_000_000}},
		{name: "max", src: "9223372036854775807.999999999", want: amount{"USD", math.MaxInt64, 999_999_999}},
		{name: "min", src: "-9223372036854775808.5", want: amount{"USD", math.MinInt64, -500_000_000}},
		{name: "empty", src: "", kind: googletype.ErrSyntax},
		{name: "sign only", src: "-", kind: googletype.ErrSyntax},
		{name: "letters", src: "abc", kind: googletype.ErrSyntax},
		{name: "trailing dot", src: "1.", kind: googletype.ErrSyntax},
		{name: "leading dot", src: ".5", kind: googletype.ErrSyntax},
		{name: "ten fraction digits", src: "1.1234567890", kind: googletype.ErrSyntax},
		{name: "exponent", src: "1e3", kind: googletype.ErrSyntax},
		{name: "space", src: " 1", kind: googletype.ErrSyntax},
		{name: "double sign", src: "--1", kind: googletype.ErrSyntax},
		{name: "two dots", src: "1.2.3", kind: googletype.ErrSyntax},
		{name: "overflow", src: "9223372036854775808", kind: googletype.ErrOverflow},
		{name: "negative overflow", src: "-9223372036854775809", kind: googletype.ErrOverflow},
		{name: "huge", src: "123456789012345678901234567890", kind: googletype.ErrOverflow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var dst money.Money

			err := run(googletype.NewMoney(googletype.WithCurrency("USD")), reflect.ValueOf(tt.src), &dst)
			if tt.kind != 0 {
				requireKind(t, err, tt.kind)
				require.Equal(t, amount{}, amountOf(&dst))

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, amountOf(&dst))
		})
	}
}

// price is a user struct holding minor units and a currency.
type price struct {
	Amount   int64
	Currency string
}

// display is a user struct holding a decimal string and a currency.
type display struct {
	Amount   string
	Currency string
}

// smallPrice has an amount too narrow for large values.
type smallPrice struct {
	Amount   int8
	Currency string
}

// unsignedPrice has an unsigned amount.
type unsignedPrice struct {
	Amount   uint32
	Currency string
}

func TestNewMoney_Struct(t *testing.T) {
	t.Parallel()

	codec := googletype.NewMoney(googletype.WithMoneyStruct("Amount", "Currency"), googletype.WithCurrencyScales(map[string]int{"JPY": 0}))

	t.Run("int amount both directions", func(t *testing.T) {
		t.Parallel()

		var p price
		require.NoError(t, run(codec, moneyValue(amount{"JPY", 1500, 0}), &p))
		require.Equal(t, price{Amount: 1500, Currency: "JPY"}, p)

		var m money.Money
		require.NoError(t, run(codec, reflect.ValueOf(price{Amount: -1999, Currency: "EUR"}), &m))
		require.Equal(t, amount{"EUR", -19, -990_000_000}, amountOf(&m))
	})

	t.Run("string amount both directions", func(t *testing.T) {
		t.Parallel()

		var d *display
		require.NoError(t, run(codec, moneyValue(amount{"USD", 3, 50_000_000}), &d))
		require.Equal(t, &display{Amount: "3.05", Currency: "USD"}, d)

		var m money.Money
		require.NoError(t, run(codec, reflect.ValueOf(&display{Amount: "-2.5", Currency: "GBP"}), &m))
		require.Equal(t, amount{"GBP", -2, -500_000_000}, amountOf(&m))
	})

	t.Run("unsigned amount", func(t *testing.T) {
		t.Parallel()

		var p unsignedPrice
		require.NoError(t, run(codec, moneyValue(amount{"USD", 4, 0}), &p))
		require.Equal(t, unsignedPrice{Amount: 400, Currency: "USD"}, p)
		requireKind(t, run(codec, moneyValue(amount{"USD", -4, 0}), &p), googletype.ErrOverflow)
	})

	t.Run("narrow amount overflow", func(t *testing.T) {
		t.Parallel()

		var p smallPrice
		requireKind(t, run(codec, moneyValue(amount{"USD", 2, 0}), &p), googletype.ErrOverflow)
		require.Equal(t, smallPrice{}, p)
	})

	t.Run("empty currency falls back to WithCurrency", func(t *testing.T) {
		t.Parallel()

		withDefault := googletype.NewMoney(googletype.WithMoneyStruct("Amount", "Currency"), googletype.WithCurrency("CHF"))

		var m money.Money
		require.NoError(t, run(withDefault, reflect.ValueOf(price{Amount: 100}), &m))
		require.Equal(t, amount{"CHF", 1, 0}, amountOf(&m))

		requireKind(t, run(codec, reflect.ValueOf(price{Amount: 100}), &m), googletype.ErrInvalidCurrency)
		requireKind(t, run(codec, reflect.ValueOf(price{Amount: 100, Currency: "eur"}), &m), googletype.ErrInvalidCurrency)
	})

	t.Run("other structs and disabled shape are delegated", func(t *testing.T) {
		t.Parallel()

		type other struct{ Value int64 }

		var o other
		require.ErrorIs(t, run(codec, moneyValue(amount{"USD", 1, 0}), &o), errNext)

		var p price
		require.ErrorIs(t, run(googletype.NewMoney(), moneyValue(amount{"USD", 1, 0}), &p), errNext)
	})
}

func TestNewMoney_NilAndZero(t *testing.T) {
	t.Parallel()

	codec := googletype.NewMoney(googletype.WithCurrency("USD"))

	cleared := int64(7)
	require.NoError(t, run(codec, reflect.ValueOf((*money.Money)(nil)), &cleared))
	require.Zero(t, cleared)

	dst := int64(7)

	var m *money.Money
	require.NoError(t, run(codec, reflect.ValueOf((*int64)(nil)), &m))
	require.Nil(t, m)

	ignoreZero := googletype.NewMoney(googletype.WithCurrency("USD"), googletype.WithIgnoreZero())
	require.NoError(t, run(ignoreZero, moneyValue(amount{}), &dst))
	require.Equal(t, int64(7), dst)
	require.NoError(t, run(ignoreZero, reflect.ValueOf(""), &m))
	require.Nil(t, m)

	// Without WithIgnoreZero an empty Money has no valid currency.
	requireKind(t, run(codec, moneyValue(amount{}), &dst), googletype.ErrInvalidCurrency)
}

func TestNewMoney_Pointers(t *testing.T) {
	t.Parallel()

	codec := googletype.NewMoney(googletype.WithCurrency("USD"))

	src := &money.Money{CurrencyCode: "USD", Units: 2, Nanos: 10_000_000}

	var dst **int64
	require.NoError(t, run(codec, reflect.ValueOf(&src), &dst))
	require.NotNil(t, dst)
	require.NotNil(t, *dst)
	require.Equal(t, int64(201), **dst)

	cents := int64(99)

	var m **money.Money
	require.NoError(t, run(codec, reflect.ValueOf(&cents), &m))
	require.Equal(t, amount{"USD", 0, 990_000_000}, amountOf(*m))
}

func TestNewMoney_Delegates(t *testing.T) {
	t.Parallel()

	codec := googletype.NewMoney(googletype.WithCurrency("USD"))

	var i int64
	require.ErrorIs(t, run(codec, reflect.ValueOf("1"), &i), errNext)

	var f float64
	require.ErrorIs(t, run(codec, moneyValue(amount{"USD", 1, 0}), &f), errNext)

	var m money.Money
	require.ErrorIs(t, run(codec, moneyValue(amount{"USD", 1, 0}), &m), errNext)
}
