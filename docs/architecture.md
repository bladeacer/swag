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
internal/formats/         One package per format: Reader, Writer, Name()
  |-- ytt/                YouTube Timed Text (format 3)
  |-- srv3/               YouTube SRV3 (XML with <pen>, window positions)
  |-- ass/                Advanced SubStation Alpha (+ tag parser)
  |-- srt/                SubRip
  |-- sbv/                YouTube SBV
  |-- ttml/               TTML / DFXP (YouTube dialect reader, general writer)
  |-- vtt/                WebVTT
  |-- kdenlive/           Kdenlive subtitle track JSON
  `-- json1/              Lossless internal exchange format, versioned
internal/envelope/        The integrity block of a plain format: read the
                          lines, split off the block, write the block
internal/model/           Core IR types: Cue, Style, TextSpan, colours,
                          plus style resolution helpers
internal/richtext/        ASS-style tag parser and serialiser (shared)
internal/i18n/            Message catalogue, the system locale, and the two English locales
pkg/sub/                  Public API: Identify, Parse, Render, Convert,
                          ConvertWith, and the format registry
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
                  Strikeout *bool, ScaleX/ScaleY *float64,
                  Fore, Secondary, Back *Colour,
                  Shadows []Shadow, OutlineWidth *float64,
                  ShadowDepth *float64,
                  Vertical *Vertical, Script *Script, Direction *Direction,
                  Packed *bool, Voice *string (the speaker of the span),
                  Ruby *Ruby (annotation spans only)
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
                  Chroma{Offsets,InTime,OutTime,Colours,Alpha},
                  Keyframes{Start,End,Easing,Steps},
                  Karaoke{Kind,Cursor,CursorTags,CursorLeft,
                          CursorInterval,CursorFrames}
model.Anchor      uint8 in numpad numbering: 1 = bottom-left, 5 = centre,
                  9 = top-right (ASS alignment values; YTT ap values map)
```

Rules:

1. Pointer and slice fields mean "the source set this". Nil or empty means "inherit from the style". Bare fields always carry a value, and readers fill their defaults. A `Colour` has no unset state, so an optional colour is a pointer.
2. Colours carry alpha everywhere. A format without alpha writes a loss note.
3. Karaoke: spans carry `Start`/`End` offsets from the cue start. A cue is a karaoke cue when any span has a non-zero offset. The unsung colour comes from `Style.Secondary`, or from `span.Secondary` when the source set it.
4. Ruby: one or more annotation spans that carry `Ruby` come immediately after their base span in `Spans`. An annotation inherits the timing of its base. Converters must not separate a base from its annotations, and render parenthetical ruby as bracketed text on formats without ruby support.
5. `Vertical`, `Script`, `Direction`, and `Packed` are span-level overrides. `Layout` carries cue-level placement: position, anchor, and motion.
6. Shadows are a list with at most one shadow per kind, in the fixed order soft, hard, bevel, glow. YTT carries one shadow per pen, so its writer layers duplicated lines. ASS carries one outline channel and one shadow channel. A glow writes through the outline channel, and any other kind writes through the shadow channel. A list with more than one non-glow shadow records a loss.
7. `Style` carries the defaults of a format: font, size, colours, widths, alignment, and the box flag (`BorderStyle` 3 in ASS). A reader always emits at least one style per document.
8. New fields use pointers or slices, so the zero value keeps its meaning and old readers stay valid.

9. `Voice` names the speaker of a span. WebVTT carries it in a `<v Speaker>` tag, JSON1 and the integrity block keep it, and a format without a voice form records a loss.

## Plain format integrity

SubRip and SBV hold text and timing only, so a conversion through one of them loses every other feature. A plain writer that loses at least one feature appends an integrity block to the end of the file. The block holds the whole document as JSON1. A `swag` reader restores the document from the block, and a plain player stops at the last cue and ignores it.

```
NOTE swag-ir 1
<base64 of a JSON1 document>
```

The block is a versioned extension of this project. A document that fits in the plain format writes no block. A plain file then stays plain, and a hand-written plain file reads exactly as before. [The file integrity page](integrity.md) covers the rules and the limits.

