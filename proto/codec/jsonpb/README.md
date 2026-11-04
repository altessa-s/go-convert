# jsonpb

```go
import "github.com/altessa-s/go-convert/proto/codec/jsonpb"
```

Package `jsonpb` provides codecs for converting between the `structpb` messages and their plain Go counterparts.

## Codecs

| Constructor | Kind               | Conversions                                                                       |
|-------------|--------------------|-----------------------------------------------------------------------------------|
| `New`       | `convcodec.Codec`  | `json.RawMessage` and `structpb.Struct`; invalid JSON is passed to the next codec |
| `NewE`      | `convcodec.CodecE` | `json.RawMessage` and `structpb.Struct`, `structpb.Value` and `any`, `structpb.ListValue` and `[]any`, `structpb.Struct` and `map[string]any` |

Register `NewE` with `converter.WithErrorCodecs` and convert with `converter.ConvertE`:

```go
_, err := converter.ConvertE(src, &dst, converter.WithErrorCodecs(jsonpb.NewE()))
```

`NewE` maps JSON values to `nil`, `bool`, `float64`, `string`, `[]any` and `map[string]any`. On the way in it also accepts signed
and unsigned integers, `float32` and named types of these kinds. A `structpb.Struct` or `structpb.ListValue` also converts into
an `any` field, and an `any` field converts into a `structpb.Value`, `structpb.ListValue` or `structpb.Struct` by its dynamic value.

A nil source sets the destination to its zero value. `NullValue` converts to `nil`, and `nil` inside a list or map converts to
`NullValue`.

Conversion fails with:

- `ValueError` for non-finite numbers (in either direction), strings that are not valid UTF-8 and unsupported Go types;
  its `Path` locates the value inside nested lists (`tags[2]`) and maps (`meta.key`).
- `Error` when a `json.RawMessage` does not hold a JSON object.

## Options

| Option          | Description                                                     |
|-----------------|-----------------------------------------------------------------|
| `WithIgnoreNil` | Skip conversion when the source is nil, leaving the destination |

Both pointer and non-pointer variants are supported.
