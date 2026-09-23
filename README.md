# swag

`swag` (Subtitles With A Gopher) is a tool for reading, writing, and converting subtitles. It converts between YouTube subtitle formats (YTT, SRV3), Advanced SubStation Alpha (ASS), and the plain formats (SRT, SBV, TTML, WebVTT), and it keeps the styling, colour, effects, positioning, CJK layout, ruby text, and karaoke timing that those formats can express.

The project started as a clean-room reimagining of [YTSubConverter](https://github.com/arcusmaximus/YTSubConverter). We thank that project for the inspiration. All code here is original. See `NOTICE` for attribution.

## Status

The project is in planning. The feature set, architecture, and milestones live in [ROADMAP.md](ROADMAP.md). The first release (v0.1.0) lands the model and build tooling.

## Documentation

- [Roadmap and architecture](ROADMAP.md)
- [Documentation index](docs/index.md)
- [Changelog](docs/changelogs/index.md)

## LLM Usage Disclaimer

AI Assistance is used when working on the codebase.

## Licence

[Apache 2.0](LICENSE). The `simple-english` skill vendored under `.agents/skills/simple-english/` keeps its MIT licence (see the file header).
