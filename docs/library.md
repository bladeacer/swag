# Library usage guide

`pkg/sub` is the public API of `swag`. The package detects a format, parses a document into the intermediate representation (IR), and renders the IR in any registered format. A caller of the library never imports an internal package.

## Import the package

```go
import "github.com/bladeacer/swag/pkg/sub"
```

`sub.Document` is an alias of the IR document, so a caller names one type.

## Find the formats

- `sub.Registered()` returns the registry names in alphabetical order.
- `sub.DetectFormat(fileName)` returns the format name of a file name, or an empty string when no format claims the extension.
- `sub.Identify(fileName, source)` reads the content and returns the format name. It uses the file name extension first, then the content signature. It returns an empty string when the format stays unknown.

## Read a document

```go
doc, err := sub.Parse(fileName, source, formatName)
```

`fileName` names the file in an error message and helps detection. Pass an empty `formatName` to auto-detect, or a name such as `"srt"` to force a reader.

## Write a document

```go
losses, err := sub.Render(doc, "ass", sink)
```

The returned slice holds one entry per feature the target cannot express. [The loss report review](loss-report.md) lists the degradations of every format.

## Convert in one step

```go
losses, err := sub.Convert(fileName, source, "ass", sink)
```

`Convert` parses and renders in one call. It reads the input format from the content.

## Configure a conversion

`ConvertWith` takes an `Options` value.

```go
losses, err := sub.ConvertWith(fileName, source, sub.Options{
    Target: "ass",
    Styles: map[string]string{"Default": "Main"},
    Font:   "Verdana",
    Loss:   sub.LossStrict,
}, sink)
```

- `Format` names the input format. An empty value means auto-detection.
- `Target` names the output format. It is required.
- `Styles` renames a style on the way out. The key is the source name and the value the target name. A missing key keeps the name.
- `Font` replaces the font of every style and every span. A writer may then snap the name to its own font list.
- `Loss` decides the handling of a degraded write.

`LossPolicy` has three values:

- `LossReport` returns the loss report. It is the default.
- `LossStrict` returns an error when the target drops a feature. The loss report comes back with the error.
- `LossSilent` drops the loss report.

## A complete example

```go
package main

import (
    "log"
    "os"

    "github.com/bladeacer/swag/pkg/sub"
)

func main() {
    source, err := os.Open("in.srt")
    if err != nil {
        log.Fatal(err)
    }
    defer source.Close()

    sink, err := os.Create("out.ass")
    if err != nil {
        log.Fatal(err)
    }
    defer sink.Close()

    losses, err := sub.ConvertWith("in.srt", source, sub.Options{
        Target: "ass",
        Font:   "Verdana",
    }, sink)
    if err != nil {
        log.Fatal(err)
    }
    for _, loss := range losses {
        log.Printf("loss: %s", loss)
    }
}
```

## Notes

- The internal packages stay internal. The IR types are visible through the `sub.Document` alias, and the format packages are not.
- The registry is fixed at build time. A third-party format needs the plugin hook that the roadmap holds for a later release.
- The command line interface is a thin layer over this API. `swag convert` calls `ConvertWith` with the flags it parsed.
