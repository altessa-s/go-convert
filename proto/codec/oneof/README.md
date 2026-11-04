# oneof

```go
import "github.com/altessa-s/go-convert/proto/codec/oneof"
```

Package `oneof` provides a codec for bidirectional conversion between struct-based oneof representations and protobuf oneof interface types.

## Directions

- Struct with optional pointer fields -> protobuf oneof interface wrapper
- Protobuf oneof interface wrapper -> struct with optional pointer fields

## Constructors

| Function      | Description                                     |
|---------------|-------------------------------------------------|
| `New`         | Create a global oneof codec                     |
| `NewForField` | Create a codec scoped to a specific field name  |

## Options

| Option                 | Description                                                          |
|------------------------|----------------------------------------------------------------------|
| `WithWrapperRegistry`  | Field name → wrapper prototype map; required for struct → oneof      |
| `WithFieldConverter`   | Custom `FieldConverter` for one field                                |
| `WithStrictValidation` | Pass to the next codec when more than one oneof field is non-nil     |
| `WithIgnoreNilFields`  | Leave the destination unchanged and stop the chain when every field is nil and no fallback is set |
| `WithFallbackField`    | Variant to set when every oneof field is nil                         |

## Value conversion

Without a `FieldConverter`, a field value is assigned, converted, or copied field by field. Conversions follow Go's rules with two exceptions:
an integer converted to a string is formatted in decimal (`65` → `"65"`, not `"A"`), and a slice shorter than the destination array is left
unconverted instead of panicking. `string` ↔ `[]byte` and `string` ↔ `[]rune` keep Go's semantics (UTF-8 bytes, decoded code points). A zero
`reflect.Value` or a nil struct pointer is passed to the next codec.

Option and registry lookups are case-insensitive; the struct and wrapper field names themselves must match case exactly.
