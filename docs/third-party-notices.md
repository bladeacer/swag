# Third-party notices

This page lists the third-party works this project builds on, what licence each one carries, and how this project uses it. Report a missing or wrong entry in the project issue tracker.

The maintainers license this project under [the Apache License 2.0](../LICENSE).

## Inspiration

### YTSubConverter

| | |
|---|---|
| Upstream | [the YTSubConverter repository](https://github.com/arcusmaximus/YTSubConverter) |
| Licence | [the MIT licence](https://github.com/arcusmaximus/YTSubConverter/blob/master/LICENSE) |
| Use | Inspiration for the supported feature set |

`swag` is a clean-room reimagining (not just reimplementation) of the feature set of YTSubConverter. We thank the YTSubConverter authors for documenting the YouTube subtitle feature set, the platform quirks, and the ASS tag behaviour that `swag` reproduces.

Clean-room statement: the maintainers studied the behaviour and public documentation of YTSubConverter and wrote all `swag` code from scratch. `swag` contains no code, comments, or data tables from YTSubConverter. Where `swag` reproduces a behaviour (for example, the YouTube font allow-list, the opacity ceiling, or the Android dark text workaround), it does so with its own expression and its own tests.

[The sample fetch script](../scripts/fetch-samples.sh) downloads upstream sample files at test time for the end-to-end tests. The files land in the ignored `testdata/upstream/` directory and stay out of the repository. The tests skip when the files are absent.

### Aegisub tag reference

| | |
|---|---|
| Upstream | [the Aegisub ASS tag reference](https://aegi.vmoe.info/docs/3.0/ASS_Tags/) |
| Licence | Documentation, quoted for reference only |
| Use | The source of the ASS Tier 3 tag list |

`swag` names the Tier 3 tags after the public Aegisub tag manual. No text from the manual is copied into this repository.

## Vendored agent skill

### SimpleEnglish

| | |
|---|---|
| Upstream | [the SimpleEnglish repository](https://github.com/AminBlg/SimpleEnglish) |
| Vendored copy | [the vendored skill entry point](../skills/simple-english/SKILL.md) |
| Licence | [the MIT licence](https://github.com/AminBlg/SimpleEnglish/blob/main/LICENSE) |
| Use | Documentation discipline (ASD-STE100 Simplified Technical English) |

The vendored copy carries one project-level change: British English spelling replaces the upstream American spelling rule. [The skill file header](../skills/simple-english/SKILL.md) documents the change, and [the British English spelling rules](../skills/simple-english/references/spelling.md) implement it. Those rules are specific to this project. The upstream licence covers the derivative.

## Go dependencies

[The module file](../go.mod) declares the runtime and CLI dependencies, and each one carries its own licence:

| Package | Licence | Use |
|---|---|---|
| [the kong command line parser](https://github.com/alecthomas/kong) | MIT | CLI flag grammar |
| [the pterm terminal toolkit](https://github.com/pterm/pterm) | MIT | Styled terminal output |

The transitive dependency licences ship with their modules in the Go module cache. Run `go mod graph` for the full transitive list.

## Standards and specifications

The format work refers to public specifications and platform documentation. We copy no specification text into this repository:

- [the WebVTT specification](https://www.w3.org/TR/webvtt1/) for the WebVTT reader and writer
- [the TTML specification](https://www.w3.org/TR/ttml2/) for the TTML reader and writer
- [the Google Timed Text documentation](https://developers.google.com/youtube/v3/docs/captions) for the YouTube dialect
