# Architecture

`swag` (Subtitles With A Gopher) reads a subtitle file into one intermediate representation (IR) and writes it out in another format. This page fixes the architecture, the IR, the format matrix, and the ASS tag tiers. The ordered milestones live in [the roadmap](../ROADMAP.md).

## Guiding decisions

- One intermediate representation (IR) sits between every reader and writer. Readers produce IR. Writers consume IR. No format talks to another format.
- The IR carries the union of the feature sets that the target formats express. A writer degrades features it cannot express, and records each loss in a conversion report.
- Lossy conversion is normal and visible. A writer must not guess silently.
- Clean-room rule: behaviour can match YTSubConverter, expression must be our own.
- The public library surface is `pkg/sub`. Everything else stays internal until a second consumer asks for it.

## Packages

```
cmd/swag/                 CLI: kong flags, pterm output, locale selection
internal/converter/       Registry + Convert(from, to, opts) + loss report
internal/formats/         One package per format: Reader, Writer, Name()
  |-- ytt/                YouTube Timed Text (format 3)
  |-- srv3/               YouTube SRV3 (XML with <pen>, window positions)
  |-- ass/                Advanced SubStation Alpha (+ tag parser)
  |-- srt/                SubRip
  |-- sbv/                YouTube SBV
  |-- ttml/               TTML / DFXP
  |-- vtt/                WebVTT
  |-- kdenlive/           Kdenlive subtitle module (JSON)
  |-- json1/              FCPXML-capable JSON subtitle exchange
  `-- scc/                Scenarist Closed Caption (stretch)
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
4. Ruby: one or more annotation spans that carry `Ruby` come immediately after their base span in `Spans`. An annotation inherits the timing of its base. Converters must not separate a base from its annotations, and render parenthetical ruby as bracketed text on formats without ruby support.
5. `Vertical`, `Script`, `Direction`, and `Packed` are span-level overrides. `Layout` carries cue-level placement: position, anchor, and motion.
6. Shadows are a list with at most one shadow per kind, in the fixed order soft, hard, bevel, glow. YTT carries one shadow per pen, so its writer layers duplicated lines. ASS carries one shadow, so its writer keeps the first shadow and records a loss note.
7. `Style` carries the defaults of a format: font, size, colours, widths, alignment, and the box flag (`BorderStyle` 3 in ASS). A reader always emits at least one style per document.
8. New fields use pointers or slices, so the zero value keeps its meaning and old readers stay valid.

## Format support matrix

Legend: R = read, W = write, ⊕ = with the platform quirks that the writer applies.

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
| FCPXML captions | Later | - | W | Stretch goal, for NLE round-trips |
| SCC / CEA-608 | Later | - | W | Stretch goal, 32-column grid limits everything |

We deliberately reproduce these platform quirks on write, each with a test:

- YTT font allow-list snapping to Roboto
- Font scale re-mapping, where `scale = 1 + (ytt/100 - 1)/4`
- Opacity ceilings at 254
- White shift to `0xFEFEFE`
- Shadow clipping space-stealing
- Dark text brightening
- Zero-width-space padding sections
- Karaoke zero-duration bump
- Italic prefetch line
- Multi-shadow line layering

## ASS feature tiers

Readers map every supported tag into the IR. Writers emit tags from the IR. Tier 1 and 2 cover the YTSubConverter tag set from its README. Tier 3 comes from the Aegisub tag manual at [the Aegisub ASS tag reference](https://aegi.vmoe.info/docs/3.0/ASS_Tags/) where the IR can express it.

- Tier 1 (styling and timing): `\b`, `\i`, `\u`, `\fn`, `\fs`, `\c`/`\1c`, `\2c`, `\3c`, `\4c`, `\1a` to `\4a`, `\alpha`, `\k`, `\K`, `\kf`, `\ko`, `\r`, `\an`, `\pos`
- Tier 2 (effects and motion): `\move`, `\fad`, `\fade`, `\t`, `\ytshake`, `\ytchroma`, `\ytkt` variants (fade, glitch, cursor), `\ytsup`, `\ytsub`, `\ytsur`
- Tier 3 (CJK and direction): `\ytruby` (positions 2 and 8), `\ytvert` (1, 3, 7, 9), `\ytpack`, `\ytdir4`, `\ytdir6`
- Tier 4 (accepted, ignored with a note): tags outside the tiers that the IR cannot express, for example `\clip`, `\iclip`, `\bord` beyond width, `\be`, `\blur`, `\fr*`, `\org`, `\p`, drawing mode

Font allow-list (YouTube): Arial, Arial Black, Arial Narrow, Comic Sans MS, Courier New, Georgia, Impact, Roboto (default snap target), Tahoma, Times New Roman, Trebuchet MS, Verdana. Everything else snaps to Roboto on YTT/SRV3 write, with a loss note. `internal/formats/ytt/fonts.go` owns the single table.
