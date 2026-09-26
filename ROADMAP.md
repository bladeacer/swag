# swag roadmap to v1.1.0

`swag` (Subtitles With A Gopher) is a clean-room Go library and CLI for reading, writing, and converting subtitles. This document fixes the scope and the ordered milestones to v1.1.0. The architecture, the intermediate representation, the format matrix, and the ASS tag tiers live in [the architecture page](docs/architecture.md). Agent contributors: tick a checkbox in the same change that completes the work behind it. Documentation follows the vendored `simple-english` skill with the British English override, as [the contributor rules](AGENTS.md) describe.

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
- [x] The configuration file location per platform: `$XDG_CONFIG_HOME` (or `$HOME/.config`) on Linux, `~/Library/Application Support` on macOS, and `%AppData%` on Windows. `SWAG_CONFIG_DIR` and `SWAG_CONFIG` override the directory and the file, and `swag config` reports the resolved location.
- [x] Platform and architecture support documented for every release target, with the pure-Go and cgo-free build recorded
- [x] Batch conversion (`swag convert dir/`) with pterm progress bars and multi-writer output
- [x] Colour and style previews rendered in the terminal (ANSI, best effort), as `swag preview` and in the interactive result frame
- [x] Karaoke timeline preview in the terminal

### v0.9.0: configuration, compatibility, and release candidate
- [x] Vim inspired keybinds for the interactive mode, with support for multi-key chords such as `Ctrl+Shift+R` and `Alt+Y`, and a leader key that a bind writes as `<leader>`
  - The notation joins a modifier with `+` and a chord with a space, so `ctrl+shift+r` is one key and `ctrl+x a` is two. The actions are `accept`, `help`, and `quit`, and the tool refuses an unknown action and a key that two bindings share. The prompts stay one answer per line, so the keymap is a layer over them and not a full screen input widget.
- [x] A TOML configuration file with feature parity to the command line: default flags, flag options, the locale, the preferred formats, and custom keybinds. The repository ships the file with its defaults, its other valid options, and comments for each setting
  - The file carries the `keybinds` table now, and the table takes action with the keybind item below. The `preferred` list is the default target set of a batch run and the first choices of the interactive picker.
- [x] `--strict-compat`: turn the integrity block off, so a read and a write stay inside the original specification of the format. The flag is off by default, because a reader that ignores the block still reads the cues
- [x] Locale selection from the active operating system locale. An unsupported locale falls back to `en-US`, this also means changing the default locale to `en-US` as it is a more sensible default for most users
  - The tool reads `LC_ALL`, `LC_MESSAGES`, and `LANG` in the order of the POSIX standard, and drops the codeset and the modifier from the value. The flag, the environment, and the file come first, in that order.
- [x] A parallel conversion flag that spreads the work across the CPU cores. The default is `max_cores - 2`, a caller can name a count, and zero uses every core
  - The flag is `--jobs` (`-j`), and the configuration file carries the same value under `jobs`. The workers never outnumber the planned outputs, and the progress bar stays on one goroutine. The failure report follows the walk order, so the same directory reports the same failure.
- [x] Public API freeze review. Deprecation notes for anything we cut.
  - Nothing was cut, so no name carries a deprecation note. The frozen surface and the rules for a later change live in [the library guide](docs/library.md).
- [x] Full doc sweep under `docs/` with the simple-english skill (CHECK mode pass)
  - The pass covers the prose of every page under `docs/`, the readme, and the contributor rules. No em-dash, no bare link alias, no prose semicolon, no banned modal, and no sentence or list item over the 25-word limit of rule 6.3 remains there. The pass split or rewrote 41 long sentences and the list items around them, most of them in the format notes, and it left every fact unchanged. The milestone lines of this roadmap keep the wording of their own author.
- [x] Compatibility notes: tested players and platforms matrix published
  - [The compatibility page](docs/compatibility.md) carries the matrix, the evidence beside each claim, and the limits of the evidence.
- [x] `goreleaser release --snapshot` produces installable artifacts on the six native targets
  - The snapshot writes six archives, a source archive, and a checksums file, and the Linux archive runs after it is unpacked. The `wasm` demo moves to v1.0.0, because the build has no main package yet. The configured build id stays skipped until then.
