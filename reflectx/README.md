# reflectx

Reflection helpers shared by the converter and its codecs, including the codecs
of the nested modules (`proto`, `googletype`, `codec/optionalcodec`). The package
is a public, versioned API of the root module; codecs written outside this
repository may use it too.

| Function        | Purpose                                                                                       |
|-----------------|-----------------------------------------------------------------------------------------------|
| `IndirectType`  | Dereference every pointer layer of a type                                                     |
| `IndirectValue` | Follow non-nil pointer layers of a value; invalid when a pointer is nil                       |
| `MakeDst`       | Allocate nil pointer layers of a destination and move it to the pointed-to value              |
| `SetZero`       | Zero a valid, settable value                                                                  |
| `Writable`      | Report whether `MakeDst` and a write through a destination succeed                            |
| `IsPrimitive`   | Report whether a kind is a number, bool or string                                             |
| `ConvertValue`  | Go conversion with decimal integer to string, single rounding to float32, no short-slice panic |

The nested modules import only exported packages of the root module, so a
consumer combining an older nested module release with a newer root release
keeps compiling. `TestArchitecture` enforces this and `make nested-build-check`
builds every nested module against the root without `replace` directives.
