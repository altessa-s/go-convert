// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package googletype_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/genproto/googleapis/type/latlng"
	"google.golang.org/genproto/googleapis/type/money"
	"google.golang.org/genproto/googleapis/type/timeofday"

	"github.com/altessa-s/go-convert/googletype"

	converter "github.com/altessa-s/go-convert"
	convcodec "github.com/altessa-s/go-convert/codec"
)

// store is a domain struct.
type store struct {
	Name     string
	Revenue  int64
	Deposit  *string
	OpenedOn time.Time
	Holiday  string
	Location [2]float64
	Opens    time.Duration
	Closes   string
}

// storeMessage mirrors a generated protobuf message.
type storeMessage struct {
	Name     string
	Revenue  *money.Money
	Deposit  *money.Money
	OpenedOn *date.Date
	Holiday  *date.Date
	Location *latlng.LatLng
	Opens    *timeofday.TimeOfDay
	Closes   *timeofday.TimeOfDay
}

func TestConvertE_RoundTrip(t *testing.T) {
	t.Parallel()

	deposit := "250.5"
	src := store{
		Name:     "Main Street",
		Revenue:  1_234_567,
		Deposit:  &deposit,
		OpenedOn: time.Date(2019, 3, 1, 0, 0, 0, 0, time.UTC),
		Holiday:  "2025-12-25",
		Location: [2]float64{40.7128, -74.006},
		Opens:    9 * time.Hour,
		Closes:   "21:30:00",
	}
	opts := []converter.Option{converter.WithErrorCodecs(googletype.New(googletype.WithCurrency("USD")))}

	msg, err := converter.ConvertE(src, &storeMessage{}, opts...)
	require.NoError(t, err)
	require.Equal(t, "Main Street", msg.Name)
	require.Equal(t, amount{"USD", 12_345, 670_000_000}, amountOf(msg.Revenue))
	require.Equal(t, amount{"USD", 250, 500_000_000}, amountOf(msg.Deposit))
	require.Equal(t, ymd{2019, 3, 1}, ymdOf(msg.OpenedOn))
	require.Equal(t, ymd{2025, 12, 25}, ymdOf(msg.Holiday))
	require.Equal(t, [2]float64{40.7128, -74.006}, [2]float64{msg.Location.GetLatitude(), msg.Location.GetLongitude()})
	require.Equal(t, hmsn{9, 0, 0, 0}, hmsnOf(msg.Opens))
	require.Equal(t, hmsn{21, 30, 0, 0}, hmsnOf(msg.Closes))

	back, err := converter.ConvertE(msg, &store{}, opts...)
	require.NoError(t, err)

	deposit = "250.50"

	require.Equal(t, src, *back)
}

func TestConvertE_NilMessagesLeaveFieldsZero(t *testing.T) {
	t.Parallel()

	got, err := converter.ConvertE(&storeMessage{Name: "empty"}, &store{}, converter.WithErrorCodecs(googletype.New()))
	require.NoError(t, err)
	require.Equal(t, store{Name: "empty"}, *got)
}

func TestConvertE_ReturnsCodecErrorUnchanged(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		src   any
		dst   any
		path  string
		kind  googletype.ErrorKind
		codec convcodec.CodecE
	}{
		{
			name: "money nanos sign", src: struct{ Total *money.Money }{&money.Money{CurrencyCode: "USD", Units: 1, Nanos: -1}},
			dst: &struct{ Total int64 }{}, path: "total", kind: googletype.ErrNanosSign, codec: googletype.NewMoney(),
		},
		{
			name: "money overflow", src: struct{ Total *money.Money }{&money.Money{CurrencyCode: "USD", Units: 1 << 62}},
			dst: &struct{ Total int64 }{}, path: "total", kind: googletype.ErrOverflow, codec: googletype.NewMoney(),
		},
		{
			name: "money invalid currency", src: struct{ Total string }{"1.00"},
			dst: &struct{ Total *money.Money }{}, path: "total", kind: googletype.ErrInvalidCurrency, codec: googletype.NewMoney(),
		},
		{
			name: "nested partial date", src: struct{ Order struct{ Due *date.Date } }{struct{ Due *date.Date }{&date.Date{Year: 2025}}},
			dst: &struct{ Order struct{ Due time.Time } }{}, path: "order.due", kind: googletype.ErrPartialDate, codec: googletype.NewDate(),
		},
		{
			name: "latitude", src: struct{ At [2]float64 }{[2]float64{91, 0}},
			dst: &struct{ At *latlng.LatLng }{}, path: "at", kind: googletype.ErrOutOfRange, codec: googletype.NewLatLng(),
		},
		{
			name: "time of day", src: struct{ Closes string }{"25:00:00"},
			dst: &struct{ Closes *timeofday.TimeOfDay }{}, path: "closes", kind: googletype.ErrOutOfRange, codec: googletype.NewTimeOfDay(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var codecErr error

			capture := func(path string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
				err := tt.codec(path, src, dst, next)
				if err != nil {
					codecErr = err
				}

				return err
			}

			_, err := converter.ConvertE(tt.src, tt.dst, converter.WithErrorCodecs(capture))
			require.Error(t, err)
			require.Equal(t, codecErr, err, "ConvertE must return the codec error unchanged")
			require.ErrorIs(t, err, tt.kind)

			var gtErr googletype.Error
			require.ErrorAs(t, err, &gtErr)
			require.Equal(t, tt.path, gtErr.Path)
		})
	}
}

func TestError_Message(t *testing.T) {
	t.Parallel()

	err := googletype.Error{Path: "order.total", Kind: googletype.ErrOverflow, Detail: "units 1 at scale 2"}
	require.Equal(t, `googletype: field "order.total": value overflows the destination: units 1 at scale 2`, err.Error())
	err.Path = ""
	require.Equal(t, "googletype: value overflows the destination: units 1 at scale 2", err.Error())
	require.Equal(t, "googletype error kind 99", googletype.ErrorKind(99).Error())
}
