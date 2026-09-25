# Loss report review

Every writer in `swag` returns a loss report: one entry per feature the writer cannot express. The conversion is lossy when the report is not empty. This page records the degradation of every format in [the format support matrix](architecture.md), so a reader knows what a target drops before the conversion runs.

The lists come from the writer code. [The cross-check suite](../internal/formats/ass/crosscheck_test.go) runs an ASS fixture through every shipped writer and asserts the report of each target, so a list that drifts from the code fails a test.

| Format | Registry name | Features the writer reports |
|---|---|---|
| Advanced SubStation Alpha | `ass` | A shadow list with more than one non-glow shadow, a karaoke gap, a chroma with more than one offset, a keyframe with no value, a cursor karaoke type with no text, a ruby base with more than one reading, and a voice name |
| YouTube Timed Text | `ytt` | Cue fade, cue move, fade, move, shake, chroma, keyframe, and karaoke-type animations, a strikeout run, a glyph scale, a voice name, a font outside the YouTube list, and a font size that clamps to the 75 percent minimum |
| YouTube SRV3 | `srv3` | The same list as YTT, because the two formats share one writer |
| SubRip | `srt` | Karaoke timing, positioning, animation, ruby text, vertical text, a script offset, transparency, foreground colour, shadow effects, a right-to-left marking, a strikeout run, a glyph scale, and a voice name |
| YouTube SBV | `sbv` | Karaoke timing, positioning, inline styling, a glyph scale, ruby text, and a voice name |
| WebVTT | `vtt` | Karaoke timing, a cue fade, a cue move, a vertical alignment, an animation, a foreground, secondary, or background colour, a font, a font size, a strikeout run, a glyph scale, a shadow, an outline width, vertical text, a script offset, a right-to-left marking, packing, and ruby text |
| TTML / DFXP | `ttml` | A cue fade, a cue move, an animation, a glyph scale, a shadow, vertical text, packing, a right-to-left marking, a script offset, ruby text, and a voice name |
| Kdenlive subtitle JSON | `kdenlive` | Karaoke timing, positioning, an animation, inline styling, a glyph scale, vertical text, packing, a right-to-left marking, a script offset, ruby text, and a voice name |
| Lossless JSON exchange | `json1` | Nothing. The format carries the whole IR, so it degrades no feature |

Three rules cut across the table.

- A format without ruby support writes the reading in brackets after the base span, so the reading is never lost. The writer still reports the loss, because the rendering changed.
- A format without an alpha channel reports the transparency of a colour, because it drops that channel.
- The ASS writer reports the avoidable cases only. A chroma whose offsets form the symmetric spread and a blank `\ytvert` reset carry no loss, because the writer rebuilds them exactly.
- WebVTT carries a voice name, so it is the one non-lossless format that keeps it without the integrity block. A writer that reports a voice name drops it, and [the integrity block](integrity.md) of SubRip and SBV keeps it.
