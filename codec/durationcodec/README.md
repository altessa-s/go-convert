# durationcodec

```go
import "github.com/altessa-s/go-convert/codec/durationcodec"
```

Package `durationcodec` provides an error codec (`convcodec.CodecE`) for bidirectional conversion between `time.Duration` and strings (`"1h30m0s"`, via `time.ParseDuration` and `Duration.String`) or numbers of a configurable unit (seconds by default).

Register it with `convert.WithErrorCodecs(durationcodec.New())`. Parse errors, values that overflow the destination and non-finite floats are returned as `*durationcodec.Error`, which `convert.ConvertE` returns unchanged; overflow errors wrap `strconv.ErrRange`.

## Options

| Option      | Description                                                              |
|-------------|--------------------------------------------------------------------------|
| `WithUnit`  | Duration of one numeric unit, such as `time.Millisecond` (default is `time.Second`) |

Integer destinations truncate toward zero; float destinations keep the fraction, and float sources are rounded to the nanosecond. Only predeclared numeric types (`int`, `int64`, `float64`, ...) are scaled; a defined type such as `type Timeout time.Duration` is left to the engine, which converts it unchanged. An empty string or a nil source sets the destination to its zero value (`nil` for pointer destinations). Both pointer and non-pointer variants are supported.
