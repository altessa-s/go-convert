# textcodec

```go
import "github.com/altessa-s/go-convert/codec/textcodec"
```

Package `textcodec` provides an error codec (`convcodec.CodecE`) that converts types implementing `encoding.TextMarshaler` to `string` or `[]byte`, and `string` or `[]byte` into types whose pointer implements `encoding.TextUnmarshaler`. A `TextMarshaler` source and a `TextUnmarshaler` destination of another type exchange text directly. This covers `uuid.UUID`, `netip.Addr`, `netip.Prefix`, `big.Int`, `time.Time`, `slog.Level` and user types.

Register it with `convert.WithErrorCodecs(textcodec.New())`. `MarshalText` and `UnmarshalText` failures are returned as `*textcodec.Error`, which `convert.ConvertE` returns unchanged.

## Options

| Option            | Description                                                              |
|-------------------|--------------------------------------------------------------------------|
| `WithParseEmpty`  | Pass empty text to `UnmarshalText` instead of zeroing the destination    |

Empty text sets the destination to its zero value (`nil` for pointer destinations) by default. A nil source pointer also sets the destination to its zero value. `MarshalText` with a pointer receiver is called on a copy of the source, so it cannot modify the source. Identical source and destination types are passed to the next codec. Both pointer and non-pointer variants are supported.
