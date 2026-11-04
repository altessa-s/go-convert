# enumpb

```go
import "github.com/altessa-s/go-convert/proto/codec/enumpb"
```

Package `enumpb` provides error codecs for protobuf enums, any generated type implementing `protoreflect.Enum`.

| Constructor              | Conversion                                                    |
|--------------------------|---------------------------------------------------------------|
| `New(opts...)`           | Enum and any string kind, by value name                       |
| `NewMapping(table, ...)` | Enum `P` and a Go type `G` through a one-to-one `map[P]G`     |
| `NewTable(table, ...)`   | The same table for direct calls: `ToGo`, `ToEnum`, and `Codec` |

Names come from the enum descriptor (`EnumDescriptor.Values()`). A number, name or Go value without a counterpart fails with
`UnknownValueError`, so register the codecs with `convert.WithErrorCodecs` and use `convert.ConvertE`, which returns the error
unchanged. Pointer variants of either side are supported; a nil source pointer sets the destination to its zero value (a nil
pointer, `""` or the zero enum or Go value).

```go
codec := enumpb.MustNew(enumpb.WithTrimEnumPrefix(), enumpb.WithCase(enumpb.CaseLower), enumpb.WithUnspecifiedAsEmpty())
dto, err := convert.ConvertE(msg, &DTO{}, convert.WithErrorCodecs(codec))
// STATUS_ACTIVE -> "active", STATUS_UNSPECIFIED -> "", "unknown" -> UnknownValueError
```

## Options

| Option                   | Description                                                                         |
|--------------------------|-------------------------------------------------------------------------------------|
| `WithTrimPrefix(p)`      | Strip prefix `p`, such as `STATUS_`, from value names                               |
| `WithTrimEnumPrefix()`   | Strip the prefix derived from the enum name (`FieldPresence` -> `FIELD_PRESENCE_`) |
| `WithCase(c)`            | `CaseLower` or `CaseUpper` names; parsing then ignores case (default `CaseKeep`)    |
| `WithUnspecifiedAsEmpty` | Map the zero enum number to `""`, a nil pointer or the zero Go value, and back      |

Without `WithCase`, names are parsed case-sensitively. `New` and `NewMapping` return a `ConfigError` when `WithCase` is given a
value other than `CaseKeep`, `CaseLower` and `CaseUpper`; `MustNew` and `MustNewMapping` panic with it instead. Only the transformed names are accepted: with a trimmed prefix,
`STATUS_ACTIVE` is unknown and `ACTIVE` is not. Aliases parse to their number; the first declared alias is written.

`NewMapping` ignores the name options. Its table takes precedence over `WithUnspecifiedAsEmpty`, which only covers the zero
number and the zero Go value when the table does not list them. `NewMapping` returns a `ConfigError` when two enum values map to the
same Go value.

## Direct conversion

`NewTable` takes the table and options of `NewMapping` and converts single values without `convert.ConvertE`, for handlers that
read one enum field. It returns the same `ConfigError` as `NewMapping`; `MustNewTable` panics with it instead, for package-level
tables:

```go
var statuses = enumpb.MustNewTable(map[userv1.Status]user.Status{
    userv1.Status_STATUS_ACTIVE:  user.Active,
    userv1.Status_STATUS_BLOCKED: user.Blocked,
}, enumpb.WithUnspecifiedAsEmpty())

status, err := statuses.ToGo("user.status", req.GetUser().GetStatus())
```

A value missing from the table fails with `UnknownValueError` carrying the given path; `ToEnum` then returns the zero enum number.
`Codec` returns the table as an error codec, the same as `NewMapping`.

