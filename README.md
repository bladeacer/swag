# swag

`swag` (Subtitles With A Gopher) is a tool for reading, writing, and converting subtitles. It converts between YouTube subtitle formats (YTT, SRV3), Advanced SubStation Alpha (ASS), and the plain formats (SRT, SBV, TTML, WebVTT), and it keeps the styling, colour, effects, positioning, CJK layout, ruby text, and karaoke timing that those formats can express.

The project started as a clean-room reimagining of [YTSubConverter](https://github.com/arcusmaximus/YTSubConverter). We thank that project for the inspiration. All code here is original. See [third-party notices](docs/third-party-notices.md) for attribution.

## Install

Install the CLI with the Go toolchain (Go 1.24 or newer):

```sh
go install github.com/bladeacer/swag/cmd/swag@latest
```

The `convert` command is available from v0.2.0. Build from a clone with `make build` instead when you need the library itself.

## Status

The project is in planning. The feature set, architecture, and milestones live in [ROADMAP.md](ROADMAP.md). The first release (v0.1.0) lands the model and build tooling.

## Documentation

- [Roadmap and architecture](ROADMAP.md)
- [Documentation index](docs/index.md)
- [Changelog](docs/changelogs/index.md)

## LLM Usage Disclaimer

AI Assistance is used when working on the codebase.

## Licence

[Apache 2.0](LICENSE). The `simple-english` skill vendored under [`skills/simple-english/`](skills/simple-english/SKILL.md) keeps its MIT licence (see the file header). Third-party attributions: [docs/third-party-notices.md](docs/third-party-notices.md).
