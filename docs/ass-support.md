# ASS support

`swag` reads Advanced SubStation Alpha (ASS) into the intermediate representation (IR) and writes it back. [The format notes](formats.md) give the format matrix, and [the architecture page](architecture.md) gives the IR and the tag tiers.

This page maps the ASS feature list of [YTSubConverter](https://github.com/arcusmaximus/YTSubConverter) onto the swag implementation. Each row names the IR field, the reader path, the writer path, and the test that covers it. [The Aegisub ASS tag reference](https://aegi.vmoe.info/docs/3.0/ASS_Tags/) covers the extra tags that swag reads where the IR can express them.

The reader maps each style onto `model.Style` and flattens the cue style onto the spans of the cue. The base style is the one named `Default`, or the first style when no `Default` exists. The base style is also the size baseline. The YTT writer measures every other size against it.

## Style features

| Feature | IR field | Reader | Writer | Test |
|---|---|---|---|---|
| Font name | `Style.Font`, `TextSpan.Font` | `Fontname`, `\fn` | style table, `\fn` | `TestParseStyleFlattening` |
| Font size | `Style.Size`, `TextSpan.Size` | `Fontsize`, `\fs` | style table, `\fs` | `TestParseInlineStyling` |
| Bold | `Style.Bold`, `TextSpan.Bold` | `Bold`, `\b` | style table, `\b` | `TestParseInlineStyling` |
| Italic | `Style.Italic`, `TextSpan.Italic` | `Italic`, `\i` | style table, `\i` | `TestParseMidSyllableColour` |
| Underline | `Style.Underline`, `TextSpan.Underline` | `Underline`, `\u` | style table, `\u` | `TestParseInlineStyling` |
| Primary colour | `Style.Primary`, `TextSpan.Fore` | `PrimaryColour`, `\c` or `\1c` | style table, `\c` | `TestParseInlineStyling` |
| Secondary colour | `Style.Secondary`, `TextSpan.Secondary` | `SecondaryColour`, `\2c` | style table, `\2c` | `TestParseSecondaryColour` |
| Outline colour | `Style.Outline`, `TextSpan.Back` | `OutlineColour`, `\3c` | `\3c` | `TestParseStyleFlattening` |
| Shadow colour | `Style.Shadow`, `TextSpan.Shadows` | `BackColour`, `\4c` | `\4c`, or `\3c` for a glow | `TestWriterEmitsTierOneToThreeTags` |
| Transparency | `model.Colour.A` | `\1a` to `\4a`, `\alpha` | `\1a` to `\4a` | `TestParseTransparency` |
| Outline thickness | `Style.OutlineWidth`, `TextSpan.OutlineWidth` | `Outline`, `\bord` | style table, `\bord` | `TestParseStyleFlattening` |
| Shadow distance | `Style.ShadowDepth`, `TextSpan.ShadowDepth` | `Shadow`, `\shad` | style table, `\shad` | `TestParseShadowAndAlignmentOverrides` |
| Alignment | `Style.Alignment`, `Layout.Anchor` | `Alignment`, `\an` | style table, `\an` | `TestParseLayoutTags` |
| Background box | `Style.Box` | `BorderStyle` 3 | `BorderStyle` 3 | `TestParseBoxStyle` |

## Override tags from the YTSubConverter list

| Tag | Effect | IR field | Reader | Writer | Test |
|---|---|---|---|---|---|
| `\b` | bold | `TextSpan.Bold` | yes | `\b` | `TestParseInlineStyling` |
| `\i` | italic | `TextSpan.Italic` | yes | `\i` | `TestParseMidSyllableColour` |
| `\u` | underline | `TextSpan.Underline` | yes | `\u` | `TestParseInlineStyling` |
| `\fn` | font name | `TextSpan.Font` | yes | `\fn` | `TestParseInlineStyling` |
| `\fs` | font size | `TextSpan.Size` | yes | `\fs` | `TestParseInlineStyling` |
| `\c`, `\1c` | primary colour | `TextSpan.Fore` | yes | `\c` | `TestParseInlineStyling` |
| `\2c` | unsung karaoke colour | `TextSpan.Secondary` | yes | `\2c` | `TestParseSecondaryColour` |
| `\3c` | outline or box colour | `TextSpan.Back` | yes | `\3c` | `TestWriterEmitsTierOneToThreeTags` |
| `\4c` | shadow colour | `TextSpan.Shadows` | yes | `\4c` | `TestWriterEmitsTierOneToThreeTags` |
| `\1a` to `\4a` | per channel transparency | `model.Colour.A` | yes | matching tag | `TestParseTransparency` |
| `\alpha` | all transparencies | `model.Colour.A` | yes | no, the writer sets each channel | `TestParseTransparency` |
| `\pos` | position | `Layout.Position` | yes | `\pos` | `TestParseLayoutTags` |
| `\an` | alignment | `Layout.Anchor` | yes | `\an` | `TestParseLayoutTags` |
| `\k`, `\K`, `\kf`, `\ko` | karaoke timing | `TextSpan.Start`, `TextSpan.End` | yes | `\k` | `TestParseKaraokeTiming` |
| `\r` | style reset | every style field | yes | `\r` | `TestParseStyleReset` |
| `\fad` | simple fade | `Layout.Fade` | yes | `\fad` | `TestParseAnimations` |
| `\fade` | complex fade | `Layout.Fade` | yes | `\fade` | `TestComplexFade` |
| `\move` | motion | `Layout.Move` | yes | `\move` | `TestParseLayoutTags` |
| `\t` | animation | `Animation.Keyframes` | yes | `\t` | `TestParseAnimations` |
| `\ytsub`, `\ytsup`, `\ytsur` | script offset | `TextSpan.Script` | yes | matching tag | `TestParseScriptTags` |
| `\ytruby` | ruby annotation | `TextSpan.Ruby` | yes | `\ytruby`, `\ytruby2` | `TestParseRuby` |
| `\ytvert` | vertical text | `TextSpan.Vertical` | yes | `\ytvert` | `TestParseVerticalAndPacked` |
| `\ytpack` | packing | `TextSpan.Packed` | yes | `\ytpack1`, `\ytpack0` | `TestParseVerticalAndPacked` |
| `\ytdir4`, `\ytdir6` | reading direction | `TextSpan.Direction` | yes | matching tag | `TestParseDirection` |
| `\ytshake` | shake | `Animation.Shake` | yes | `\ytshake` | `TestShakeArguments` |
| `\ytchroma` | chromatic aberration | `Animation.Chroma` | yes | `\ytchroma` | `TestChromaArguments` |
| `\ytktFade`, `\ytktGlitch` | karaoke type | `Animation.Karaoke` | yes | matching tag | `TestKaraokeTypes` |
| `\ytkt` cursor | cursor type | `Animation.Karaoke` | yes | `\ytkt` | `TestParseCursorForms` |

## Extra Aegisub overrides

The reader and the writer also handle these tags. They come from [the Aegisub ASS tag reference](https://aegi.vmoe.info/docs/3.0/ASS_Tags/), and not from the YTSubConverter list.

| Tag | Effect | IR field | Reader | Writer | Test |
|---|---|---|---|---|---|
| `\bord` | outline width | `TextSpan.OutlineWidth` | yes | `\bord` | `TestWriterEmitsTierOneToThreeTags` |
| `\shad`, `\xshad`, `\yshad` | shadow distance | `TextSpan.ShadowDepth` | yes | `\shad` | `TestParseShadowAndAlignmentOverrides` |
| `\a` | legacy alignment | `Layout.Anchor` | yes | no, the writer uses `\an` | `TestParseShadowAndAlignmentOverrides` |
| `\s` | strikeout | `TextSpan.Strikeout` | yes | `\s` | `TestParseStrikeoutAndScale` |
| `\fscx`, `\fscy` | glyph scale | `TextSpan.ScaleX`, `TextSpan.ScaleY` | yes | matching tag | `TestParseStrikeoutAndScale` |

The `\xshad` and `\yshad` tags set the one shadow distance of the IR, so the tag that appears last wins when both appear. The `\a` tag maps the SSA row numbering onto the same numpad anchors as `\an`. The writer always emits `\an`.

## YouTube limits

The YouTube player and the upload path constrain several features. [The format notes](formats.md) describe the quirks that the YTT writer applies. The ASS reader and writer keep the values, and the format notes name each case where a write reports a loss.

- A font outside the YouTube allow-list snaps to Roboto.
- The font scale remaps through the virtual percentage, so the real factor is `1 + (sz/100 - 1)/4`. A factor below 0.75 clamps to the minimum.
- A style size other than the base style is relative to the base style. The Android app ignores the size.
- The player checks whether the outline thickness and the shadow distance are zero or greater than zero. The exact values do not carry.
- A shadow and an outline fade with the text only when their colour is `&H222222&`.
- `\4a` takes effect only when the shadow colour is `&H222222&` and the shadow transparency equals the text transparency.
- The alignment moves a subtitle on a mouseover. A top-aligned subtitle moves down, a centre-aligned subtitle stays, and a bottom-aligned subtitle moves up.
- `\ytruby`, `\ytvert`, and `\ytpack` work on the PC player only. A mobile app shows the parenthetical fallback for ruby text.
- `\ytdir4` marks a right-to-left run inside a left-to-right subtitle. YouTube sets the direction for a right-to-left upload, so the tag is not needed there.

## Deliberate losses

The ASS writer reports a loss for each feature that the format cannot express. The list follows.

- A shadow list with more than one non-glow shadow, because ASS carries one shadow channel. A hard shadow, a soft shadow, and a bevel write through the shadow colour of the span, and a glow writes through the border colour.
- A karaoke gap between two segments.
- A chroma with more than one offset, when the offsets do not form the spread that the argument form rebuilds.
- A keyframe with no animated value.
- A ruby base with more than one reading.
- A cursor karaoke type with no text.

The writer keeps a chroma whose offsets form the symmetric spread that the argument form rebuilds, so a chroma that came from ASS round-trips without a loss. A blank `\ytvert` returns a run to horizontal text, so a run can leave vertical text without a loss.

## Tests

The reader tests live in `internal/formats/ass/ass_test.go` and `internal/formats/ass/tags_test.go`. The writer tests live in `internal/formats/ass/writer_test.go`. The round-trip test walks each fixture under `internal/formats/ass/testdata`. The cross-check suite in `internal/formats/ass/crosscheck_test.go` runs one ASS source through every shipped writer. A chain suite runs one fixture per reader through a sequence of two or more formats and back to the reader, so the readers and the writers compose. The plain writers report a strikeout run and a glyph scale as losses, so the conversion report stays complete.
