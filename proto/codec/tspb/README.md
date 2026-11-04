# tspb

```go
import "github.com/altessa-s/go-convert/proto/codec/tspb"
```

Package `tspb` provides a codec for bidirectional conversion between `timestamppb.Timestamp` and `time.Time` or `int64` Unix timestamps.

Millisecond inputs before the Unix epoch are normalized to non-negative nanoseconds, as required by Protobuf Timestamp.

## Options

| Option              | Description                                      |
|---------------------|--------------------------------------------------|
| `WithIgnoreZero`    | Skip conversion for zero-value timestamps        |
| `WithMilliseconds`  | Use millisecond precision (default is seconds)   |

Both pointer and non-pointer variants are supported. Uses unsafe pointer arithmetic for zero-allocation reads/writes when the underlying `reflect.Value`
is addressable.
