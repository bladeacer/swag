![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/bladeacer/swag?style=for-the-badge&logo=go)
![GitHub License](https://img.shields.io/github/license/bladeacer/swag?style=for-the-badge)

![Coverage](coverage.svg)

# swag

`swag` (Subtitles With A Gopher) reads, writes, and converts subtitles. It keeps the styling, colour, effects, positioning, CJK layout, ruby text, karaoke timing, and voice names that the formats can express. It reports every feature that a target format cannot carry.

It converts between the YouTube caption formats (YTT, SRV3), Advanced SubStation Alpha (ASS), and the plain formats (SRT, SBV, TTML, WebVTT). It also reads and writes the Kdenlive subtitle JSON and a versioned lossless JSON exchange format.

## Quick start

Install the command with the Go toolchain (Go 1.24 or newer):

```sh
go install github.com/bladeacer/swag/cmd/swag@latest
```

Convert a file, and print what the target format cannot carry:

```sh
swag -i in.ass -o out.vtt -v
```

```
 SUCCESS  Wrote out.vtt.
 WARNING  Features the target format does not carry (1):
 WARNING    karaoke timing
```

The report holds one entry per feature the target cannot express, so a file with ruby text or a shadow adds its own lines. [The loss report review](docs/loss-report.md) lists the entries of every format.

A conversion that must not drop a feature fails instead, which suits a build or a batch:

```sh
swag -i in.ass -o out.srt --strict
```

Prefer a prompt to a flag list? The interactive command asks for the input, the target, and the output, then shows the result:

```sh
swag interactive
```

Convert a whole directory, and write several formats at once:

```sh
swag -i captions/ -f srt,vtt -o converted/
```

A three-way conversion keeps the exact document. A plain writer appends an integrity block that holds the whole document. SubRip and SBV then carry ruby text, karaoke, and styling through the tool, while a plain player shows plain subtitles:

```sh
swag -i in.ass -o mid.srt
swag -i mid.srt -o out.ass
```

[The usage page](docs/usage.md) covers the flags, the interactive mode, the batch conversion, the preview, the format detection, and the exit codes. [The install page](docs/install.md) covers releases, builds from source, the platform support, and the development loop. [The configuration page](docs/configuration.md) covers where the settings file lives on each platform.

## Supported formats

| Format | Registry name | Extensions | Support |
|---|---|---|---|
| SubRip | `srt` | `srt` | Read, write |
| YouTube SBV (SubViewer) | `sbv` | `sbv` | Read, write |
| YouTube Timed Text | `ytt` | `ytt` | Read, write |
| YouTube SRV3 | `srv3` | `srv3` | Read, write |
| Advanced SubStation Alpha | `ass` | `ass`, `ssa` | Read, write |
| WebVTT | `vtt` | `vtt` | Read, write |
| TTML and DFXP | `ttml` | `ttml`, `dfxp` | Read, write |
| Kdenlive subtitle JSON | `kdenlive` | `kdenlive` | Read, write |
| Lossless JSON exchange | `json1` | `json1` | Read, write |

[The format notes](docs/formats.md) give the support, the specification, and the caveats and limitations of each format. [The loss report review](docs/loss-report.md) lists the degradation of every format at a glance.

## What makes it different

- No writer drops a feature in silence. Every writer returns a complete loss report for its target, and `--strict` turns a report into a failure. [The loss report review](docs/loss-report.md) documents every entry.
- A conversion chain keeps its fidelity. [The integrity block](docs/integrity.md) holds a full document inside a SubRip, SBV, WebVTT, or TTML file. A conversion through one of those formats then returns the same document.
- The YouTube writer handles the quirks of the platform. It applies the rules of the upload path, from the font allow-list to the zero-width space that keeps a pen. [The format notes](docs/formats.md) list them.
- `pkg/sub` is a library as well as a command. [The library guide](docs/library.md) covers the conversion API, with style renaming, a font override, and a loss policy.
- The tests cover the module to the last statement. The module holds 100 percent statement coverage, a fuzz target for every reader, and benchmarks over a ten-thousand-cue document. [The testing notes](docs/testing.md) cover all three.
- One file carries the settings. The TOML configuration file holds the locale, the default flags, the preferred formats, the keybinds, and the worker count. [The configuration page](docs/configuration.md) records the schema.
- The output is localised. Every user-facing message comes from the message catalogue, with `en-US` as the default and `en-GB` as the second locale. [The internationalisation page](docs/i18n.md) shows how to add a language and states the translation guidelines.
- One source tree builds everywhere. The code is pure Go with no cgo dependency, so it builds for Linux, macOS, Windows, and the web. [The install page](docs/install.md) lists the release targets.

## Library

```go
doc, err := sub.Parse("in.ass", source, "")
if err != nil {
	return err
}
losses, err := sub.Render(doc, "vtt", sink)
```

[The library guide](docs/library.md) covers the document model, the format registry, and the configured conversion entry point.

## Documentation

- [The documentation index](docs/index.md)
- [Install](docs/install.md)
- [Usage](docs/usage.md)
- [Configuration](docs/configuration.md)
- [Formats](docs/formats.md)
- [File integrity](docs/integrity.md)
- [The JSON1 exchange format](docs/json1.md)
- [The library guide](docs/library.md)
- [The architecture page](docs/architecture.md)
- [The roadmap](ROADMAP.md)
- [The changelog](docs/changelogs/index.md)

## Status

The project is in development. [The roadmap](ROADMAP.md) carries the milestones to v1.0.0 and [the changelog index](docs/changelogs/index.md) carries the releases. The v0.9.0 release candidate adds the configuration file, `--strict-compat`, the interactive keybinds, the system locale, and the parallel batch conversion. The v0.8.0 release adds the interactive mode, the layout-diffing renderer, the `--from` flag, the shorthands, and the widened integrity block. The v0.7.0 release adds the configured conversion API, the integrity block for the plain formats, and the JSON1 version chain. It adds the WebVTT voice span, the second locale, and the reference docs. The v0.6.0 release closes the ASS tag list and adds the TTML, WebVTT, Kdenlive, and JSON1 formats. The v0.5.0 release adds the ASS writer, and the v0.4.0 release adds the ASS reader. The YouTube pair (YTT and SRV3) arrives in v0.3.0, and the plain formats (SRT and SBV) in v0.2.0.

The project started as a clean-room reimagining of [YTSubConverter](https://github.com/arcusmaximus/YTSubConverter). We thank that project for the inspiration. All code here is original. [The third-party notices](docs/third-party-notices.md) record the attribution.

The coverage badge shows the statement coverage of the module. Regenerate it with `make coverage-svg`.

## LLM Usage Disclaimer

We use AI assistance when we work on the codebase.

## Licence

[Apache 2.0](LICENSE). The `simple-english` skill vendored under [the skill directory](skills/simple-english/SKILL.md) keeps its MIT licence (see the file header). [The third-party notices](docs/third-party-notices.md) list the other attributions.
