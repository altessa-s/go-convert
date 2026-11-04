# enumcodec

```go
import "github.com/altessa-s/go-convert/codec/enumcodec"
```

Package `enumcodec` provides error codecs for bidirectional conversion between a Go enum type `E` and strings or integers through a user-supplied table.

| Constructor                              | Conversion                                                                |
|------------------------------------------|---------------------------------------------------------------------------|
| `New[E](names map[E]string, opts...)`    | `E` / `*E` <-> `string` / `*string`                                       |
| `NewInt[E](values map[E]int64, opts...)` | `E` / `*E` <-> predeclared integer types (`int`, `int32`, ...) / pointers |

Both return a `convcodec.CodecE`; register it with `converter.WithErrorCodecs`. Only pairs with exactly `E` or `*E` on one side and a predeclared
type on the other are claimed; everything else, including a pair of two enum types, is delegated to the next codec. A value missing from the table
is an `UnknownValueError`, and a table integer that does not fit the destination is a `RangeError`; `converter.ConvertE` returns both unchanged. A
nil source zeroes the destination. The constructors panic when two enum values share a name or number, and `NewInt` panics when given
`WithCaseInsensitive`.

## Options

| Option                | Description                                                         |
|-----------------------|---------------------------------------------------------------------|
| `WithCaseInsensitive` | Match source strings against the table ignoring case (`New` only)   |
| `WithUnspecified`     | Map the zero enum value to `""` or `0` and back                     |
| `WithUnknownAsZero`   | Map unknown values to the zero destination value instead of failing |

Both pointer and non-pointer variants are supported.
