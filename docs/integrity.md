# File integrity

SubRip and SBV hold plain text and timing. Everything else that the intermediate representation (IR) can carry, such as ruby text, karaoke timing, and colour, has no form in those formats. A conversion through one of them would therefore lose that content.

`swag` keeps the content instead. A plain writer that would drop at least one feature appends an integrity block to the end of the file. The block holds the whole document as JSON1, so a `swag` reader restores every feature. A plain subtitle player stops at the last cue and ignores the block.

The block is an extension of this project. A file outside `swag` stays a plain SubRip or SBV file, and a hand-written plain file reads exactly as before. [The lossless exchange page](json1.md) covers the format inside the block.

## The block

The block is a `NOTE` line, a base64 payload, and a blank line, at the end of the file:

```
1
00:00:01,000 --> 00:00:04,000
Hello world.

NOTE swag-ir 1
eyJ2ZXJzaW9uIjogIjIiLAogICJkb2N1bWVudCI6IHsKICA...
```

| Part | Meaning |
|---|---|
| `NOTE swag-ir` | The marker. A reader looks for it at the start of a line. |
| `1` | The version of the block shape. |
| The payload | The JSON1 document, base64 encoded, on one line. |

SubRip has no comment form, so the marker line reads as text to a strict parser. The block still survives, because a subtitle parser needs a cue counter and a timing line, and the block carries neither. The block also sits after the last cue and after a blank line, where most parsers have already stopped.

## Rules

- A document that fits in the plain format writes no block. A plain file stays plain.
- A write with at least one loss always writes the block, so the loss report and the block tell the same story.
- A reader restores the embedded document in place of the plain cues. It does not merge the two.
- A damaged block is an error, not a fallback. A file that looks truncated does not pass as a plain file.
- An unsupported block version is an error, so a newer file never reads as an older one by accident.

## What this buys

A three-way conversion keeps the exact content:

```sh
swag -i in.ass -o mid.srt
swag -i mid.srt -o out.ass
```

`out.ass` equals the ASS output of the original document, with the same style names, ruby readings, karaoke windows, and effects. [The integrity test](../pkg/sub/integrity_test.go) proves this for the ASS fixtures, and the voice span test proves the same for a speaker name.

The same conversion without the block loses the features that SubRip cannot carry, and the loss report lists them.

## Limits

- The block grows the file. A large document adds its JSON1 size, base64 encoded, which is about a third more. The block is a single line, so an editor that wraps long lines shows it as many lines while it stays one line in the file.
- The block carries only the document. A comment, the byte order, and the line endings of the source file are not kept.
- Only SubRip and SBV use the block, because they are the formats with no room of their own for the extra content. [The format notes](formats.md) list where every other format stands.
