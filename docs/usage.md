# Usage

`swag` converts one subtitle file into another format. This page covers the command line: the flags, the format detection, the loss report, and the exit codes. [The install page](install.md) covers the install, and [the format notes](formats.md) cover what each format can carry.

## The command

```sh
swag -i INPUT -o OUTPUT [flags]
```

The `convert` word is optional. `swag -i in.ass -o out.srt` and `swag convert -i in.ass -o out.srt` are the same command.

Every flag has a long form and a short form, so `-i in.ass` and `--input in.ass` are the same flag. A separator `--` ends the flag list. Every argument after it is a value, so a file name that starts with a dash still converts:

```sh
swag -i -- -odd-name.srt -f vtt
```

## Flags

| Flag | Short | Meaning |
|---|---|---|
| `--input` | `-i` | The input subtitle file. Required. |
| `--from` | `-F` | The input format name, for example `srt`. The content decides when it is empty. The alias `--input-format` does the same. |
| `--output` | `-o` | The output file. The extension picks the target format. Omit it to write to standard output. |
| `--format` | `-f` | The target format name, for example `srt`. Overrides the output extension. The alias `--to` does the same. |
| `--font` | `-n` | Replace the font of every style and every span before the write. |
| `--strict` | `-s` | Fail when the target format drops a feature. |
| `--verbose` | `-v` | Print the conversion report. |
| `--locale` | `-l` | The message locale, `en-GB` or `en-US`. The `SWAG_LOCALE` environment variable sets the default. |
| `--version` | `-V` | Print the version, the commit, and the build date. |
| `--help` | `-h` | Print the help page. The word `help` does the same. |

`--strict` and `--font` work with the long form and with the short form. `-v` prints the report, and `-V` prints the version.

The other commands carry the flags they need. `interactive` takes `-i`, `-F`, `-f` for the target, `-o`, `-n`, and `-s`. `preview` takes `-i`, `-F`, and `--limit` (`-m`), which bounds the cue rows.

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

## Interactive mode

The `interactive` command asks for the input file, the target format, and the output file, then converts and shows the result:

```sh
swag interactive
```

A value given on the command line skips its question, so this form converts without a prompt:

```sh
swag interactive -i in.ass -f vtt -o out.vtt
```

The result is one frame. It carries the conversion, the loss report, and a preview of the styles and the karaoke timeline. The command repaints only the rows that change, so a still screen costs nothing and a small change costs a small write.

## Batch conversion

An input that is a directory converts every subtitle file under it, including the files in its subdirectories. The `-f` flag names the target, and a comma separates several targets, so one run writes several formats at once:

```sh
swag -i captions/ -f vtt
swag -i captions/ -f srt,vtt -o converted/
```

The `-o` flag names an output directory, which the command creates. Omit it and each output lands beside its input. The relative path of an input is kept, so a file in a subdirectory stays in that subdirectory.

A file whose extension no format claims is skipped. The command prints a progress bar while it works, and it reports the files that failed. A failed file does not stop the run, and the command exits 1 when any file failed.

## Previews

The `preview` command paints the colours, the styles, and the karaoke timeline of a document without converting it:

```sh
swag preview -i in.ass
```

Each style takes one row with a colour swatch, the name, the font, the size, the flags, and a second swatch for a box or an outline. Each cue takes one row with its time range and its text, and a karaoke cue gains a bar underneath, where a covered cell is a sung syllable.

The colours are a best effort. A terminal that carries no colour shows the hex value of each colour instead, and the command sets `--limit` (`-m`) bounds the cue rows. Pass `--limit 0` to show every cue.

## Format detection

The output extension picks the target format. The `-f` flag overrides it, so a file with an unusual extension or a pipe still converts.

The input format comes from the file extension first. When the extension is unknown, the content decides. The `-F` flag names the input format and skips detection, which suits a file with an unusual name:

| Content | Detected format |
|---|---|
| A `WEBVTT` signature, with or without a byte order mark | `vtt` |
| A `<timedtext` root element | `ytt` |
| A line with `-->` | `srt` |
| A first line of the form `h:mm:ss.mmm,h:mm:ss.mmm` | `sbv` |

A file that matches none of those needs a name the tool can use. Rename it with the extension of its format. A caller of the library names the input format instead, as [the library guide](library.md) shows.

## The loss report

No format carries every feature of the others. A writer therefore returns a loss report: one entry per feature that the target format cannot express. The report is always complete for the target, and every entry names the feature. [The loss report review](loss-report.md) lists the entries of every format.

`-v` prints the report:

```
Features the target format does not carry (2):
  karaoke timing
  ruby text in span 3
```

`--strict` turns a non-empty report into an error, and the command exits 1. Use it in a build or a script when a silent downgrade is not acceptable.

A conversion that drops nothing has an empty report, and the command still succeeds.

## File integrity

SubRip, SBV, WebVTT, and TTML hold no room for the extra content, so a conversion that loses features appends an integrity block to the file. The block holds the whole document, and a `swag` reader restores every feature from it. A plain subtitle player stops at the last cue and ignores the block. [The integrity page](integrity.md) describes the block.

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

Every user-facing message comes from the message catalogue, so a bare run reads the same in every shipped locale. The shipped locales are `en-GB` and `en-US`, and the default is `en-GB`. Pass `--locale` or set `SWAG_LOCALE` to change the locale. A tag that is not shipped is an error, so a typo never falls back in silence. [The internationalisation page](i18n.md) covers the catalogue, the translation guidelines, and how to add a language.

The settings file of the tool follows the platform convention, and the `config` command reports where it looks. [The configuration page](configuration.md) covers the location and the two environment variables that move it.
