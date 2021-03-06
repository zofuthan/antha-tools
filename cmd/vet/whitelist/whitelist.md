# whitelist
--
    import "."

Package whitelist defines exceptions for the vet tool.

## Usage

```go
var UnkeyedLiteral = map[string]bool{

	"crypto/x509/pkix.RDNSequence":                  true,
	"crypto/x509/pkix.RelativeDistinguishedNameSET": true,
	"database/sql.RawBytes":                         true,
	"debug/macho.LoadBytes":                         true,
	"encoding/asn1.ObjectIdentifier":                true,
	"encoding/asn1.RawContent":                      true,
	"encoding/json.RawMessage":                      true,
	"encoding/xml.CharData":                         true,
	"encoding/xml.Comment":                          true,
	"encoding/xml.Directive":                        true,
	"antha/scanner.ErrorList":                       true,
	"image/color.Palette":                           true,
	"net.HardwareAddr":                              true,
	"net.IP":                                        true,
	"net.IPMask":                                    true,
	"sort.Float64Slice":                             true,
	"sort.IntSlice":                                 true,
	"sort.StringSlice":                              true,
	"unicode.SpecialCase":                           true,

	"image/color.Alpha16": true,
	"image/color.Alpha":   true,
	"image/color.Gray16":  true,
	"image/color.Gray":    true,
	"image/color.NRGBA64": true,
	"image/color.NRGBA":   true,
	"image/color.RGBA64":  true,
	"image/color.RGBA":    true,
	"image/color.YCbCr":   true,
	"image.Point":         true,
	"image.Rectangle":     true,
	"image.Uniform":       true,
}
```
UnkeyedLiteral are types that are actually slices, but syntactically, we cannot
tell whether the Typ in pkg.Typ{1, 2, 3} is a slice or a struct, so we whitelist
all the standard package library's exported slice types.
