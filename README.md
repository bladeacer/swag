![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/bladeacer/swag?style=for-the-badge&logo=go)
![GitHub License](https://img.shields.io/github/license/bladeacer/swag?style=for-the-badge)

![Coverage](coverage.svg)

# swag

`swag` (Subtitles With A Gopher) is a tool for reading, writing, and converting subtitles. It converts between the YouTube subtitle formats (YTT, SRV3), Advanced SubStation Alpha (ASS), and the plain formats (SRT, SBV, TTML, WebVTT). It keeps the styling, colour, effects, positioning, CJK layout, ruby text, and karaoke timing that those formats can express.

The project started as a clean-room reimagining of [YTSubConverter](https://github.com/arcusmaximus/YTSubConverter). We thank that project for the inspiration. All code here is original. See [third-party notices](docs/third-party-notices.md) for attribution.

## Install

Install the CLI with the Go toolchain (Go 1.24 or newer):

```sh
go install github.com/bladeacer/swag/cmd/swag@latest
```

The `convert` command is available from v0.2.0. When you need the library itself, build from a clone with `make build` instead.

## Status

The project is in development. The feature set, architecture, and milestones live in [the roadmap](ROADMAP.md) and [the architecture page](docs/architecture.md). The v0.5.0 release adds the Advanced SubStation Alpha (ASS) writer. ASS is now a conversion target, with a semantic round-trip test and a cross-check against YouTube Timed Text. The same release splits the architecture into its own page, raises statement coverage to 100%, and removes the em-dash from the repository. The v0.4.0 release adds the ASS reader, with styles, karaoke, animations, ruby text, vertical layout, and direction. The v0.3.0 release adds the YouTube pair: YouTube Timed Text (YTT) and SRV3 in and out, with the platform quirks that the upload path expects. The plain formats (SRT, SBV) ship in v0.2.0. [The format notes](docs/formats.md) list every format and its degradation notes.

The coverage badge shows the statement coverage of the module. Regenerate it with `make coverage-svg`.

## Documentation

- [The roadmap](ROADMAP.md)
- [The architecture page](docs/architecture.md)
- [The documentation index](docs/index.md)
- [The changelog](docs/changelogs/index.md)

## LLM Usage Disclaimer

We use AI assistance when we work on the codebase.

## Licence

[Apache 2.0](LICENSE). The `simple-english` skill vendored under [the skill directory](skills/simple-english/SKILL.md) keeps its MIT licence (see the file header). [The third-party notices](docs/third-party-notices.md) list the other attributions.
