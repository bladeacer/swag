# Formats

`swag` reads a subtitle file into one intermediate representation (IR) and writes it out in another format. A reader turns its own concepts into IR types. A writer turns IR types into its own concepts and reports every feature it cannot express.

Legend: R reads, W writes.

| Format | Registry name | Extensions | Support |
|---|---|---|---|
| SubRip | `srt` | `srt` | R, W |
| YouTube SBV | `sbv` | `sbv` | R, W |
| YouTube Timed Text | `ytt` | `ytt` | R, W |
| YouTube SRV3 | `srv3` | `srv3` | R, W |
| Advanced SubStation Alpha | `ass` | `ass`, `ssa` | R, W |
| WebVTT | `vtt` | `vtt` | R, W |
| TTML / DFXP | `ttml` | `ttml`, `dfxp` | R, W |
| Kdenlive subtitle JSON | `kdenlive` | `kdenlive` | R, W |
| Lossless JSON exchange | `json1` | `json1` | R, W |

A writer returns a loss report, which is one entry per degraded feature. The command line interface prints the report when `-v` is set. [The loss report review](loss-report.md) gives the degradation of every format at a glance.

## SubRip (srt)

SubRip carries plain text. It reads a cue counter, a timing line of the form `hh:mm:ss,mmm --> hh:mm:ss,mmm`, and one or more text lines. Two inline tags are understood: `<i>` and `<b>`.

The writer drops everything else. It reports karaoke timing, positioning, animations, foreground colour, transparency, shadows, vertical text, script offsets, right-to-left marking, ruby text, strikeout, and glyph scale. Ruby text falls back to bracketed readings.

## YouTube SBV (sbv)

SBV carries plain text only. It reads a timing line of the form `h:mm:ss.mmm,h:mm:ss.mmm` and one or more text lines.

The writer drops styling, karaoke, positioning, glyph scale, and a strikeout run. Ruby text falls back to bracketed readings.

## YouTube Timed Text (ytt) and SRV3 (srv3)

YouTube Timed Text is XML. A `<pen>` element carries the style of a text run, and a `<ws>` element carries the window of a line. A `<wp>` element carries a named window position. The body holds `<p>` lines, each with `<s>` runs whose `t` attribute gives karaoke timing. SRV3 is an older dialect of the same document, so both formats share one reader, one writer, and one pen model.

The reader maps pens onto span overrides, window positions onto the cue layout, and window styles onto the vertical mode or the direction. It pairs ruby base spans with their readings and drops the parenthesis fallback. It turns the relative appearance time of each run into karaoke span offsets.

The writer deduplicates pens and lists them in increasing id order. It applies the platform quirks of the YouTube upload path, because the player and the upload path reject or rewrite several values:

- A font outside the YouTube list snaps to Roboto, and the writer reports the loss.
- The font scale remaps through the virtual percentage of the format, where the real factor is `1 + (sz/100 - 1)/4`. A factor below 0.75 clamps to the minimum, and the writer reports it.
- An alpha value of 255 becomes 254, because the upload drops an attribute with the maximum value.
- A pure white foreground becomes `#FEFEFE`, because the Android client ignores `#FFFFFF`.
- A pure black foreground becomes `#010101`, because the upload drops a foreground that matches the default background.
- A line with more than one run gets a zero-width space after the first run. Without that space, the upload removes the pen of the first run.
- A karaoke segment with no length gains one millisecond, and the segments stay in order.
- An italic or shadowed line start gains one space, so the player does not clip the overhang.
- A cue with more than one shadow kind is layered, one line per kind, because a pen carries one edge.

The writer reports the effects it cannot express: cue fades, cue moves, the shake, chroma, keyframe, and karaoke-type animations, a strikeout run, and a glyph scale.

## Advanced SubStation Alpha (ass)

ASS carries a Script Info section, a V4+ Styles section, and an Events section. The reader maps each style onto an IR style. It flattens the style of a cue onto the spans of that cue, because the IR carries one document style and per-span overrides.

The reader supports the Tier 1 tags (`\b`, `\i`, `\u`, `\s`, `\fn`, `\fs`, `\fscx`, `\fscy`, the colour tags, the alpha tags, `\bord`, `\shad`, `\a`, `\k` and its variants, `\r`, `\an`, and `\pos`), the Tier 2 effects (`\move`, `\fad`, `\fade`, `\t`, `\ytshake`, `\ytchroma` with its custom colours, and the `\ytkt` karaoke types with the cursor forms), and the Tier 3 tags (`\ytruby`, `\ytvert`, `\ytpack`, `\ytdir4`, and `\ytdir6`). A tag outside those tiers is accepted and ignored. [The ASS support page](ass-support.md) maps each tag onto its IR field and its test.

