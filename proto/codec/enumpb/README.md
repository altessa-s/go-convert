# enumpb

```go
import "github.com/altessa-s/go-convert/proto/codec/enumpb"
```

Package `enumpb` provides error codecs for protobuf enums, any generated type implementing `protoreflect.Enum`.

| Constructor              | Conversion                                                    |
|--------------------------|---------------------------------------------------------------|
| `New(opts...)`           | Enum and any string kind, by value name                       |
| `NewMapping(table, ...)` | Enum `P` and a Go type `G` through a one-to-one `map[P]G`     |

Names come from the enum descriptor (`EnumDescriptor.Values()`). A number, name or Go value without a counterpart fails with
`UnknownValueError`, so register the codecs with `converter.WithErrorCodecs` and use `converter.ConvertE`, which returns the error
unchanged. Pointer variants of either side are supported; a nil source pointer sets the destination to its zero value (a nil
pointer, `""` or the zero enum or Go value).

```go
codec := enumpb.New(enumpb.WithTrimEnumPrefix(), enumpb.WithCase(enumpb.CaseLower), enumpb.WithUnspecifiedAsEmpty())
dto, err := converter.ConvertE(msg, &DTO{}, converter.WithErrorCodecs(codec))
// STATUS_ACTIVE -> "active", STATUS_UNSPECIFIED -> "", "unknown" -> UnknownValueError
```

## Options

| Option                   | Description                                                                         |
|--------------------------|-------------------------------------------------------------------------------------|
| `WithTrimPrefix(p)`      | Strip prefix `p`, such as `STATUS_`, from value names                               |
| `WithTrimEnumPrefix()`   | Strip the prefix derived from the enum name (`FieldPresence` -> `FIELD_PRESENCE_`) |
| `WithCase(c)`            | `CaseLower` or `CaseUpper` names; parsing then ignores case (default `CaseKeep`)    |
| `WithUnspecifiedAsEmpty` | Map the zero enum number to `""`, a nil pointer or the zero Go value, and back      |

Without `WithCase`, names are parsed case-sensitively. `New` and `NewMapping` panic when `WithCase` is given a value other
than `CaseKeep`, `CaseLower` and `CaseUpper`. Only the transformed names are accepted: with a trimmed prefix,
`STATUS_ACTIVE` is unknown and `ACTIVE` is not. Aliases parse to their number; the first declared alias is written.

`NewMapping` ignores the name options. Its table takes precedence over `WithUnspecifiedAsEmpty`, which only covers the zero
number and the zero Go value when the table does not list them. `NewMapping` panics when two enum values map to the same
Go value.
