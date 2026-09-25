# The terminal palette

A terminal paints text on a background that the user chose. To match that background, a tool can ask the terminal for its colours. The query uses an operating system command (OSC) sequence, an extension of the xterm terminal. No portable interface exists for the query, and many terminals ignore it. This page records [the palette prototype](../internal/tui/palette.go), what it found, and the steps for a full implementation. [The roadmap](../ROADMAP.md) carries theming as a stretch goal after 1.0.0.

The full xterm sequences live in [the xterm control sequence reference](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html).

## The query

A program writes an OSC sequence to the terminal and reads the answer from its input. Three queries carry the colours:

| Colour | Query | Answer |
|---|---|---|
| Indexed colour N | `ESC ] 4 ; N ; ? ST` | `ESC ] 4 ; N ; rgb:RRRR/GGGG/BBBB ST` |
| Foreground | `ESC ] 10 ; ? ST` | `ESC ] 10 ; rgb:RRRR/GGGG/BBBB ST` |
| Background | `ESC ] 11 ; ? ST` | `ESC ] 11 ; rgb:RRRR/GGGG/BBBB ST` |

`ESC` is the escape byte (`\x1b`) and `ST` is the string terminator (`ESC \`). A question mark stands where the terminal writes the value. The first sixteen indexed colours are the ANSI palette.

Some terminals end the answer with the bell byte (`\x07`) instead of the string terminator. A reader accepts both.

## The reply forms

Two forms appear in the field:

| Form | Example | Notes |
|---|---|---|
| xterm | `rgb:ffff/0000/0000` | Each channel carries one to four hexadecimal digits. |
| Hash | `#ff0000` | Six digits, or three when each digit repeats. |

The xterm form scales each channel onto a byte. Four digits scale by 65535, two digits by 255, and one digit by 15. The parser in [the palette prototype](../internal/tui/palette.go) handles all three lengths, so a terminal that writes a short channel still yields a colour.

## The fallback

A terminal that ignores the query sends nothing. [The prototype](../internal/tui/palette.go) then waits for a timeout and returns a plain palette: a white foreground on a black background. The `Known` field stays false, so a caller knows that the values are the fallback and not an answer. The probe returns the same plain palette when the write fails.

An answer that arrives before the timeout stays in the palette. A terminal that answers the foreground and then goes quiet therefore contributes the foreground, and the background keeps its fallback value.

## What the prototype found

[The palette prototype](../internal/tui/palette.go) sends the OSC 4, 10, and 11 queries and parses the answers. [The palette suite](../internal/tui/palette_test.go) proves the query text, the two terminators, the two reply forms, a partial answer, a stray escape, an unterminated sequence, a foreign sequence, every malformed shape, and the plain fallback.

The prototype settles the shape of the problem:

1. The queries work on a terminal that implements them. The parser must accept both terminators and both reply forms, because terminals differ.
2. A terminal that ignores the query costs one short wait and no error.
3. The timeout matters. A terminal keeps its input open after the answers and never reports the end of the stream.
4. The reading goroutine leaves a reader on the input when the terminal is silent. In cooked mode that goroutine takes the next line the user types. This is the reason the prototype does not run by default.

Point 4 names the one change that a full implementation needs. The goroutine solves the read, but it does not solve the ownership of the input.

## The plan for a full implementation

The steps below replace the goroutine with a wait on the file descriptor. [The termenv package](https://github.com/muesli/termenv) follows this sequence, and it is a useful reference.

1. Make sure that the input is a terminal. When the input is a pipe or a file, skip the probe.
2. Make sure that the process is in the foreground. A background process must not read from the terminal that the shell owns.
3. When `TERM` starts with `screen`, `tmux`, or `dumb`, skip the probe. A terminal multiplexer can connect to several terminals at once, so it cannot answer for one of them.
4. Save the terminal mode, then turn off the echo and the canonical mode for the query. The canonical mode holds input until the user presses Enter, so the answer arrives late.
5. Write the queries, then write a cursor position report (`CSI 6n`). Every terminal answers the cursor report, so the report acts as a sentinel. The answer to the report means that every OSC answer arrived. When the first response is the cursor report, the terminal ignores the OSC query.
6. Wait for data with `select(2)` and a timeout, then read. This is the key change. The wait bounds the probe without a goroutine, so the probe keeps ownership of the input.
7. Restore the terminal mode.
8. Cap the response at about 25 bytes. Every answer is short, and the cap stops a flood.
9. Parse the answers, and keep the plain palette for anything missing.

A terminal build of the tool carries the file descriptor path for Unix and a separate path for the Windows console. Keep the queries, the parser, and the fallback in a portable file, and keep the terminal mode and the wait behind a small interface. The portable part then runs on every platform, and a test injects a fake in place of the real terminal.

### The colour depth

The palette carries the colours, and the depth decides how many of them a run can paint. Two variables report the depth:

| Variable | Value | Meaning |
|---|---|---|
| `COLORTERM` | `truecolor` or `24bit` | 16 million colours |
| `TERM` | contains `256color` | 256 colours |
| `TERM` | contains `color` or `ansi` | 16 colours |

A run without either variable uses plain text. The depth and the palette answer different questions, so a full implementation reads both. [The termenv colour profile](https://github.com/muesli/termenv) maps the two variables onto the four levels.

The `COLORFGBG` variable carries the foreground and the background as ANSI indexes, in the form `fg;bg`. The rxvt terminal family sets it. It is a cheap fallback when the query fails, and it costs one environment lookup.

### The order of the theme sources

A full implementation resolves the theme in this order, and the first source that answers wins:

1. The theme in the configuration file. [The configuration page](configuration.md) covers the file, and [the roadmap](../ROADMAP.md) scopes the schema to v0.9.0.
2. The palette from the terminal query.
3. The `COLORFGBG` variable.
4. The plain theme.

A run without a terminal, such as a pipe, stays on the plain theme. The colour predicate in `cmd/swag` reads the pterm colour state today. A full implementation adds the terminal check, the palette, and the depth to that one predicate, so a single place decides the theme.

## Tests

[The palette suite](../internal/tui/palette_test.go) covers the portable part with a scripted reader. It needs no terminal.

The Unix path needs a pseudo terminal, because `select(2)` waits on a real file descriptor. A test opens a pseudo terminal pair, writes an answer on the master side, and runs the probe on the slave side. [The testing notes](testing.md) record the suites of the project.