JSON1 itself is versioned, and the reader lifts an older file to the current shape before it returns the document. The chain of steps lives in [the JSON1 package](../internal/formats/json1/json1.go), and the reader reports a version that no step reaches. [The JSON1 page](json1.md) records the version history and the rules for a new step.

## Format support matrix

Legend: R = read, W = write, ⊕ = with the platform quirks that the writer applies.

| Format | Tier | Read | Write | Notes |
|---|---|---|---|---|
| YTT (YouTube Timed Text, format 3) | Core | R | W ⊕ | Pens, window positions, ruby, karaoke, vertical, shadow types |
| SRV3 (YouTube XML) | Core | R | W | Same model as YTT, older dialect |
| ASS / SSA | Core | R | W | Tag parser in `internal/richtext`, animation mapping |
| SRT | Core | R | W | Plain text, italic via `<i>`, loss notes for the rest |
| SBV | Core | R | W | Plain text |
| TTML / DFXP | Core | R | W | YouTube TTML dialect reader, general TTML writer |
| WebVTT | Broader | R | W | Signature, cue ids, note blocks, align and position settings, inline tags |
| Kdenlive subtitle JSON | Broader | R | W | The subtitle track JSON of Kdenlive |
| JSON1 (exchange) | Broader | R | W | Our lossless interchange format for editors and pipelines |
| FCPXML captions | Later | - | Deferred | For NLE round-trips, tracked by [the FCPXML issue](https://github.com/bladeacer/swag/issues?q=FCPXML) |
| SCC / CEA-608 | Later | - | Deferred | The 32-column grid limits everything, tracked by [the SCC issue](https://github.com/bladeacer/swag/issues?q=SCC) |

Every read cell and every write cell of a Core or Broader format ships in v1.0.0. Two cells stay deferred, and each one carries a link to its issue: [the FCPXML issue](https://github.com/bladeacer/swag/issues?q=FCPXML) and [the SCC issue](https://github.com/bladeacer/swag/issues?q=SCC). [The roadmap](../ROADMAP.md) holds them as post-1.1 candidates.

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

Readers map every supported tag into the IR. Writers emit tags from the IR. Tier 1 and 2 cover the YTSubConverter tag set from its README, plus the Aegisub tags that the IR can express. Tier 3 comes from [the ASS override tag reference](https://aegisub.org/docs/latest/ass_tags/). [The ASS support page](ass-support.md) maps each tag onto its IR field and its test.

- Tier 1 (styling): `\b`, `\i`, `\u`, `\s`, `\fn`, `\fs`, `\fscx`, `\fscy`, `\c`/`\1c`, `\2c`, `\3c`, `\4c`, `\1a` to `\4a`, `\alpha`, `\bord`, `\shad`/`\xshad`/`\yshad`
- Tier 1 (karaoke and layout): `\k`, `\K`, `\kf`, `\ko`, `\r`, `\an`, `\a`, `\pos`
- Tier 2 (effects and motion): `\move`, `\fad`, `\fade`, `\t`, `\ytshake`, `\ytchroma` (including the custom colour and alpha form), `\ytsup`, `\ytsub`, `\ytsur`
- Tier 2 (karaoke types): the `\ytkt` variants, which are fade, glitch, and the cursor forms with side, tags, and frames
- Tier 3 (CJK and direction): `\ytruby` (positions 2 and 8), `\ytvert` (1, 3, 7, 9), `\ytpack`, `\ytdir4`, `\ytdir6`
- Tier 4 (accepted, ignored with a note): tags outside the tiers that the IR cannot express
- Tier 4 (examples): `\clip`, `\iclip`, `\be`, `\blur`, `\fsp`, `\fr*`, `\org`, `\p`, and drawing mode

Font allow-list (YouTube): Arial, Arial Black, Arial Narrow, Comic Sans MS, Courier New, Georgia, Impact, Tahoma, Times New Roman, Trebuchet MS, and Verdana. Roboto is the default snap target. Everything else snaps to Roboto on YTT/SRV3 write, with a loss note. `internal/formats/ytt/fonts.go` owns the single table.
