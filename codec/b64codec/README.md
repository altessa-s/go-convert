# b64codec

```go
import "github.com/altessa-s/go-convert/codec/b64codec"
```

Package `b64codec` provides an error codec (`convcodec.CodecE`) for bidirectional conversion between `[]byte` and base64 strings.

Register it with `converter.WithErrorCodecs(b64codec.New())`. Invalid base64 input is returned as `*b64codec.Error`, which `converter.ConvertE` returns unchanged.

## Options

| Option          | Description                                                                          |
|-----------------|--------------------------------------------------------------------------------------|
| `WithEncoding`  | Base64 encoding: `base64.StdEncoding` (default), `URLEncoding`, `RawStdEncoding`, `RawURLEncoding` or custom |

An empty string decodes to a non-nil empty slice (`[]byte{}`), so an empty slice and `nil` stay distinct; a nil source pointer sets the destination to its zero value (`nil` slice, `""` or `nil` pointer). Any byte slice kind and string kind is supported, as are pointer variants. The codec claims every byte slice to string conversion, so register more specific codecs before it.
