# swag roadmap to v1.0.0

`swag` (Subtitles With A Gopher) is a clean-room Go library and CLI for reading, writing, and converting subtitles. This document fixes the scope and the ordered milestones to v1.0.0. The architecture, the intermediate representation, the format matrix, and the ASS tag tiers live in [the architecture page](docs/architecture.md). Agent contributors: tick a checkbox in the same change that completes the work behind it. Documentation follows the vendored `simple-english` skill with the British English override, as [the contributor rules](AGENTS.md) describe.

We credit [YTSubConverter](https://github.com/arcusmaximus/YTSubConverter) as the source of inspiration for the feature set. We wrote all code from scratch. See [the third-party notices](docs/third-party-notices.md).

## How to read this document

1. Milestones are ordered. Do not start a later milestone before the earlier ones ship, unless a checkbox says otherwise.
2. Each milestone maps to a SemVer minor version. A milestone ships when every box is ticked, coverage stays at 100%, and the changelog carries the release notes.
3. The format support matrix and the ASS tag tiers in [the architecture page](docs/architecture.md) are the source of truth for scope. Update them in the same change that changes scope.

## Milestones

### v0.1.0: foundation
- [x] `go.mod` (module path `freebuff.dev/swag` or final repo path), Go 1.24 toolchain directive
- [x] `internal/model`: Colour (RGBA), Cue, TextSpan, Style, Layout types with constructors and zero-value semantics
- [x] Unit tests for colour parsing, rounding, and alpha edge cases
- [x] `.air.toml` for hot reload during development
- [x] `.goreleaser.yml` v2: linux/darwin/windows (amd64, arm64) + js/wasm, `CGO_ENABLED=0`, `-s -w`, checksums (the wasm build id stays skipped until `cmd/swag-wasm` lands at v1.0.0)
- [x] `Makefile`: build, test, cover, vet, fmt, snapshot release
- [x] CI workflow: vet, test with a coverage floor, `goreleaser check`
- [x] `internal/i18n` skeleton with message catalogue and `en-GB`

### v0.2.0: plain formats
- [x] `internal/formats/srt` reader and writer
- [x] `internal/formats/sbv` reader and writer
- [x] `pkg/sub.Identify` and `pkg/sub.Parse` with format auto-detection
- [x] Round-trip fixtures and tests for both formats
- [x] `cmd/swag` with kong (`-i`, `-o`, `-f`, `--verbose`) and pterm banner, warnings, and result output
- [x] Conversion report (list of feature losses) printed with pterm when `--verbose`

### v0.3.0: the YouTube pair
- [x] `internal/formats/ytt` reader: pens, window positions/styles, ruby groups, karaoke offsets
- [x] `internal/formats/ytt` writer: pen deduplication, the full quirk pipeline, multi-shadow layering
- [x] `internal/formats/srv3` reader and writer sharing the pen model
- [x] Font allow-list table and scale re-mapping with tests
- [x] Karaoke timing model tests (zero-duration bump, offset ordering)
- [x] Sample-driven tests from real YTT/SRV3 fixtures
- [x] Pull sample files from the upstream YTSubConverter repository for end-to-end tests
- [x] The coverage badge in the README, with a `make coverage-svg` target
- [x] The `.goreleaser.yml` release configuration, so a version tag publishes the release

### v0.4.0: ASS reader
- [x] `internal/richtext`: tag lexer, escape resolution, tag argument grammar
- [x] Script Info + V4+ Styles + Events parsing (PlayRes, WrapStyle, Collisions)
- [x] Style-to-IR mapping with the Default-style size baseline rule
- [x] Tier 1 tags, karaoke spans with secondary colour handling
- [x] Tier 3 tags: ruby, vertical, packed, direction
- [x] Tier 2 animations: `\fad`, `\fade`, `\move`, `\t`, shake, chroma, karaoke types
- [x] Fixtures: karaoke sample and colour sample in the YTSubConverter style, written fresh for this project

### v0.5.0: ASS writer
- [x] IR-to-tag emission for tiers 1 to 3
- [x] Outline, shadow, and box (BorderStyle) emission with alpha
- [x] Animation emission and degradation notes where YouTube limits apply
- [x] Round-trip test: ASS to IR to ASS stays semantically equal (colour and timing equality, not byte equality)
- [x] Cross-check suite: ASS to YTT to IR to ASS loses only documented features
- [x] Cross-check suite widened to every shipped writer, so one ASS source proves each target
- [x] Statement coverage raised to 100%, with the floor in [the contributor rules](AGENTS.md) and in CI
- [x] The architecture split out of this roadmap into [the architecture page](docs/architecture.md)
- [x] Em-dash free prose and code, with the rule in [the contributor rules](AGENTS.md)
- [x] Human-readable link aliases in every markdown file
- [x] Release notes linked from the GoReleaser release and from the `make tag` message
- [x] `make tag` suggests the highest changelog version and honours an override
- [x] [Third-party notices](docs/third-party-notices.md) covering the vendored skill, YTSubConverter, and the Go dependencies

### v0.6.0: ASS tag closure, TTML, WebVTT, editor formats
- [x] ASS tag closure: document the YTSubConverter ASS feature list against the code and the tests
- [x] Aegisub overrides where the IR can express them: `\shad`, `\xshad`, `\yshad`, `\a`, `\s`, `\fscx`, and `\fscy`
- [x] `\ytchroma` custom colours and alpha
- [x] `\ytkt` cursor side, tag, and animated forms
- [x] Round-trip fixture for the new overrides
- [x] `internal/formats/ttml`: YouTube dialect reader, general writer
- [x] `internal/formats/vtt`: reader and writer with voice spans and styling
- [x] `internal/formats/kdenlive`: Kdenlive subtitle JSON reader and writer
- [x] `internal/formats/json1`: lossless internal exchange format, versioned
- [x] Loss-report review: every matrix cell has a documented degradation

### v0.7.0: library hardening and format integrity
- [x] Loss notes for a strikeout run and a glyph scale in the YTT, SRV3, SRT, and SBV writers
- [x] Cross-check suite for the override fixture, with the exact loss report of every target
- [x] Multi-way cross-check chains: one fixture per reader through a sequence of formats of length two or more, then back to the source format, with text and timing integrity
- [x] Loss fixes: a symmetric chroma spread and a blank `\ytvert` reset, so ASS carries both without a loss
- [x] `pkg/sub.Convert` stable API with a configuration (style mapping, font policy, loss tolerance)
- [x] Fuzzing for all readers (`go test -fuzz` targets, 30-minute runs in CI nightly)
- [x] Benchmarks for large files (10k cues) with regression tracking
- [x] Statement coverage verified per package, not just module-wide
- [x] i18n coverage for every CLI message. A second locale lands as proof.
- [x] File integrity: the envelope block keeps a whole document in a plain SubRip or SBV file, so a three-way conversion returns the exact document
- [x] `internal/formats/json1` version 2 with an automatic migration chain, and a reader that rejects a version with no step
- [x] WebVTT voice spans (`<v Speaker>`) carried through `TextSpan.Voice`, with a loss note in every writer that has no voice form
- [x] A bare run of the CLI prints the banner and exits 0, and the default command accepts its flags at the top level, so `air` and `make run` work
- [x] End-to-end tests over the command line for every registered target, and a voice span suite across the formats
- [x] Docs: per-format notes with the specification, the support, and the caveats and limits of each format, a library usage guide, an install page, a usage page, a file integrity page, a JSON1 page, and an internationalisation page

### v0.8.0: terminal UI
- [x] `swag interactive`: pick the input, detect the format, pick the target, show the result
- [x] A layout-diffing renderer (`internal/tui`) that writes no bytes for an unchanged frame and repaints only the changed rows
- [x] `--from`/`-F`, a short form for every flag, the `--` separator, the `help` and `convert` words, and a theme-neutral banner
- [x] The integrity block for WebVTT and TTML, with the XML comment form for the XML format
- [x] Batch conversion (`swag convert dir/`) with pterm progress bars and multi-writer output
- [x] Colour and style previews rendered in the terminal (ANSI, best effort), as `swag preview` and in the interactive result frame
- [x] Karaoke timeline preview in the terminal

### v0.9.0: configuration, compatibility, and release candidate
- [ ] Vim inspired keybinds for the interactive mode, with support for multi-key chords such as `Ctrl+Shift+R` and `Alt+Y`, and a leader key that a bind writes as `<leader>`
- [ ] A TOML configuration file with feature parity to the command line: default flags, flag options, the locale, the preferred formats, and custom keybinds. The repository ships the file with its defaults, its other valid options, and comments for each setting
- [ ] `--strict-compat`: turn the integrity block off, so a read and a write stay inside the original specification of the format. The flag is off by default, because a reader that ignores the block still reads the cues
- [ ] Locale selection from the active operating system locale. An unsupported locale falls back to `en-US`, this also means changing the default locale to `en-US` as it is a more sensible default for most users
- [ ] A parallel conversion flag that spreads the work across the CPU cores. The default is `max_cores - 2`, a caller can name a count, and zero uses every core
- [ ] Public API freeze review. Deprecation notes for anything we cut.
- [ ] Full doc sweep under `docs/` with the simple-english skill (CHECK mode pass)
- [ ] Compatibility notes: tested players and platforms matrix published
- [ ] `goreleaser release --snapshot` produces installable artifacts on all 7 targets, wasm demo included
- [ ] Signed release tags and changelog discipline verified from v0.1.0 onward

### v1.0.0: stable
- [ ] A performance audit with the Go profiling tools (`pprof` and `go tool trace`) as well as `strace` and `perf`, over the parse, the render, and the conversion of every format, with the bottlenecks it finds recorded and either fixed or documented
- [ ] SemVer stability guarantee published for `pkg/sub` (breaking changes only at 2.0.0)
- [ ] Guilt-free WASM: `swag` compiles and runs core conversions in a browser demo page
- [ ] All matrix cells shipped or explicitly deferred with an issue link
- [ ] 1.0 release notes, migration guide from YTSubConverter workflows

## Stretch goals (post-1.0 candidates)

- [ ] Survey the subtitle formats that the tool does not support, and add the ones that carry the most value. The candidates include SSA version 4, MicroDVD, SAMI, WebVTT regions and chapters, Universal Subtitle Format, and the broadcast formats below.
- [ ] Simple subtitle editor: start terminal-native with pterm (cue list editor, style editor, live karaoke preview). Web (WASM + a light widget layer) after 1.0 if the terminal editor finds users. Native widget toolkit stays out of scope until then.
- [ ] SCC/CEA-608 writer on the 32-column grid
- [ ] FCPXML caption writer for NLE round-trips
- [ ] Plugin registry for third-party formats (Go interface + registration hook)
