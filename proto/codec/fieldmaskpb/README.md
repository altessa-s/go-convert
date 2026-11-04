# fieldmaskpb

```go
import "github.com/altessa-s/go-convert/proto/codec/fieldmaskpb"
```

Package `fieldmaskpb` converts `fieldmaskpb.FieldMask` values and applies them to partial updates. Its name matches
`google.golang.org/protobuf/types/known/fieldmaskpb`, so import one of the two under an alias.

## Codec

`New` converts between `fieldmaskpb.FieldMask` and a slice of any string kind holding its paths, in both directions.
Paths are copied, never aliased. A nil mask or nil slice sets the destination to its zero value (a nil pointer or slice), and a
mask without paths leaves a slice pointer destination nil; `WithIgnoreEmpty` keeps the destination instead. Pointer variants
of either side are supported.

`NewE` returns an error codec (`convcodec.CodecE`) that converts the same way but validates every path first and fails with
`PathError`, leaving the destination unchanged. Without `WithMessage` a path must be lower snake_case segments separated by
dots (`address.city`), and `*` is accepted as the only path; with `WithMessage(msg)` every path must select a field of `msg`,
following the `IgnorePaths` rules below. `New` keeps passing paths through unchecked.

```go
codec := fieldmaskpb.NewE(fieldmaskpb.WithMessage(&userpb.User{}))
_, err := converter.ConvertE(src, &dst, converter.WithErrorCodecs(codec))
```

| Option            | Description                                                         |
|-------------------|---------------------------------------------------------------------|
| `WithIgnoreEmpty` | Skip nil or empty masks and slices, leaving the destination         |
| `WithNormalize`   | Sort the paths and drop duplicates and paths covered by a parent    |
| `WithMessage`     | `NewE` only: validate paths against the message descriptor          |

## Partial updates

`IgnorePaths(mask, msg)` returns the ignore paths that restrict a conversion whose source is `msg` to the fields listed in
the mask:

```go
paths, err := fieldmaskpb.IgnorePaths(req.GetUpdateMask(), req.GetUser())
if err != nil {
	return status.Error(codes.InvalidArgument, err.Error())
}
_, err = converter.ConvertE(req.GetUser(), &user, converter.WithIgnorePaths(paths...))
```

A field mask lists the fields to update, while the converter excludes fields, so `IgnorePaths` returns the complement over
the message descriptor:

| Mask                        | Ignore paths for `google.protobuf.Option {name, value: Any {type_url, value}}` |
|-----------------------------|----------------------------------------------------------------------------------|
| `name`                      | `value`                                                                          |
| `value.type_url`            | `name`, `value.value`                                                            |
| `value` (whole subtree)     | `name`                                                                           |
| empty, nil or `*`           | none: every field is updated                                                     |

- **Names.** Mask paths use protobuf field names; the converter matches lower-cased Go field names. `display_name` becomes
  `displayname`, read from the generated struct tags, with the protoc-gen-go naming rules as the fallback for dynamic
  messages. The conversion source must be `msg` or a struct with the same field names, without `WithFieldMappings`.
- **Validation.** Unknown fields, empty segments and paths that traverse a scalar, repeated, map or oneof member field fail
  with `PathError`. JSON names (`displayName`) are not accepted. A repeated or map field may end a path.
- **Oneofs.** A oneof is one Go field: it is ignored unless a member is listed, and then converted whole.
- **Clearing.** Masked fields are written even when zero, which clears them, so do not combine the paths with
  `WithIgnoreZeroValues` or `WithSparseMerge`. The converter skips nil struct pointers, so an unset masked message field
  leaves the destination unchanged.

`IgnorePaths` returns paths rather than `[]convert.Option` because protobuf codec packages may not import the root
package (`architecture_test.go`).
