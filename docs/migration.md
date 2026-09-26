# Migrating from YTSubConverter

This page maps a YTSubConverter workflow onto `swag`. It covers the command line, the reverse conversion, the shadow and karaoke settings, and the places where the two tools differ. [The third-party notices](third-party-notices.md) record the attribution to [the YTSubConverter project](https://github.com/arcusmaximus/YTSubConverter).

## What stays the same

Both tools serve one main job: turn styled Advanced SubStation Alpha (ASS) subtitles into a file that the YouTube player renders with styling. The feature set is the same, so a document that YTSubConverter handles keeps its features here. Both tools:

- Apply the YouTube font allow-list and snap an unknown font to Roboto.
- Use the baseline rule of the `Default` style for the font size of every other style.
- Carry karaoke timing, vertical text, ruby text, shakes, chromas, and the `\ytkt` karaoke types.
- Reproduce the documented quirks of the upload path, such as the opacity ceiling and the white shift.

## The command line

YTSubConverter takes one input and infers the output. `swag` names the target with the output extension or with the `-f` flag.

| YTSubConverter | `swag` |
|---|---|
| `YTSubConverter in.ass` | `swag -i in.ass -f ytt` |
| `YTSubConverter in.ass out.ytt` | `swag -i in.ass -o out.ytt` |
| `YTSubConverter in.ytt out.ass` | `swag -i in.ytt -o out.ass` |
| `YTSubConverter in.srv3 out.ass` | `swag -i in.srv3 -o out.ass` |
| `YTSubConverter in.sbv out.srt` | `swag -i in.sbv -o out.srt` |
| `YTSubConverter in.ttml out.ytt` | `swag -i in.ttml -o out.ytt` |

`swag` writes to standard output when `-o` is absent, so a pipe replaces a temporary file:

```sh
swag -i in.ass -f ytt | head
```

A directory input converts every subtitle under it, and `-f` names one target or a comma-separated list:

```sh
swag -i captions/ -f ytt
swag -i captions/ -f ytt,srt -o converted/
```

## The reverse conversion

The default reverse conversion of YTSubConverter produces an ASS file that converts back to the same look with different text. `swag` always produces that form, because its reader keeps the document in the intermediate representation and its writer emits the same tags again. The round trip is the tested path.

YTSubConverter also offers a visual form through the `--visual` flag for `.ytt` or `.srv3` to `.ass`. The visual form approximates the player in a local media player, and it cannot convert back. `swag` has no `--visual` flag. A viewer that renders SRV3 directly covers that use case, or the reader keeps the document exact for a later edit.

## Shadow types

YTSubConverter asks for the shadow types in the conversion window, and it combines the four kinds through checkboxes. `swag` carries a shadow list in the document instead. A YTT or SRV3 write turns each non-glow shadow kind into one line, because a pen carries one edge. [The format notes](formats.md) list the shadow rules.

## Karaoke highlighting

YTSubConverter configures current-word highlighting in its window. `swag` reads the same effect from the source. The `\k` tags carry the timing, and the `\ytkt` cursor forms name the highlight text or the animated frames. [The ASS support page](ass-support.md) maps each tag onto its field and its test.

## Differences to know

- `swag` reports every feature that a target drops. YTSubConverter applies the upload path in silence. Pass `--strict` to fail a conversion that drops a feature, or read the report with `-v`.
- `swag` reads and writes more formats. WebVTT, TTML, the Kdenlive subtitle JSON, and the lossless JSON1 exchange join the ASS, SRT, SBV, YTT, and SRV3 set.
- A conversion through SubRip or SBV keeps the whole document for `swag` through [the integrity block](integrity.md). A player outside `swag` sees the plain cues.
- `swag` runs on Linux, macOS, Windows, and the web. [The install page](install.md) lists the release targets, and [the demo page](demo/index.html) runs a conversion in the browser.
- `swag` has no autoconvert watch mode. A short loop or the batch conversion covers a repeated run.

## A full example

The style-assignment workflow of YTSubConverter ends with an upload file. The same ending works here:

1. Create the styles in Aegisub, and save the subtitles as an ASS file.

2. Convert the file for upload.

   ```sh
   swag -i episode.ass -o episode.ytt --font "Trebuchet MS"
   ```

3. Import `episode.ytt` on the YouTube Studio subtitle page, and publish it.

Use `--strict` when a dropped feature must fail the run:

```sh
swag -i episode.ass -o episode.ytt --strict
```
