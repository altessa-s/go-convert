# go-convert

[![Go Reference](https://pkg.go.dev/badge/github.com/altessa-s/go-convert.svg)](https://pkg.go.dev/github.com/altessa-s/go-convert)
[![CI](https://github.com/altessa-s/go-convert/actions/workflows/ci.yml/badge.svg)](https://github.com/altessa-s/go-convert/actions/workflows/ci.yml)

Struct-to-struct conversion for Go with field mapping, filtering, partial
updates and pluggable codecs. Typical uses are mapping domain models to API
DTOs and protobuf messages, and applying sparse updates to stored entities.

```sh
go get github.com/altessa-s/go-convert
```

Requires Go 1.27 or later. The root module depends only on the standard
library. [`codec/optionalcodec`](./codec/optionalcodec) and
[`proto`](./proto) are separate modules with their own dependencies.

## Quick start

```go
type User struct {
	ID       int64
	Name     string
	Password string
	Tags     []string
}

type UserDTO struct {
	ID   int64
	Name string
	Tags []string
}

dto := convert.Convert(user, &UserDTO{}, convert.WithIgnoreFields("password"))
```

Fields are matched by name, case-insensitively. Slices, arrays, maps, nested
and embedded structs are converted recursively; convertible scalars follow
Go's conversion rules.

For repeated conversions build a `Converter` once and reuse it. It is safe for
concurrent use:

```go
var toDTO = convert.New[*User, *UserDTO](convert.WithIgnoreFields("password"))

func handle(u *User) (*UserDTO, error) {
	var dto UserDTO
	if err := toDTO.ConvertE(u, &dto); err != nil {
		return nil, err
	}
	return &dto, nil
}
```

Large collections can be converted lazily:

```go
for dto := range convert.ConvertSeq[UserDTO](users) {
	// ...
}

for id, dto := range convert.ConvertMapSeq[string, UserDTO](usersByID) {
	// ...
}
```

## API

| Function / method                | Description                                                    |
|----------------------------------|----------------------------------------------------------------|
| `Convert(src, dst, opts...)`     | Convert a struct or slice of structs; panics on failure        |
| `ConvertE(src, dst, opts...)`    | Like `Convert`, returning failures as errors                   |
| `New[T, U](opts...)`             | Reusable typed `Converter[T, U]`                               |
| `NewAny(opts...)`                | Reusable `Converter[any, any]`                                 |
| `ConvertSeq[U](src, opts...)`    | Lazy iterator over converted slice elements                    |
| `ConvertMapSeq[K, V](src, ...)`  | Lazy iterator over converted map entries                       |
| `Converter.ConvertMapSeq[K, V]`  | Generic method form of `ConvertMapSeq` for a reusable converter |

## Options

| Option                            | Effect                                                              |
|-----------------------------------|---------------------------------------------------------------------|
| `WithIgnoreFields(names...)`      | Skip fields by name at every nesting level                          |
| `WithIgnorePaths(paths...)`       | Skip fields by dotted path from the root, e.g. `smtp.password`      |
| `WithFieldMappings(map)`          | Rename source fields to destination fields                          |
| `WithIgnoreZeroValues()`          | Skip zero-valued source fields                                      |
| `WithIgnoreNilValues()`           | Skip nil pointers, slices and maps                                  |
| `WithSparseMerge()`               | Apply the source as a partial update (JSON Merge Patch semantics)   |
| `WithHandleEmbeddedStructs(init)` | Fill embedded destination structs from the whole source             |
| `WithCopySlices()`                | Give assignable slices their own backing array                      |
| `WithOverflowCheck()`             | Fail on lossy narrowing numeric conversions                         |
| `WithMaxDepth(n)`                 | Limit nesting depth (default `DefaultMaxDepth`)                     |
| `WithCodecs(codecs...)`           | Add custom conversion middleware                                    |
| `WithErrorCodecs(codecs...)`      | Add error-returning middleware that receives full field paths       |

## Errors and untrusted input

`Convert` panics on failure; `ConvertE` and `Converter.ConvertE` return the
same failures as errors and should be used for input that does not come from
a schema known at compile time. The destination may be partially populated
when an error is returned.

| Error                  | Cause                                                             |
|------------------------|-------------------------------------------------------------------|
| `UnsupportedTypeError` | Source and destination shapes cannot be converted                 |
| `OverflowError`        | Lossy numeric conversion with `WithOverflowCheck`                 |
| `DepthError`           | Nesting deeper than `WithMaxDepth`, including cyclic pointer graphs |
| codec errors           | Returned unchanged from codecs registered with `WithErrorCodecs`  |

Traversal depth is always bounded, so cyclic or adversarially deep input fails
with a `DepthError` instead of exhausting the goroutine stack.

## Value-handling modes

Three mutually exclusive options control how source values are applied; the
last one passed wins. The examples share these types:

```go
type Address struct {
	City *string
	Zip  *string
}

type User struct {
	Name    *string
	Age     *int
	Address *Address
	Tags    []string
}

dst := User{
	Name:    ptr("Ann"),
	Age:     ptr(30),
	Address: &Address{City: ptr("Berlin"), Zip: ptr("10115")},
	Tags:    []string{"vip"},
}
src := User{Name: ptr("Bob"), Address: &Address{City: ptr("Paris")}}
```

**Default.** Every matching exported field is copied. Nil struct-field pointers
are skipped; everything else overwrites the destination. A nil slice or map
wipes the destination field and a same-type nested struct pointer is assigned
as a whole.

```go
convert.Convert(&src, &dst)
// Name == "Bob", Age == 30, Address.Zip == nil, Tags == nil
```

**`WithIgnoreZeroValues`** skips every zero source value, so a zero scalar can
never be written. Non-nil empty slices and maps and non-nil pointers to zero
values are not zero and still overwrite.

**`WithIgnoreNilValues`** skips nil pointers, slices and maps. A pointer to a
zero value is written, so scalars can be cleared explicitly. Same-type nested
structs are still assigned wholesale.

**`WithSparseMerge`** applies the source as a patch, the Go-struct analogue of
[JSON Merge Patch](https://www.rfc-editor.org/rfc/rfc7386), at every depth:

| Source field                                    | Effect on destination          |
|-------------------------------------------------|--------------------------------|
| nil pointer, slice or map                       | skipped                        |
| non-nil scalar pointer, including to zero       | written; an explicit empty clears |
| non-nil struct pointer with a set field         | merged recursively             |
| non-nil struct pointer with no set fields       | destination field zeroed       |
| non-nil slice                                   | replaces the destination slice |

```go
convert.Convert(&src, &dst, convert.WithSparseMerge())
// Name == "Bob", Age == 30, Address == {City: "Paris", Zip: "10115"}, Tags == ["vip"]

convert.Convert(&User{Address: &Address{}}, &dst, convert.WithSparseMerge())
// Address == nil
```

The source must express optionality through pointers: a non-pointer scalar is
always present, and a non-pointer nested struct is always merged and never
clears. Structs without exported fields, such as `time.Time`, are assigned
wholesale.

## Ignore paths

`WithIgnoreFields("password")` matches a name at every level the converter
walks. `WithIgnorePaths("smtp.password")` matches one location; a parent path
such as `smtp` skips the whole subtree. Slices, arrays and maps contribute a
single segment, so `items.secret` matches every element of `Items`; indices
and statically typed map keys are never part of a path. Promoted fields use
their promoted path.

Values behind interfaces are walked only when an ignore path lies below them.
String keys of maps reached through an interface then act as path segments.
Nil interfaces and typed nils keep their distinction, containers are detached
before writes, and back edges with ignored descendants leave the destination
unchanged.

## Type conversions

Convertible types follow Go's conversion rules with two exceptions: an integer
converted to a string is formatted in decimal (`65` becomes `"65"`, not
`"A"`), and a slice shorter than a destination array leaves the array
unchanged instead of panicking.

## Codecs

A codec is middleware that handles a field or delegates to the next codec;
the built-in conversion ends the chain. Error codecs (`WithErrorCodecs`) run
first, receive the full dotted source path and stop the conversion with the
first error, which `ConvertE` returns unchanged.

```go
upper := func(field string, src, dst reflect.Value, next convcodec.CodecHandler) {
	if src.Kind() == reflect.String && dst.Kind() == reflect.String {
		dst.SetString(strings.ToUpper(src.String()))
		return
	}
	next(field, src, dst)
}
convert.Convert(in, &out, convert.WithCodecs(upper))
```

| Package                                       | Conversion                                          |
|-----------------------------------------------|-----------------------------------------------------|
| [codec](./codec)                              | Codec contract and chain execution                  |
| [codec/mapslice](./codec/mapslice)            | Map keys or values to a slice                       |
| [codec/unixtime](./codec/unixtime)            | `time.Time` and Unix timestamps                     |
| [codec/optionalcodec](./codec/optionalcodec)  | `optional.Optional[T]` and `T`, `*T`                |
| [proto](./proto)                              | Protobuf codecs (separate module)                   |
| [proto/codec/tspb](./proto/codec/tspb)        | `timestamppb.Timestamp` and `time.Time`             |
| [proto/codec/durpb](./proto/codec/durpb)      | `durationpb.Duration` and `time.Duration`           |
| [proto/codec/pbwrap](./proto/codec/pbwrap)    | `wrapperspb` wrappers and Go primitives             |
| [proto/codec/jsonpb](./proto/codec/jsonpb)    | `json.RawMessage` and `structpb.Struct`             |
| [proto/codec/oneof](./proto/codec/oneof)      | Structs and protobuf oneof fields                   |

## Architecture

```text
convert (facade: Converter[T, U], options, errors)
  └── internal/engine     reflection walker, per-call state and depth guard
        ├── internal/typeinfo   bounded, lock-free struct metadata cache
        ├── internal/primitive  kind-indexed primitive conversions
        └── internal/reflect    reflection helpers
codec                      codec contract, depended on by every adapter
codec/*, proto/codec/*     adapters for specific types
```

Dependencies point inward and are enforced by `architecture_test.go`. The
engine is not generic, so `Converter[T, U]` instantiations share one
implementation.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) and [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE)
