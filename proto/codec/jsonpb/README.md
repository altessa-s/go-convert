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

Register `NewE` with `convert.WithErrorCodecs` and convert with `convert.ConvertE`:

```go
_, err := convert.ConvertE(src, &dst, convert.WithErrorCodecs(jsonpb.NewE()))
```

`NewE` maps JSON values to `nil`, `bool`, `float64`, `string`, `[]any` and `map[string]any`. On the way in it also accepts signed
and unsigned integers, `float32` and named types of these kinds. A `structpb.Struct` or `structpb.ListValue` also converts into
an `any` field, and an `any` field converts into a `structpb.Value`, `structpb.ListValue` or `structpb.Struct` by its dynamic value.

A nil source sets the destination to its zero value. A `json.RawMessage` that is empty or holds JSON `null` is a nil source too,
so it converts to a nil `*structpb.Struct`; `{}` converts to an empty `Struct`. `NullValue` converts to `nil`, and `nil` inside a
list or map converts to `NullValue`.

Both codecs parse and render JSON text with `encoding/json/v2`, like the root `jsoncodec`: duplicate object names and invalid
UTF-8 are rejected, and object members are written in sorted order.

Conversion fails with:

- `ValueError` for non-finite numbers (in either direction), strings that are not valid UTF-8, unsupported Go types, lists and
  maps nested deeper than `WithMaxDepth` levels (10000 by default) and lists or maps that contain themselves; its `Path` locates
  the value inside nested lists (`tags[2]`) and maps (`meta.key`).
- `Error` when a `json.RawMessage` is not valid JSON or holds a JSON value other than an object or `null`.

## Options

| Option            | Description                                                          |
|-------------------|----------------------------------------------------------------------|
| `WithIgnoreNil`   | Skip conversion when the source is nil, leaving the destination      |
| `WithMaxDepth(n)` | Nesting limit of lists and maps for `NewE` (`DefaultMaxDepth`, 10000) |

Both pointer and non-pointer variants are supported.