The writer emits those same tiers, including `\bord`, `\shad`, `\s`, `\fscx`, `\fscy`, and the cursor forms of `\ytkt`. It writes the style table in a fixed field order and one Dialogue line per cue. It emits the vertical mode and the reading direction where they change, so a direction applies from the tag onward. A background box or an outline colour writes through the `\3c` and `\3a` tags. A hard shadow writes through `\4c` and `\4a`, and a box style writes BorderStyle 3. A colour tag carries the transparency of the channel, where 0 is opaque.

The writer reports these features that it cannot express:

- A shadow list with more than one non-glow shadow, because ASS carries one shadow channel
- A karaoke gap
- A chroma with more than one offset, when the offsets do not form the spread that the argument form rebuilds
- A keyframe with no animated value
- A cursor karaoke type with no text
- A ruby base with more than one reading

The writer keeps a chroma whose offsets form the symmetric spread that the argument form rebuilds, so a chroma that came from ASS round-trips without a loss. A blank `\ytvert` returns a run to horizontal text, so a run can leave vertical text without a loss.

Two limitations apply. The outline colour of a style that is not the base style becomes a glow shadow on the span. The IR carries an outline colour only on the style. Karaoke text keeps the timing of its syllable, so several spans can share one karaoke window.

## WebVTT (vtt)

WebVTT carries a signature line, one or more cue blocks, and comment blocks. The reader accepts the signature with or without a byte order mark, skips a `NOTE` block, and reads an optional cue id. It maps the `align` and `position` settings onto the cue anchor and position, the inline tags `<i>`, `<b>`, and `<u>` onto span overrides, and the character references such as `&amp;` onto their characters. A voice span has no IR field, so the reader keeps the spoken text and drops the span.

The writer emits the signature, the `align` and `position` settings where a cue carries a layout, and the inline tags. It reports karaoke timing, a cue fade, a cue move, a vertical alignment, an animation, a foreground, secondary, or background colour, a font, a font size, a strikeout run, a glyph scale, a shadow, an outline width, vertical text, a script offset, a right-to-left marking, packing, and ruby text. Ruby text falls back to bracketed readings.

## TTML (ttml)

The reader covers the YouTube TTML dialect. It reads the named styles of the `styling` section and the placements of the `layout` section, then maps a style onto the document styles and onto the spans of its paragraph. A style reference may extend another style. It maps a region onto the cue anchor and position. It reads the timing from the `begin` and `end` attributes, or from `begin` and `dur`, and the karaoke timing from the `begin` attribute of a run. It accepts the clock forms `HH:MM:SS`, `HH:MM:SS.mmm`, `HH:MM:SS:FF`, and `MM:SS`, plus the offset forms with an `h`, `m`, `s`, `ms`, or `f` unit.

The writer emits general TTML: one `style` element per document style, one `region` element per cue placement, and one `p` element per cue with its inline spans. The inline form carries the font, the size, the bold, the italic, the underline, the strikeout, the foreground colour, the background colour, and the outline width. It reports a cue fade, a cue move, an animation, a glyph scale, a shadow, vertical text, packing, a right-to-left marking, a script offset, and ruby text. Ruby text falls back to bracketed readings.

## Kdenlive subtitle JSON (kdenlive)

Kdenlive keeps a subtitle track as a JSON array. Each element carries a `layer`, the `startPos` in seconds, and the `dialogue` as an ASS event line. The event line holds the layer, the start, the end, the style, the name, the margins, the effect, and the text, with the two-character sequence `\N` for a line break.

The reader takes the start from `startPos` and the end from the event line. It turns `\N` into a line break and strips the override blocks, because the format carries no styling of its own. The writer emits one element per cue with layer zero and the text in the last field. It reports karaoke timing, positioning, an animation, inline styling, a glyph scale, vertical text, packing, a right-to-left marking, a script offset, and ruby text. Ruby text falls back to bracketed readings.

## Lossless JSON exchange (json1)

JSON1 is the internal exchange format of `swag`. It carries a version field and the whole IR document, so a reader and a writer of the format lose nothing. The writer indents with two spaces. The reader rejects a bad document, a missing version, and a version it does not know. Use JSON1 to pass a document between tools without a loss: convert into JSON1, keep the file, and convert out of it later.

## Karaoke

A span carries a start and an end offset from the cue start. A cue is a karaoke cue when any span carries a non-zero offset. The unsung colour comes from the style, or from the span when the source set it. `model.NormaliseKaraoke` bumps a zero-length segment by one millisecond and keeps the segments in order for a format that rejects them.

## Ruby text

A base span is followed in the span list by one or more annotation spans that carry the reading. A converter moves a base and its annotations together. A format without ruby support writes the reading in brackets after the base.

## Colour

An IR colour is RGBA, with the full 0 to 255 range on every channel. A format without an alpha channel writes a loss note when it drops that channel.
