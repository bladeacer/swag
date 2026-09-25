# swag roadmap to v1.0.0

`swag` (Subtitles With A Gopher) is a clean-room Go library and CLI for reading, writing, and converting subtitles. This document fixes the scope, the architecture, and the ordered milestones to v1.0.0. Agent contributors: tick a checkbox in the same change that completes the work behind it. Documentation follows the vendored `simple-english` skill with the British English override (see `AGENTS.md`).

We credit [YTSubConverter](https://github.com/arcusmaximus/YTSubConverter) as the source of inspiration for the feature set. We wrote all code from scratch. See `docs/third-party-notices.md`.

## How to read this document

1. Milestones are ordered. Do not start a later milestone before the earlier ones ship, unless a checkbox says otherwise.
2. Each milestone maps to a SemVer minor version. A milestone ships when every box is ticked, coverage stays at or above 75%, and the changelog carries the release notes.
3. The format support matrix and the ASS tag tiers below are the source of truth for scope. Update them in the same change that changes scope.

## Guiding decisions

- One intermediate representation (IR) sits between every reader and writer. Readers produce IR. Writers consume IR. No format talks to another format.
- The IR carries the union of the feature sets that the target formats express. A writer degrades features it cannot express, and records each loss in a conversion report.
- Lossy conversion is normal and visible. A writer must not guess silently.
- Clean-room rule: behaviour may match YTSubConverter, expression must be our own.
- The public library surface is `pkg/sub`. Everything else stays internal until a second consumer asks for it.

## Architecture

```
cmd/swag/                 CLI: kong flags, pterm output, locale selection
internal/converter/       Registry + Convert(from, to, opts) + loss report
internal/formats/         One package per format: Reader, Writer, Name()
  ├─ ytt/                 YouTube Timed Text (format 3)
  ├─ srv3/                YouTube SRV3 (XML with <pen>, window positions)
  ├─ ass/                 Advanced SubStation Alpha (+ tag parser)
  ├─ srt/                 SubRip
  ├─ sbv/                 YouTube SBV
  ├─ ttml/                TTML / DFXP
  ├─ vtt/                 WebVTT
  ├─ kdenlive/            Kdenlive subtitle module (JSON)
  ├─ json1/               FCPXML-capable JSON subtitle exchange
  └─ scc/                 Scenarist Closed Caption (stretch)
internal/model/           Core IR types: Cue, Style, TextSpan, colours,
                          plus style resolution helpers
internal/richtext/        ASS-style tag parser and serialiser (shared)
internal/i18n/            Message catalogue, locales (en-GB default)
pkg/sub/                  Public API: Identify, Parse, Render, Convert
```

## The intermediate representation

The IR splits timing (cue level) from rendering (span level). One cue holds spans, an optional layout, and an optional animation list.

```
model.Document    Metadata map[string]string, Styles []Style (at least one),
                  Cues []Cue, VideoDimensions Point
model.Cue         Start, End time.Duration (from media start),
                  Spans []TextSpan, Layout *Layout, Animations []Animation
model.Style       Name, Font string, Size float64,
                  Bold, Italic, Underline bool,
                  Primary, Secondary, Outline, Shadow Colour,
                  OutlineWidth, ShadowDepth float64,
                  Alignment Anchor, Box bool
model.TextSpan    Text string,
                  Start, End time.Duration (offsets from cue start;
                  both zero means untimed),
                  Font *string, Size *float64, Bold/Italic/Underline *bool,
                  Fore, Secondary, Back *Colour,
                  Shadows []Shadow, OutlineWidth *float64,
                  Vertical *Vertical, Script *Script, Direction *Direction,
                  Packed *bool, Ruby *Ruby (annotation spans only)
model.Shadow      Kind (SoftShadow | HardShadow | Bevel | Glow), Colour
model.Colour      R, G, B, A uint8
model.Layout      Position *Point, Anchor *Anchor, Fade *Fade, Move *Move
model.Vertical    Mode (None | ColumnsRTL | ColumnsLTR | RotatedCCW |
                  RotatedCCWReversed), Packed bool
model.Ruby        Position (Over | Under | Parenthetical)
model.Script      Kind (Regular | Subscript | Superscript)
model.Direction   (LeftToRight | RightToLeft)
model.Animation   one of: Fade{In,Out}, Move{From,To,Start,End},
                  Shake{RadiusX,RadiusY,Start,End},
                  Chroma{Offsets,InTime,OutTime},
                  Keyframes{Start,End,Easing,Steps},
                  Karaoke{Kind,Cursor}
model.Anchor      uint8 in numpad numbering: 1 = bottom-left, 5 = centre,
                  9 = top-right (ASS alignment values; YTT ap values map)
```

Rules:

1. Pointer and slice fields mean "the source set this". Nil or empty means "inherit from the style". Bare fields always carry a value, and readers fill their defaults. A `Colour` has no unset state, so an optional colour is a pointer.
2. Colours carry alpha everywhere. A format without alpha writes a loss note.
3. Karaoke: spans carry `Start`/`End` offsets from the cue start. A cue is a karaoke cue when any span has a non-zero offset. The unsung colour comes from `Style.Secondary`, or from `span.Secondary` when the source set it.
4. Ruby: a base span is followed immediately in `Spans` by one or more annotation spans that carry `Ruby`. The annotation inherits the timing of its base. Converters must not separate a base from its annotations, and render parenthetical ruby as bracketed text on formats without ruby support.
5. `Vertical`, `Script`, `Direction`, and `Packed` are span-level overrides. `Layout` carries cue-level placement: position, anchor, and motion.
6. Shadows are a list with at most one shadow per kind, in the fixed order soft, hard, bevel, glow. YTT carries one shadow per pen, so its writer layers duplicated lines. ASS carries one shadow, so its writer keeps the first shadow and records a loss note.
7. `Style` carries the defaults of a format: font, size, colours, widths, alignment, and the box flag (`BorderStyle` 3 in ASS). A reader always emits at least one style per document.
8. New fields use pointers or slices, so the zero value keeps its meaning and old readers stay valid.

## Format support matrix

Legend: R = read, W = write, ⊕ = with platform quirks applied on write.

| Format | Tier | Read | Write | Notes |
|---|---|---|---|---|
| YTT (YouTube Timed Text, format 3) | Core | R | W ⊕ | Pens, window positions, ruby, karaoke, vertical, shadow types |
| SRV3 (YouTube XML) | Core | R | W | Same model as YTT, older dialect |
| ASS / SSA | Core | R | W | Tag parser in `internal/richtext`, animation mapping |
| SRT | Core | R | W | Plain text, italic via `<i>`, loss notes for the rest |
| SBV | Core | R | W | Plain text |
| TTML / DFXP | Core | R | W | YouTube TTML dialect first, general TTML later |
| WebVTT | Broader | R | W | Voice spans, styling block, position cues |
| Kdenlive subtitle module | Broader | R | W | JSON, matches Kdenlive 24.12 import/export |
| JSON1 (exchange) | Broader | R | W | Our lossless interchange format for editors and pipelines |
| FCPXML captions | Later | — | W | Stretch goal, for NLE round-trips |
| SCC / CEA-608 | Later | — | W | Stretch goal, 32-column grid limits everything |

Platform quirks we deliberately reproduce on write (each with a test): YTT font allow-list snapping to Roboto, font scale re-mapping (scale = 1 + (ytt/100 − 1)/4), opacity ceilings at 254, white-shift to 0xFEFEFE, shadow clipping space-stealing, dark text brightening hack, zero-width-space padding sections, karaoke zero-duration bump, italic prefetch line, multi-shadow line layering.

## ASS feature tiers

Readers map every supported tag into the IR. Writers emit tags from the IR. Tier 1 and 2 cover the YTSubConverter tag set from its README. Tier 3 comes from the Aegisub tag manual (https://aegi.vmoe.info/docs/3.0/ASS_Tags/) where the IR can express it.

- Tier 1 (styling and timing): `\b`, `\i`, `\u`, `\fn`, `\fs`, `\c`/`\1c`, `\2c`, `\3c`, `\4c`, `\1a`–`\4a`, `\alpha`, `\k`, `\K`, `\kf`, `\ko`, `\r`, `\an`, `\pos`
- Tier 2 (effects and motion): `\move`, `\fad`, `\fade`, `\t`, `\ytshake`, `\ytchroma`, `\ytkt` variants (fade, glitch, cursor), `\ytsup`, `\ytsub`, `\ytsur`
- Tier 3 (CJK and direction): `\ytruby` (positions 2 and 8), `\ytvert` (1, 3, 7, 9), `\ytpack`, `\ytdir4`, `\ytdir6`
- Tier 4 (accepted, ignored with a note): tags outside the tiers that the IR cannot express, for example `\clip`, `\iclip`, `\bord` beyond width, `\be`, `\blur`, `\fr*`, `\org`, `\p`, drawing mode

Font allow-list (YouTube): Arial, Arial Black, Arial Narrow, Comic Sans MS, Courier New, Georgia, Impact, Roboto (default snap target), Tahoma, Times New Roman, Trebuchet MS, Verdana. Everything else snaps to Roboto on YTT/SRV3 write, with a loss note. `internal/formats/ytt/fonts.go` owns the single table.

## Milestones

### v0.1.0 — foundation
- [x] `go.mod` (module path `freebuff.dev/swag` or final repo path), Go 1.24 toolchain directive
- [x] `internal/model`: Colour (RGBA), Cue, TextSpan, Style, Layout types with constructors and zero-value semantics
- [x] Unit tests for colour parsing, rounding, and alpha edge cases
- [x] `.air.toml` for hot reload during development
- [x] `.goreleaser.yaml` v2: linux/darwin/windows (amd64, arm64) + js/wasm, `CGO_ENABLED=0`, `-s -w`, checksums (the wasm build id stays skipped until `cmd/swag-wasm` lands at v1.0.0)
- [x] `Makefile`: build, test, cover, vet, fmt, snapshot release
- [x] CI workflow: vet, test with coverage floor 75%, `goreleaser check`
- [x] `internal/i18n` skeleton with message catalogue and `en-GB`

### v0.2.0 — plain formats
- [x] `internal/formats/srt` reader and writer
- [x] `internal/formats/sbv` reader and writer
- [x] `pkg/sub.Identify` and `pkg/sub.Parse` with format auto-detection
- [x] Round-trip fixtures and tests for both formats
- [x] `cmd/swag` with kong (`-i`, `-o`, `-f`, `--verbose`) and pterm banner, warnings, and result output
- [x] Conversion report (list of feature losses) printed with pterm when `--verbose`

### v0.3.0 — the YouTube pair
- [x] `internal/formats/ytt` reader: pens, window positions/styles, ruby groups, karaoke offsets
- [x] `internal/formats/ytt` writer: pen deduplication, the full quirk pipeline, multi-shadow layering
- [x] `internal/formats/srv3` reader and writer sharing the pen model
- [x] Font allow-list table and scale re-mapping with tests
- [x] Karaoke timing model tests (zero-duration bump, offset ordering)
- [x] Sample-driven tests from real YTT/SRV3 fixtures
- [x] Pull sample files from upstream YTSubConverter repo for e2e and other testing purposes
- [x] Adapt ../ocd coverage.svg rendering, place code coverage badge and other badges in README. Use exaact badge format and placement in README file as ../ocd.
- [x] Create .goreleaser.yml configuration file like ../ocd,  goreleaser in CI should push release when running make tag, not just create the tag and draft release.

### v0.4.0 — ASS reader
- [x] `internal/richtext`: tag lexer, escape resolution, tag argument grammar
- [x] Script Info + V4+ Styles + Events parsing (PlayRes, WrapStyle, Collisions)
- [x] Style-to-IR mapping with the Default-style size baseline rule
- [x] Tier 1 tags; karaoke spans with secondary colour handling
- [x] Tier 3 tags: ruby, vertical, packed, direction
- [x] Tier 2 animations: `\fad`, `\fade`, `\move`, `\t`, shake, chroma, karaoke types
- [x] Fixtures: karaoke sample and colour sample in the YTSubConverter style, written fresh for this project

### v0.5.0 — ASS writer
- [x] IR-to-tag emission for tiers 1 to 3
- [x] Outline, shadow, and box (BorderStyle) emission with alpha
- [x] Animation emission and degradation notes where YouTube limits apply
- [x] Round-trip test: ASS → IR → ASS stays semantically equal (colour and timing equality, not byte equality)
- [x] Cross-check suite: ASS → YTT → IR → ASS loses only documented features

### v0.6.0 — TTML, WebVTT, editor formats
- [ ] `internal/formats/ttml`: YouTube dialect reader, general writer
- [ ] `internal/formats/vtt`: reader and writer with voice spans and styling
- [ ] `internal/formats/kdenlive`: Kdenlive subtitle JSON reader and writer
- [ ] `internal/formats/json1`: lossless internal exchange format, versioned
- [ ] Loss-report review: every matrix cell has a documented degradation

### v0.7.0 — library hardening
- [ ] `pkg/sub.Convert` stable API with options (style mapping, font policy, loss tolerance)
- [ ] Fuzzing for all readers (`go test -fuzz` targets, 30-minute runs in CI nightly)
- [ ] Benchmarks for large files (10k cues) with regression tracking
- [ ] 75%+ coverage verified per package, not just module-wide
- [ ] i18n coverage for every CLI message; second locale lands as proof
- [ ] Docs: `docs/formats.md` per-format notes, `docs/library.md` usage guide

### v0.8.0 — terminal UI
- [ ] `swag interactive`: pterm interactive mode (pick input, detect format, pick target, show report)
- [ ] Batch conversion (`swag convert dir/`) with pterm progress bars and multi-writer output
- [ ] Colour and style previews rendered in the terminal (ANSI, best effort)
- [ ] Karaoke timeline preview in the terminal

### v0.9.0 — release candidate
- [ ] Public API freeze review; deprecation notes for anything we cut
- [ ] Full doc sweep under `docs/` with the simple-english skill (CHECK mode pass)
- [ ] Compatibility notes: tested players/platforms matrix published
- [ ] `goreleaser release --snapshot` produces installable artifacts on all 7 targets, wasm demo included
- [ ] Signed release tags and changelog discipline verified from v0.1.0 onward

### v1.0.0 — stable
- [ ] SemVer stability guarantee published for `pkg/sub` (breaking changes only at 2.0.0)
- [ ] Guilt-free WASM: `swag` compiles and runs core conversions in a browser demo page under `docs/demo/`
- [ ] All matrix cells shipped or explicitly deferred with an issue link
- [ ] 1.0 release notes, migration guide from YTSubConverter workflows

## Stretch goals (post-1.0 candidates)

- [ ] Simple subtitle editor: start terminal-native with pterm (cue list editor, style editor, live karaoke preview). Web (WASM + a light widget layer) after 1.0 if the terminal editor finds users. Native widget toolkit stays out of scope until then.
- [ ] SCC/CEA-608 writer on the 32-column grid
- [ ] FCPXML caption writer for NLE round-trips
- [ ] Plugin registry for third-party formats (Go interface + registration hook)
