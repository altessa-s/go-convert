# sqlnull

```go
import "github.com/altessa-s/go-convert/codec/sqlnull"
```

Package `sqlnull` provides an error codec that bridges `database/sql` null types and their plain counterparts. `New` returns a `convcodec.CodecE`;
register it with `convert.WithErrorCodecs`.

It recognizes the generic `sql.Null[T]` and the legacy `sql.NullString`, `NullInt64`, `NullInt32`, `NullInt16`, `NullByte`, `NullFloat64`,
`NullBool` and `NullTime` by reflection (`IsNullType`). Below, `N` is such a type with value type `T`.

## Supported shapes

| Source        | Destination   | Behavior                                                                 |
|---------------|---------------|--------------------------------------------------------------------------|
| `N`           | `*T`          | `Valid=false` → `nil`; valid → pointer to the value                      |
| `N`           | `T`           | `Valid=false` → zero `T`; valid → the value                              |
| `*T`          | `N`           | `nil` → `Valid=false`; non-nil → valid value                             |
| `T`           | `N`           | valid value (`Valid=false` for a zero value with `WithZeroAsNull`)       |
| `N`           | `N2`          | `Valid=false` stays invalid; a valid value is converted by the chain (*) |
| `N`           | `W` (any)     | `Valid=false` → zero `W`; valid → delegate `T → W` to the chain          |
| `W` (any)     | `N`           | nil `W` → `Valid=false`; else delegate `W → T` to the chain and wrap (*) |
| anything else | anything else | delegated to the next codec                                              |

(*) The converted value is valid even when it is zero, unless `WithZeroAsNull` is set and the source is not a pointer; a non-nil pointer source
always produces a valid value.

Composition mirrors `codec/optionalcodec`: register `sqlnull.New()` before the codec that handles `T ↔ W` (for example `codec/unixtime` for
`sql.NullTime ↔ int64`). Errors returned by the chain are returned unchanged. Null types are recognized as values only, not through pointers.

## Options

| Option           | Description                                                            |
|------------------|------------------------------------------------------------------------|
| `WithZeroAsNull` | Store a zero `T` (or a value converted to a zero `T`) as `Valid=false` |