- [x] Signed release tags and changelog discipline verified from v0.1.0 onward
  - Every tag from v0.2.0 carries a signature, and the v0.8.0 signature verifies against the maintainer key. Every release from v0.1.0 has a changelog file, and the index links each one. The release tag itself is created with `make tag`, which suggests the highest changelog and writes the notes link into the tag message.

### v1.0.0: stable
- [x] A performance audit with the Go profiling tools (`pprof` and `go tool trace`) as well as `strace` and `perf`, over the parse, the render, and the conversion of every format, with the bottlenecks it finds recorded and either fixed or documented
  - [The performance audit page](docs/performance.md) records the tools, the machine, the numbers of every format, the bottlenecks, and the open items. The WebVTT entity replacer is fixed, so the WebVTT parse is about two and a half times faster and the render is about three times faster.
- [x] SemVer stability guarantee published for `pkg/sub` (breaking changes only at 2.0.0)
  - [The library guide](docs/library.md) freezes the surface of `pkg/sub` at v1.0.0 and names v2.0.0 as the only place for a breaking change.
- [x] Guilt-free WASM: `swag` compiles and runs core conversions in a browser demo page
  - `cmd/swag-wasm` exposes `swag.convert` to [the browser demo page](docs/demo/index.html), and the `js/wasm` build of [the GoReleaser configuration](.goreleaser.yml) no longer carries a skip.
- [x] All matrix cells shipped or explicitly deferred with an issue link
  - Every Core and Broader cell ships. FCPXML captions and SCC/CEA-608 stay deferred, and each one links its issue in [the architecture page](docs/architecture.md).
- [x] 1.0 release notes, migration guide from YTSubConverter workflows
  - [The v1.0.0 notes](docs/changelogs/v1.0.0.md) carry the release, and [the migration guide](docs/migration.md) maps the YTSubConverter commands onto `swag`.
- [x] Use colour and box styling sparingly, so a terminal theme stays in charge
  - The theme uses one accent colour for a heading and one for a name, and it draws no box and no background. The rules live in `cmd/swag/style.go`.
- [x] Integrate the pterm and kong styling into one look, so a bare run and `--help` match
  - `cmd/swag/style.go` renders the kong help page through the pterm theme through `kong.Help`, and the banner, the prompts, and the help share one palette. A test keeps each mark and the combined page.
- [x] The keybind table takes a token list, so a modifier can stand in any order, an upper-case letter means shift, a repeated modifier is dropped, and one binding can carry several keys
  - `internal/tui/keys.go` parses the tokens and builds a byte trie for the incremental matcher. [The configuration page](docs/configuration.md) records the notation, and the default file carries the built-in values as active entries.
- [x] The interactive mode reads a terminal one key at a time with a timeout, and keeps the line prompt for a pipe or a script
  - `internal/tui/reader.go` carries both paths, and `cmd/swag/keysource.go` provides the stream and terminal sources. A pause of 150 milliseconds ends a key sequence that carries no match.
- [x] A broken configuration file names itself, with tests for the syntax, the type, the duplicate key, and the unknown setting
  - `internal/config` wraps a decode failure with the file path, and the test suite covers each broken shape.
- [x] More performance audits, with benchmarks of the configuration decode and the keybind engine and a `make bench-all` target
  - [The performance audit page](docs/performance.md) records the numbers of the interactive path.
- [x] The full configuration chain: the command line flag, then the working directory file, then the global file, then the built-in default
  - `internal/config.Merge` layers the two files setting by setting, and the `keybinds` table merges action by action. `cmd/swag/config.go` reads both files, and the `config` command reports both paths. [The configuration page](docs/configuration.md) records the chain.
- [x] The active default file moves to the repository root as `swag.toml`, and a test keeps the embedded copy in step
  - [The configuration page](docs/configuration.md), [the documentation index](docs/index.md), and [the README](README.md) link the root file. Every fixed setting is active with its built-in default value, so writing or reading the file changes no behaviour.
- [x] The decoded configuration is cached by path, size, and modification time
  - A changed file is read again, and an unchanged path serves the kept result. [The performance audit](docs/performance.md) records the cached lookup at about half a microsecond against a decode near 40 microseconds.
