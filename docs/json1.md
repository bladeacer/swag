# The JSON1 exchange format

JSON1 is the lossless exchange format of `swag`. It carries the whole intermediate representation (IR), so a document that converts into it and back out keeps every field: styles, span overrides, ruby readings, karaoke timing, animations, and voice names. Use it to hold a document between two tools, or to move a document between two runs of the command line.

```sh
swag -i in.ass -o in.json1
swag -i in.json1 -o out.vtt
```

The name reads as "JSON, version one family": the file carries its own version, and the reader lifts an older file to the current shape. [The integrity block](integrity.md) of a plain subtitle file carries the same format, so a SubRip or SBV file holds a document that a plain player ignores.

## The file

The writer indents with two spaces. Every field of the IR appears under `document`, and every field of the IR is written, so a nil pointer stays visible as `null`:

```json
{
  "version": "2",
  "document": {
    "Metadata": {
      "Title": "Example"
    },
    "Styles": [
      {
        "Name": "Default",
        "Font": "Arial",
        "Size": 20,
        "Bold": false,
        "Italic": false,
        "Underline": false,
        "Primary": {
          "R": 255,
          "G": 255,
          "B": 255,
          "A": 255
        },
        "Secondary": {
          "R": 120,
          "G": 120,
          "B": 120,
          "A": 200
        },
        "Outline": {
          "R": 0,
          "G": 0,
          "B": 0,
          "A": 255
        },
        "Shadow": {
          "R": 0,
          "G": 0,
          "B": 0,
          "A": 0
        },
        "OutlineWidth": 2,
        "ShadowDepth": 0,
        "Alignment": 2,
        "Box": false
      }
    ],
    "Cues": [
      {
        "Start": 1000000000,
        "End": 4000000000,
        "Spans": [
          {
            "Text": "Hello",
            "Start": 0,
            "End": 0,
            "Font": null,
            "Size": null,
            "Bold": null,
            "Italic": null,
            "Underline": null,
            "Strikeout": null,
            "ScaleX": null,
            "ScaleY": null,
            "Fore": null,
            "Secondary": null,
            "Back": null,
            "Shadows": null,
            "OutlineWidth": null,
            "ShadowDepth": null,
            "Vertical": null,
            "Script": null,
            "Direction": null,
            "Packed": null,
            "Voice": null,
            "Ruby": null
          }
        ],
        "Layout": null,
        "Animations": null
      }
    ],
    "VideoDimensions": {
      "X": 1280,
      "Y": 720
    }
  }
}
```

The sample is shortened: the real file indents every nested object on its own lines.

Two conventions of the IR matter to a reader of the file. A pointer field that is `null` means "the source set nothing, inherit from the style", and a colour has no unset state, so an optional colour is a pointer and a plain colour is always present. [The architecture page](architecture.md) describes the IR in full.

## Field notes

| Field | Shape | Meaning |
|---|---|---|
| `version` | string | The version of the file shape. |
| `document` | object | The whole IR document. |
| `Start`, `End` of a cue | integer | Nanoseconds from the start of the media. |
| `Start`, `End` of a span | integer | Karaoke offsets from the cue start, in nanoseconds. Zero means the span is untimed. |
| `Start`, `End` of a keyframe step | integer | The same nanosecond base. |
| `Alignment` | integer | The numpad anchor, where 1 is the bottom left and 9 the top right. |
| `Colour` | object | Four channels, `R`, `G`, `B`, and `A`, each 0 to 255. |
| `Voice` | string or null | The speaker of the span, as WebVTT carries it. |

A tool that does not know a field leaves it out or sets it to `null`. The reader treats a missing field as the zero value of the field, so a file written by hand reads as long as the fields it does carry are well formed.

## Version policy

The format follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html) for the tool and a simple integer for the file shape:

1. A change that adds a field, removes a field, or changes the meaning of a field bumps the version number. A reader must not guess.
2. A change that only adds a field with a nil default can keep the number, and a reader of either version reads the file. The project still bumps the number for clarity.
3. A reader lifts an older file to the current shape before it returns the document. The chain of steps runs in order, so a file from any earlier release opens with the current reader.
4. A reader rejects a version it has no step for. An unknown version is an error, never a silent partial read.

The version history:

| Version | Release | Change |
|---|---|---|
| 1 | v0.6.0 | The first shape. No span voice. |
| 2 | v0.7.0 | Adds `TextSpan.Voice`, the speaker of a span. |

## Migrations

A migration chain lives in [the JSON1 package](../internal/formats/json1/json1.go). Each entry names the version it produces and the function that lifts a document in place:

```go
var migrations = map[string]migration{
	"1": {next: "2", apply: migrateV1},
}
```

The reader follows the chain from the file version to the current one:

```go
func migrate(version string, doc *model.Document) error {
	for version != FormatVersion {
		step, ok := migrations[version]
		if !ok {
			return fmt.Errorf("parse json1: unsupported version %q", version)
		}
		step.apply(doc)
		version = step.next
	}
	return nil
}
```

To add a version, do three things in the same change:

1. Bump `FormatVersion` to the new number.
2. Add a step to the table that lifts the previous version to it. A step that has nothing to change records why.
3. Add a test that reads a file of the previous shape and asserts the new field. [The JSON1 tests](../internal/formats/json1/json1_test.go) carry one such test per step.

A version 1 file has no `Voice` field on a span, and an empty name has no meaning in that shape, so the step only supplies the metadata map that version 1 can leave out. The reader reports the migrated document, and a nil voice keeps its meaning of "no voice".

## Limits

- The format is an internal exchange format of this project. No other tool reads it.
- The writer always writes the current version. There is no flag to write an older one, because a tool that needs an older shape must convert through the format that carries it.
- The file carries no schema URL and no self-describing metadata beyond `version`.
