# go-convert/googletype

```sh
go get github.com/altessa-s/go-convert/googletype
```

Error-returning codecs for converting between Go types and the common
`google.type` messages from `google.golang.org/genproto/googleapis/type` with
[go-convert](../README.md). This is a separate module so that the root module
does not depend on genproto or protobuf.

| Constructor    | Conversion                                                                 |
|----------------|----------------------------------------------------------------------------|
| `NewMoney`     | `money.Money` and `int64` minor units, decimal `string`, or a user struct  |
| `NewDate`      | `date.Date` and `time.Time` (UTC midnight) or `"YYYY-MM-DD"`               |
| `NewLatLng`    | `latlng.LatLng` and `[2]float64` as `[latitude, longitude]`                |
| `NewTimeOfDay` | `timeofday.TimeOfDay` and `time.Duration` since midnight or `"HH:MM:SS"`   |
| `New`          | All of the above in one codec                                              |

Every constructor returns a `convcodec.CodecE`. Pointer and non-pointer
variants are supported on both sides. A nil source zeroes the destination;
with `WithIgnoreZero` it leaves the destination unchanged.

## Options

| Option                  | Description                                                                 |
|-------------------------|-----------------------------------------------------------------------------|
| `WithCurrency`          | Currency for Money built from `int64` or `string`; required to match on the way back |
| `WithScale`             | Minor-unit digits (0-9) for currencies without their own scale; 2 by default |
| `WithCurrencyScales`    | Minor-unit digits per currency, such as `{"JPY": 0, "KWD": 3}`              |
| `WithMoneyStruct`       | Convert Money to and from structs with the named amount and currency fields |
| `WithAllowPartialDates` | Accept dates with a zero year, month or day; they are still not written as strings |
| `WithIgnoreZero`        | Skip conversion when the source is zero                                     |

## Errors

Invalid input fails the conversion with a `googletype.Error`, which
`convert.ConvertE` returns unchanged. Its `Kind` matches with `errors.Is`:

| Kind                  | Cause                                                              |
|-----------------------|--------------------------------------------------------------------|
| `ErrOverflow`         | Amount does not fit `int64` or the destination integer             |
| `ErrPrecision`        | Amount has more fraction digits than the currency scale            |
| `ErrInvalidCurrency`  | Currency is not three upper-case letters, or none is configured    |
| `ErrCurrencyMismatch` | Money currency differs from `WithCurrency` when it would be dropped |
| `ErrNanosSign`        | Money units and nanos have opposite signs                          |
| `ErrInvalidScale`     | Scale outside 0-9                                                  |
| `ErrPartialDate`      | Date with a zero part, without `WithAllowPartialDates` or written as a string |
| `ErrOutOfRange`       | Month, day, hour, minute, second, nanos, latitude or longitude out of range |
| `ErrSyntax`           | Malformed decimal, date or time-of-day string                      |

## Usage

Register the codecs with `convert.WithErrorCodecs`. They only handle pairs with
their own message on one side, so their order relative to each other does not
matter; register them before any other error codec that claims values by kind.
Error codecs run before the codecs registered with `convert.WithCodecs`, so
`optionalcodec.Codec` cannot unwrap values for them: use pointer fields around
`google.type` values.

```go
var toProto = []convert.Option{
	convert.WithErrorCodecs(
		googletype.NewMoney(googletype.WithCurrency("USD")),
		googletype.NewDate(),
	),
}

msg, err := convert.ConvertE(invoice, &invoicev1.Invoice{}, toProto...)
if errors.Is(err, googletype.ErrOverflow) {
	// ...
}
```
