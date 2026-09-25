# Usage

`swag` converts one subtitle file into another format. This page covers the command line: the flags, the format detection, the loss report, and the exit codes. [The install page](install.md) covers the install, and [the format notes](formats.md) cover what each format can carry.

## The command

```sh
swag -i INPUT -o OUTPUT [flags]
```

The `convert` word is optional. `swag -i in.ass -o out.srt` and `swag convert -i in.ass -o out.srt` are the same command.

## Flags

| Flag | Short | Meaning |
|---|---|---|
| `--input` | `-i` | The input subtitle file. Required. |
| `--output` | `-o` | The output file. The extension picks the target format. Omit it to write to standard output. |
| `--format` | `-f` | The target format name, for example `srt`. Overrides the output extension. |
| `--font` | | Replace the font of every style and every span before the write. |
| `--strict` | | Fail when the target format drops a feature. |
| `--verbose` | `-V` | Print the conversion report. |
| `--report` | `-v` | Print the conversion report. |
| `--locale` | | The message locale, for example `fr-FR`. The `SWAG_LOCALE` environment variable sets the default. |
| `--version` | | Print the version, the commit, and the build date. |

## Examples

Convert an ASS file to WebVTT and print the loss report:

```sh
swag -i in.ass -o out.vtt -v
```

Hold every feature by converting through the lossless exchange format:

```sh
swag -i in.ass -o in.json1
swag -i in.json1 -o out.ass
```

Replace the font on the way to a YouTube upload file:

```sh
swag -i in.ass -o out.ytt --font "Trebuchet MS"
```

Write to standard output and pipe it on:

```sh
swag -i in.srt -f json1 | jq '.document.Cues | length'
```

Fail a build when a conversion would drop a feature:

```sh
swag -i in.ass -o out.srt --strict
```

## Format detection

The output extension picks the target format. The `-f` flag overrides it, so a file with an unusual extension or a pipe still converts.

The input format comes from the file extension first. When the extension is unknown, the content decides:

| Content | Detected format |
|---|---|
| A `WEBVTT` signature, with or without a byte order mark | `vtt` |
| A `<timedtext` root element | `ytt` |
| A line with `-->` | `srt` |
| A first line of the form `h:mm:ss.mmm,h:mm:ss.mmm` | `sbv` |

A file that matches none of those needs a name the tool can use. Rename it with the extension of its format. A caller of the library names the input format instead, as [the library guide](library.md) shows.

## The loss report

No format carries every feature of the others. A writer therefore returns a loss report: one entry per feature that the target format cannot express. The report is always complete for the target, and every entry names the feature. [The loss report review](loss-report.md) lists the entries of every format.

`-v` and `--report` print the report:

```
Features the target format does not carry (2):
  karaoke timing
  ruby text in span 3
```

`--strict` turns a non-empty report into an error, and the command exits 1. Use it in a build or a script when a silent downgrade is not acceptable.

A conversion that drops nothing has an empty report, and the command still succeeds.

## File integrity

SubRip and SBV hold plain text, so a conversion that loses features appends an integrity block to the end of the file. The block holds the whole document, and a `swag` reader restores every feature from it. A plain subtitle player stops at the last cue and ignores the block. [The integrity page](integrity.md) describes the block.

The result is that a three-way conversion keeps the exact content:

```sh
swag -i in.ass -o mid.srt
swag -i mid.srt -o out.ass   # identically named styles, ruby, karaoke, and effects
```

## Exit codes

| Code | Meaning |
|---|---|
| 0 | The conversion succeeded. A bare run, `--help`, and `--version` also exit 0. |
| 1 | The command line could not be read, the conversion failed, or `--strict` found a loss. |

Errors name the file that failed, so a failed run in a script says which input to look at.

## Messages and locales

Every user-facing message comes from the message catalogue, so a bare run reads the same in every shipped locale. The default locale is `en-GB`. Pass `--locale` or set `SWAG_LOCALE` to change it. [The internationalisation page](i18n.md) covers the catalogue and how to add a language.
