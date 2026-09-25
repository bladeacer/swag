# Third-party notices

This page lists the third-party works this project builds on, what licence each one carries, and how this project uses it.

This project itself is licensed under the [Apache License 2.0](../LICENSE).

## YTSubConverter

| | |
|---|---|
| Upstream | [github.com/arcusmaximus/YTSubConverter](https://github.com/arcusmaximus/YTSubConverter) |
| Licence | [MIT](https://github.com/arcusmaximus/YTSubConverter/blob/master/LICENSE) |
| Use | Inspiration for the supported feature set |

`swag` is a clean-room reimplementation of the feature set of YTSubConverter. We thank the YTSubConverter authors for documenting the YouTube subtitle feature set, the platform quirks, and the ASS tag behaviour that `swag` reproduces.

Clean-room statement: the maintainers studied the behaviour and public documentation of YTSubConverter and wrote all `swag` code from scratch. `swag` contains no code, comments, or data tables from YTSubConverter. Where `swag` reproduces a behaviour (for example, the YouTube font allow-list, opacity ceilings, or the Android dark text workaround), it does so with its own expression and its own tests.

`scripts/fetch-samples.sh` downloads the upstream sample file at test time for the end-to-end tests. The file lands in the ignored `testdata/upstream/` directory and stays out of the repository. The tests skip when the file is absent.

## SimpleEnglish (vendored agent skill)

| | |
|---|---|
| Upstream | [github.com/AminBlg/SimpleEnglish](https://github.com/AminBlg/SimpleEnglish) |
| Vendored copy | [`skills/simple-english/`](../skills/simple-english/SKILL.md) |
| Licence | [MIT](https://github.com/AminBlg/SimpleEnglish/blob/main/LICENSE) |
| Use | Documentation discipline (ASD-STE100 Simplified Technical English) |

The vendored copy carries one project-level change: British English spelling replaces the upstream American spelling rule. The change is documented in the file header of [`skills/simple-english/SKILL.md`](../skills/simple-english/SKILL.md) and implemented in [`skills/simple-english/references/spelling.md`](../skills/simple-english/references/spelling.md), which is specific to this project. The upstream licence covers the derivative.

## Go dependencies

Runtime and CLI dependencies are declared in [`go.mod`](../go.mod) and carry their own licences:

| Package | Licence |
|---|---|
| [github.com/alecthomas/kong](https://github.com/alecthomas/kong) | MIT |
| [github.com/pterm/pterm](https://github.com/pterm/pterm) | MIT |

The transitive dependency licences ship with their modules in the Go module cache.
