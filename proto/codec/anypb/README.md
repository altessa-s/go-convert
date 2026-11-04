# anypb

```go
import "github.com/altessa-s/go-convert/proto/codec/anypb"
```

Package `anypb` provides an error codec (`convcodec.CodecE`) for conversion between `anypb.Any` and concrete protobuf messages or
Go domain types. Register it with `converter.WithErrorCodecs` and convert with `converter.ConvertE`.

## Conversions

All conversions work in both directions and for pointer and non-pointer fields.

| Other side             | Behavior                                                                                                    |
|------------------------|-------------------------------------------------------------------------------------------------------------|
| Protobuf message type  | Packed with `anypb.MarshalFrom`; unpacked when the type URL names the message                               |
| Registered domain type | Unpacked into the registered message, which the rest of the codec chain converts into the domain type      |
| Interface type         | Unpacked into the registered domain type for the type URL, or else a message resolved by the resolver      |

An interface source is packed only when it holds a protobuf message, an `Any` or a registered domain value; any other dynamic
value, such as `any("hello")`, is passed to the next codec. An `Any` whose type URL cannot be resolved is an error for `any` and
`proto.Message` destinations and is passed to the next codec for other interfaces.

```go
registry := anypb.NewRegistry().
	Register(&timestamppb.Timestamp{}, reflect.TypeFor[time.Time]()).
	Register(&wrapperspb.StringValue{}, reflect.TypeFor[Note]())

_, err := converter.ConvertE(src, &dst,
	converter.WithErrorCodecs(anypb.New(anypb.WithRegistry(registry))),
	converter.WithCodecs(tspb.New()), // converts timestamppb.Timestamp and time.Time
)
```

The message and domain conversion is delegated to `next`, the way `optionalcodec` composes: register the codecs that bridge
each pair (such as `tspb` above), or rely on the converter's field-by-field copy for structs with matching fields. The chain gives no signal that a codec
converted the pair, so a registered pair without one packs whatever `next` produced (for `time.Time` without `tspb`, an empty
timestamp): register a codec for every pair. The zero value of a registered domain type is absent and packs to a nil Any.

Register every pair before passing the registry to `WithRegistry`: `New` freezes it, and a later `Register` panics. `Register`
is safe to call concurrently, and a frozen registry is safe for concurrent use.

A nil source sets the destination to its zero value.

## Errors

| Error               | Cause                                                                                   |
|---------------------|-----------------------------------------------------------------------------------------|
| `TypeMismatchError` | The type URL does not name the destination's message, or does not implement its interface |
| `UnknownTypeError`  | Neither the registry nor the resolver knows the type URL of an `any` or `proto.Message` destination |
| `Error`             | Protobuf marshaling or unmarshaling failed, or the payload exceeds `WithMaxBytes` |

`converter.ConvertE` returns these errors unchanged.

## Options

| Option          | Description                                                                         |
|-----------------|-------------------------------------------------------------------------------------|
| `WithIgnoreNil` | Leave the destination untouched when the source is nil                              |
| `WithRegistry`  | Map protobuf message types to Go domain types                                       |
| `WithResolver`  | Resolve type URLs for interface destinations (default `protoregistry.GlobalTypes`) |
| `WithMaxBytes`  | Reject `Any` payloads larger than this many bytes before unmarshaling (no limit by default) |
| `WithRecursionLimit` | Nesting limit when unmarshaling payloads (protobuf's default, 10000) |
