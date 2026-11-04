# jsoncodec

```go
import "github.com/altessa-s/go-convert/codec/jsoncodec"
```

Package `jsoncodec` provides an error codec for bidirectional conversion between JSON text and Go values using `encoding/json/v2`. `New` returns a
`convcodec.CodecE`; register it with `converter.WithErrorCodecs`.

| JSON text                                    | Go value                                             |
|----------------------------------------------|------------------------------------------------------|
| `jsontext.Value` / `json.RawMessage`         | structs, maps, slices (except byte slices), `any`    |
| `[]byte` (with `WithBytes`)                  | the same                                             |
| `string` (with `WithStrings`)                | the same                                             |

Text to value unmarshals into a fresh value and assigns it, so the destination is replaced rather than merged and is left unchanged on error. Empty
text and a nil source zero the destination. Value to text marshals the value; a nil pointer or nil interface zeroes the destination, while nil maps and
slices follow the JSON options. Any other pair is delegated to the next codec.

Failures are returned as an `Error` with the field path and operation, wrapping the `encoding/json/v2` error (`errors.As` reaches
`*json.SemanticError` and `*jsontext.SyntacticError`); `converter.ConvertE` returns it unchanged. Text longer than the `WithMaxBytes` limit is
rejected before decoding with an `Error` wrapping `SizeError`.

## Options

| Option            | Description                                                              |
|-------------------|--------------------------------------------------------------------------|
| `WithBytes`       | Also treat plain `[]byte` fields as JSON text                            |
| `WithStrings`     | Also treat plain `string` fields as JSON text                            |
| `WithJSONOptions` | Pass `encoding/json/v2` and `jsontext` options to marshal and unmarshal  |
| `WithMaxBytes`    | Reject JSON text longer than n bytes before decoding (default unlimited) |

Both pointer and non-pointer variants are supported.
