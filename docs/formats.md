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

A writer returns a loss report, which is one entry per degraded feature. The command line interface prints the report when `-v` is set.

## SubRip (srt)

SubRip carries plain text. It reads a cue counter, a timing line of the form `hh:mm:ss,mmm --> hh:mm:ss,mmm`, and one or more text lines. Two inline tags are understood: `<i>` and `<b>`.

The writer drops everything else. It reports karaoke timing, positioning, animations, foreground colour, transparency, shadows, vertical text, script offsets, right-to-left marking, and ruby text. Ruby text falls back to bracketed readings.

## YouTube SBV (sbv)

SBV carries plain text only. It reads a timing line of the form `h:mm:ss.mmm,h:mm:ss.mmm` and one or more text lines.

The writer drops styling, karaoke, and positioning. Ruby text falls back to bracketed readings.

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

The writer reports the effects it cannot express: cue fades, cue moves, and the shake, chroma, keyframe, and karaoke-type animations.

## Advanced SubStation Alpha (ass)

ASS carries a Script Info section, a V4+ Styles section, and an Events section. The reader maps each style onto an IR style. It flattens the style of a cue onto the spans of that cue, because the IR carries one document style and per-span overrides.

The reader supports the Tier 1 tags (`\b`, `\i`, `\u`, `\s`, `\fn`, `\fs`, `\fscx`, `\fscy`, the colour tags, the alpha tags, `\bord`, `\shad`, `\a`, `\k` and its variants, `\r`, `\an`, and `\pos`), the Tier 2 effects (`\move`, `\fad`, `\fade`, `\t`, `\ytshake`, `\ytchroma` with its custom colours, and the `\ytkt` karaoke types with the cursor forms), and the Tier 3 tags (`\ytruby`, `\ytvert`, `\ytpack`, `\ytdir4`, and `\ytdir6`). A tag outside those tiers is accepted and ignored. [The ASS support page](ass-support.md) maps each tag onto its IR field and its test.

The writer emits those same tiers, including `\bord`, `\shad`, `\s`, `\fscx`, `\fscy`, and the cursor forms of `\ytkt`. It writes the style table in a fixed field order and one Dialogue line per cue. It emits the vertical mode and the reading direction where they change, so a direction applies from the tag onward. A background box or an outline colour writes through the `\3c` and `\3a` tags. A hard shadow writes through `\4c` and `\4a`, and a box style writes BorderStyle 3. A colour tag carries the transparency of the channel, where 0 is opaque.

The writer reports these features that it cannot express:

- A shadow that is not hard
- A chroma effect with more than one copy
- A run that leaves vertical text
- A karaoke gap
- A ruby base with more than one reading

Two limitations apply. The outline colour of a style that is not the base style becomes a glow shadow on the span. The IR carries an outline colour only on the style. Karaoke text keeps the timing of its syllable, so several spans can share one karaoke window.

## Karaoke

A span carries a start and an end offset from the cue start. A cue is a karaoke cue when any span carries a non-zero offset. The unsung colour comes from the style, or from the span when the source set it. `model.NormaliseKaraoke` bumps a zero-length segment by one millisecond and keeps the segments in order for a format that rejects them.

## Ruby text

A base span is followed in the span list by one or more annotation spans that carry the reading. A converter moves a base and its annotations together. A format without ruby support writes the reading in brackets after the base.

## Colour

An IR colour is RGBA, with the full 0 to 255 range on every channel. A format without an alpha channel writes a loss note when it drops that channel.
