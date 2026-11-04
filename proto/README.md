# go-convert/proto

```sh
go get github.com/altessa-s/go-convert/proto
```

Codecs for converting between Go types and protobuf well-known types with
[go-convert](../README.md). This is a separate module so that the root module
does not depend on `google.golang.org/protobuf`. Each release of the root
module `vX.Y.Z` is followed by `proto/vX.Y.Z`, which requires it.

| Package                          | Conversion                                    |
|----------------------------------|-----------------------------------------------|
| [codec/tspb](codec/tspb)         | `timestamppb.Timestamp` and `time.Time`       |
| [codec/durpb](codec/durpb)       | `durationpb.Duration` and `time.Duration`     |
| [codec/pbwrap](codec/pbwrap)     | `wrapperspb` wrappers and Go primitives       |
| [codec/jsonpb](codec/jsonpb)     | `json.RawMessage` and `structpb.Struct`       |
| [codec/oneof](codec/oneof)       | Structs and protobuf oneof fields             |

## Usage

Build the option set once and reuse it at the API boundary:

```go
var toProto = []convert.Option{
	convert.WithCodecs(
		optionalcodec.Codec, // before the type codecs it composes with
		tspb.New(tspb.WithIgnoreZero()),
		durpb.New(durpb.WithIgnoreZero()),
	),
	convert.WithHandleEmbeddedStructs(true),
	convert.WithIgnoreZeroValues(),
}

msg := convert.Convert(user, &userv1.User{}, toProto...)
```
