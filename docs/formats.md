# Formats

`swag` reads a subtitle file into one intermediate representation (IR) and writes it out in another format. A reader turns the concepts of its format into IR types. A writer turns IR types into the concepts of its format and reports every feature it cannot express. [The architecture page](architecture.md) describes the IR, and [the loss report review](loss-report.md) lists the degradation of every format at a glance.

## The formats

Legend: R reads, W writes.

| Format | Registry name | Extensions | Support | Specification |
|---|---|---|---|---|
| SubRip | `srt` | `srt` | R, W | [No formal specification](#subrip-srt) |
| YouTube SBV, also called SubViewer | `sbv` | `sbv` | R, W | [YouTube caption formats](https://support.google.com/youtube/answer/2734698) |
| YouTube Timed Text | `ytt` | `ytt` | R, W | [No published specification](#youtube-timed-text-ytt-and-srv3-srv3) |
| YouTube SRV3 | `srv3` | `srv3` | R, W | [No published specification](#youtube-timed-text-ytt-and-srv3-srv3) |
| Advanced SubStation Alpha | `ass` | `ass`, `ssa` | R, W | [ASS override tags](https://aegisub.org/docs/latest/ass_tags/) |
| WebVTT | `vtt` | `vtt` | R, W | [The WebVTT specification](https://www.w3.org/TR/webvtt1/) |
| TTML, also called DFXP | `ttml` | `ttml`, `dfxp` | R, W | [TTML2](https://www.w3.org/TR/ttml2/) |
| Kdenlive subtitle JSON | `kdenlive` | `kdenlive` | R, W | [The Kdenlive subtitle tool](https://docs.kdenlive.org/en/effects_and_filters/subtitles.html) |
| Lossless JSON exchange | `json1` | `json1` | R, W | [The JSON1 page](json1.md) |

Every format reads and writes. A format without a reader or without a writer is not registered.

## Reading and writing

| Step | What happens |
|---|---|
| Read | The reader detects its own signature where the format has one, parses the cues, and builds one document. It emits at least one style. |
| Write | The writer renders the document, then returns a loss report: one entry per feature that the target cannot carry. |
| Convert | `swag` chains the two, and the command prints the report when `-v` is set. |

A source file that carries no style table still gains one style, so a document always has a style to inherit from. Coordinates resolve against `VideoDimensions`, which defaults to 1280 by 720 for a format that carries no resolution.

## SubRip (srt)

**Support.** The reader takes a cue counter, a timing line of the form `hh:mm:ss,mmm --> hh:mm:ss,mmm`, and one or more text lines. It understands two inline tags, `<i>` and `<b>`, and it keeps a line break inside the span that ends the line. The hour field can carry more than two digits. The writer emits the counter, the timing, the text with those two tags, and a lossy write appends [the integrity block](integrity.md).

**Specification.** SubRip has no formal specification. The format comes from [the SubRip program](https://en.wikipedia.org/wiki/SubRip). The description on [the YouTube caption support page](https://support.google.com/youtube/answer/2734698) covers the shape that upload accepts: plain UTF-8 with no markup. This project follows that common shape and adds the two tags that most tools read.

**Caveats and limitations.** SubRip carries no style table, no positioning, no colour, and no timing inside a cue. The writer reports every one of those as a loss. A ruby reading falls back to bracketed text, which is the shape that a player without ruby support shows. Transparency cannot be carried at all. The integrity block holds the full document, so a round trip through SubRip keeps the content for `swag` while a plain player sees plain subtitles.

## YouTube SBV (sbv)

**Support.** The reader takes a timing line of the form `h:mm:ss.mmm,h:mm:ss.mmm` and one or more text lines. The writer emits the same shape, with an hour field of one digit or more. The format carries no inline tag, so a lossy write appends [the integrity block](integrity.md).

**Specification.** The format is the one that YouTube lists as SubViewer on [the YouTube caption support page](https://support.google.com/youtube/answer/2734698). YouTube also notes that upload accepts the `.sub` extension there. This project uses the `.sbv` name.

**Caveats and limitations.** The reader separates a timing line from a text line by shape. A timing line needs a comma, a dot in each half, and at least one colon in each half. A text line that carries all of those is rare. Bold, italic, and strikeout cannot be carried, because SBV has no tag for them, and the writer reports them together as inline styling. Everything else that SubRip cannot carry is missing here too.

## YouTube Timed Text (ytt) and SRV3 (srv3)

**Support.** Both are XML. A `<pen>` element carries the style of a text run. A `<ws>` element carries the window of a line, and a `<wp>` element carries a named window position. The body holds `<p>` lines, each with `<s>` runs whose `t` attribute gives the appearance time. SRV3 is an older dialect of the same document, so both formats share one reader, one writer, and one pen model.

The reader maps pens onto span overrides, window positions onto the cue layout, and window styles onto the vertical mode or the reading direction. It pairs ruby base spans with their readings and drops the parenthesis fallback. It turns the appearance time of each run into karaoke span offsets.

The writer deduplicates pens, lists them in increasing id order, and applies the quirks of the YouTube upload path. A cue with more than one shadow kind becomes one line per kind, because a pen carries one edge.

**Specification.** YouTube publishes no schema for the XML of the caption tracks. The behaviour here comes from the documented quirks of the upload path. It also comes from the feature set that [YTSubConverter](https://github.com/arcusmaximus/YTSubConverter) targets, the project that inspired this one. [The third-party notices](third-party-notices.md) record the attribution. [The YouTube caption support page](https://support.google.com/youtube/answer/2734698) covers the upload formats around it.

**Platform quirks.** The writer changes these values, because the player or the upload path rejects or rewrites them:

- A font outside the YouTube list snaps to Roboto, and the writer reports the loss.
- The font scale remaps through the virtual percentage of the format, where the real factor is `1 + (sz/100 - 1)/4`. A factor below 0.75 clamps to the minimum, and the writer reports it.
- An alpha value of 255 becomes 254, because the upload drops an attribute with the maximum value.
- A pure white foreground becomes `#FEFEFE`, because the Android client ignores `#FFFFFF`.
- A pure black foreground becomes `#010101`, because the upload drops a foreground that matches the default background.
- A line with more than one run gains a zero-width space after the first run. Without that space, the upload removes the pen of the first run.
- A karaoke segment with no length gains one millisecond, and the segments stay in order.
- An italic or shadowed line start gains one space, so the player does not clip the overhang.

**Caveats and limitations.** The writer cannot express some effects: a cue fade, a cue move, a shake, a chroma, a keyframe, and a karaoke-type animation. It also reports a strikeout run, a glyph scale, and a voice name. The quirks above mean that a write is not byte-identical to the source file, even for the features that survive. The format round-trips as a document, and not as a file. The reader accepts a document that carries no `<head>`, and it treats a missing pen as the default style.

## Advanced SubStation Alpha (ass)

**Support.** ASS carries a Script Info section, a V4+ Styles section, and an Events section. The reader maps each style onto an IR style and flattens the style of a cue onto the spans of that cue. The IR carries one document style and per-span overrides.

The reader covers the Tier 1 tags, the Tier 2 effects, and the Tier 3 tags. Tier 1 holds the styling and timing tags: `\b`, `\i`, `\u`, `\s`, `\fn`, `\fs`, `\fscx`, `\fscy`, the colour tags, and the alpha tags. It also holds `\bord`, `\shad`, `\a`, `\k` and its variants, `\r`, `\an`, and `\pos`. Tier 2 holds the effects: `\move`, `\fad`, `\fade`, `\t`, `\ytshake`, `\ytchroma` with its custom colours, and the `\ytkt` karaoke types with the cursor forms. Tier 3 holds `\ytruby`, `\ytvert`, `\ytpack`, `\ytdir4`, and `\ytdir6`. A tag outside those tiers is accepted and ignored, which is what the format asks of a renderer.

The writer emits those same tiers and writes the style table in a fixed field order, one Dialogue line per cue. It emits the vertical mode and the reading direction where they change, so a direction applies from the tag onward. A background box or an outline colour writes through `\3c` and `\3a`. A hard shadow writes through `\4c` and `\4a`, and a box style writes BorderStyle 3. A colour tag carries the transparency of the channel, where 0 is opaque.

**Specification.** [The ASS override tag reference](https://aegisub.org/docs/latest/ass_tags/) documents every tag and its arguments. The renderer that most tools use is [libass](https://github.com/libass/libass), which is the behaviour this project matches where the documentation is quiet.

**Caveats and limitations.** The writer reports these features that it cannot express:

- A shadow list with more than one shadow that is not a glow, because ASS carries one shadow channel. A glow writes through the outline channel, so a glow beside one other shadow survives.
- A karaoke gap between two segments.
- A chroma with more than one offset, when the offsets do not form the spread that the argument form rebuilds.
- A keyframe with no animated value.
- A cursor karaoke type with no text.
- A ruby base with more than one reading.
- A voice name, which has no ASS form.

Two further limits apply. The outline colour of a style that is not the base style becomes a glow shadow on the span. The IR carries an outline colour only on the style. Karaoke text keeps the timing of its syllable, so several spans can share one karaoke window. A blank `\ytvert` returns a run to horizontal text, so a run can leave vertical text without a loss. [The ASS support page](ass-support.md) maps each tag onto its IR field and its test.

## WebVTT (vtt)

**Support.** WebVTT carries a signature line, one or more cue blocks, and comment blocks. The reader accepts the signature with or without a byte order mark. It skips a `NOTE` block, and it reads an optional cue id, which it does not keep. It maps the `align` and `position` settings onto the cue anchor and position. It maps the inline tags `<i>`, `<b>`, and `<u>` onto span overrides, and the character references such as `&amp;` onto their characters. It maps the voice annotation `<v Speaker>` onto the voice name of the span and the closing `</v>` back to no voice.

The writer emits the signature, the voice annotation, the `align` and `position` settings where a cue carries a layout, and the inline tags. A ruby reading falls back to bracketed text, because the format has no ruby form.

**Specification.** [The WebVTT specification](https://www.w3.org/TR/webvtt1/) carries the normative grammar of the file, the cue settings, and the cue text tags, including the voice span.

**Caveats and limitations.** The writer reports these features:

- Karaoke timing, a cue fade, a cue move, an animation, a vertical alignment, and a foreground, secondary, or background colour
- A font, a font size, a strikeout run, a glyph scale, a shadow, and an outline width
- Vertical text, a script offset, a right-to-left marking, packing, and ruby text

The reader is narrower than the specification in these places:

- Only the `align` and `position` settings are read. The `line`, `size`, `vertical`, `snap-to-lines`, and `region` settings are ignored, and their text stays.
- The cue id is dropped, because the IR carries no id.
- Class spans (`<c.class>`), language spans, and timestamp tags are dropped with their text kept.
- A `position` setting becomes a percentage of the video width. A position written from another format returns as the same percentage, and not as the same pixel.

A lossy write appends [the integrity block](integrity.md) as a `NOTE` block, which WebVTT treats as a comment. The format therefore keeps a document whole inside `swag` while a plain player sees plain cues.

## TTML (ttml)

**Support.** The reader covers the YouTube TTML dialect. It reads the named styles of the `styling` section and the placements of the `layout` section. It then maps a style onto the document styles and onto the spans of its paragraph. A style reference can extend another style, and the reader merges the chain. It maps a region onto the cue anchor and position. It reads the timing from the `begin` and `end` attributes, or from `begin` and `dur`. It reads the karaoke timing from the `begin` attribute of a run. It accepts the clock forms `HH:MM:SS`, `HH:MM:SS.mmm`, `HH:MM:SS:FF`, and `MM:SS`, plus the offset forms with an `h`, `m`, `s`, `ms`, or `f` unit.

The writer emits general TTML. It writes one `style` element per document style, one `region` element per cue placement, and one `p` element per cue with its inline spans. The inline form carries the font, the size, the bold, the italic, the underline, and the strikeout. It also carries the foreground colour, the background colour, and the outline width.

**Specification.** [TTML2](https://www.w3.org/TR/ttml2/) is the current Recommendation, and [TTML1](https://www.w3.org/TR/ttml1/) is the basis of the dialect that YouTube accepts on [the caption upload path](https://support.google.com/youtube/answer/2734698). This project reads the YouTube dialect and writes the general form, so the output suits any TTML reader.

**Caveats and limitations.** The writer reports a cue fade, a cue move, an animation, a glyph scale, and a shadow. It also reports vertical text, packing, a right-to-left marking, a script offset, ruby text, and a voice name. A ruby reading falls back to bracketed text. The writer emits clock times and no `ttp:` frame rate attributes, so a frame-based source writes as clock values. The reader takes a frame value as one thirtieth of a second, because it does not read the frame rate of the document. The reader is deliberately lenient, so it accepts a document that a strict validator refuses.

A lossy write appends [the integrity block](integrity.md) as an XML comment inside the body. Every XML reader ignores a comment, so the document stays valid, and the block sits inside the root element.

## Kdenlive subtitle JSON (kdenlive)

**Support.** Kdenlive keeps a subtitle track as a JSON array. Each element carries a `layer`, the `startPos` in seconds, and the `dialogue` as an ASS event line. The event line holds the layer, the start, the end, the style, the name, the margins, the effect, and the text. The two-character sequence `\N` marks a line break.

The reader takes the start from `startPos` and the end from the event line. It turns `\N` into a line break and strips the override blocks, because the format carries no style table of its own. The writer emits one element per cue with layer zero and the text in the last field.

**Specification.** [The Kdenlive subtitle tool](https://docs.kdenlive.org/en/effects_and_filters/subtitles.html) documents the feature from the user side, and [the subtitle model source](https://invent.kde.org/multimedia/kdenlive/-/blob/master/src/bin/model/subtitlemodel.cpp) carries the file shape. The format has no published schema, so this reader follows the fields that Kdenlive writes.

**Caveats and limitations.** The writer reports karaoke timing, positioning, an animation, inline styling, and a glyph scale. It also reports vertical text, packing, a right-to-left marking, a script offset, ruby text, and a voice name. Kdenlive writes a start position in seconds as a floating point value, so a time below one microsecond cannot be carried exactly. The reader needs the dialogue field to hold a well formed event line. An element without one is an error rather than a skipped entry.

The format carries no integrity block. Kdenlive keeps a subtitle track as a strict JSON array, so the file has no place for a comment or a marker. The writer reports the loss, and a conversion through JSON1 keeps the content.

## Lossless JSON exchange (json1)

**Support.** JSON1 carries the version of the file and the whole IR document, so a round trip loses nothing. [The JSON1 page](json1.md) covers the shape, the field meanings, and the version policy.

**Specification.** The format belongs to this project, so the page above is its specification.

**Caveats and limitations.** The format is meant for this tool and for the integrity block of a plain subtitle file. No other tool reads it. The writer always writes the current version, and the reader lifts an older file to it. A version with no migration step is an error.

## Metadata

`Document.Metadata` holds the file-level key and value pairs of a source, for example the `Title` of an ASS Script Info section. No writer emits the map, because no target format in the list has a general place for it. A format that carries its own metadata keeps it in its own reader. The map survives a JSON1 round trip and the integrity block, so a conversion through a plain format keeps the keys.

## Karaoke

A span carries a start and an end offset from the cue start. A cue is a karaoke cue when any span carries a non-zero offset. The unsung colour comes from the style, or from the span when the source set it. `model.NormaliseKaraoke` bumps a zero-length segment by one millisecond and keeps the segments in order for a format that rejects them.

## Ruby text

A base span is followed in the span list by one or more annotation spans that carry the reading. A converter moves a base and its annotations together. A format without a ruby form writes the reading in brackets after the base. That is the fallback that the WebVTT and YouTube renderers use for mobile.

## Voice spans

A voice span names the speaker of the text. WebVTT carries it as `<v Speaker>text</v>`, and the IR holds it in `TextSpan.Voice`. WebVTT and JSON1 keep the name, and the integrity block of SubRip and SBV keeps it. Every other format reports a voice name as a loss and keeps the text.

## Colour

An IR colour is RGBA, with the full 0 to 255 range on every channel. A format without an alpha channel writes a loss note when it drops that channel. ASS stores a colour as `&HAABBGGRR`, where the alpha is inverted, so the reader and the writer map the channel in both directions.