- [x] Fuzz targets for the configuration decoder and the keybind token parser, wired into [the nightly fuzz workflow](.github/workflows/fuzz.yml)
  - The workflow now runs twelve targets, and [the testing notes](docs/testing.md) record the new properties.
- [x] More tests for the YouTube font fallback and the platform quirks
  - `internal/formats/ytt/quirks_test.go` pins the font table, the Roboto fallback, the small caps snap, the shadow space, the opacity ceiling, the white shift, the dark lift, and the scale round trip.
- [x] The readers reuse their span slices and skip the parser error on a text line, so the SubRip parse allocates half as many objects and the YouTube pair allocates fewer bytes
  - The SubRip span splitter appends into the cue, the counter test rejects a non-digit before the parser call, and the timestamp parsers find the colons instead of splitting. [The performance audit](docs/performance.md) carries the numbers.
- [x] The merged configuration result is cached, and the command line suite walks the whole precedence chain
  - `internal/config.LoadMerged` keys the merge on both paths and both file stamps. `TestPrecedenceChain` covers the flag, the working directory file, the global file, and the built-in default in one table.

### v1.1.0: performance and the smaller fixes

- [x] A writer path that pre-sizes its output slice, so a writer stops growing a slice many times
  - Every writer reserves its output buffer from the cue count, and [the audit](docs/performance.md) records the bytes.
- [x] A streaming envelope reader, so the plain readers hold no slice of the whole file
  - `envelope.Read` streams one line at a time into a handler. The SubRip and SBV readers use it, and the WebVTT reader keeps the whole list because its cue parser needs random access.
- [x] A reusable ruby grouping, so `model.RubyGroups` does not rebuild the backing slice for every cue
  - `model.RubyGroupsInto` writes into a slice that the writer owns, and every writer reuses it for the whole document.
- [x] A second full performance audit over the command with `hyperfine`, `strace`, `perf`, `pprof`, and `go tool trace`, and a fresh number for every open item
  - [The performance audit page](docs/performance.md) carries the second run. `make bench-tools` runs the three external tools over a shared fixture, and `make bench-audit` adds the Go profiles.

## Post-1.1 candidates

### Performance

- [ ] A hand-written configuration reader, so the first decode of a file does not spend its time in reflection and does not read the file twice. The decoder of BurntSushi/toml reads the whole file and then parses it, and this schema is small.

### Formats

- [ ] Survey the subtitle formats that the tool does not support, and add the ones that carry the most value. The candidates include SSA version 4, MicroDVD, SAMI, WebVTT regions and chapters, Universal Subtitle Format, and the broadcast formats below.
- [ ] SCC/CEA-608 writer on the 32-column grid
- [ ] FCPXML caption writer for NLE round-trips

### Interfaces

- [ ] Plugin registry for third-party formats (a Go interface and a registration hook)
- [ ] Simple subtitle editor: start terminal-native with pterm (cue list editor, style editor, live karaoke preview). Web (WASM and a light widget layer) after 1.0 if the terminal editor finds users. Native widget toolkit stays out of scope until then.

### Theming

- [ ] Theming for the command line, with the system theme as the default. The open question is how to read the palette of the terminal, because no portable interface exists. The candidates are the OSC 4, 10, and 11 queries, the `COLORTERM` and `TERM` variables, and a theme file. A query that gets no answer falls back to a plain theme, and every command keeps working without colour.
  - [The terminal palette page](docs/terminal-palette.md) records the findings and the plan. A prototype lives in `internal/tui/palette.go`. It sends the OSC 4, 10, and 11 queries. It reads the replies in the xterm `rgb:rrrr/gggg/bbbb` form and the `#rrggbb` form, and it returns a plain palette when the write fails or the timeout expires. The read runs on a goroutine with a timeout, because a terminal keeps its input open after the answers. The prototype settles part of the question: a terminal that implements the queries answers them, and a terminal that ignores them costs one short wait. It also names the next step. A command that calls the probe must first put the terminal in raw mode and set a read deadline. Without that step, the reading goroutine consumes the next line the user types. No command calls the prototype yet, so the theme of a run stays unchanged.
