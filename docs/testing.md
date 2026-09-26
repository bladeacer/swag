# Testing notes

The project holds statement coverage at 100 percent, runs a nightly search of every reader, and benchmarks a large document. This page records how each part runs.

## Coverage

Run `go test -cover ./...` for a package summary, or `make cover` for the per-function breakdown. The floor is 100 percent, and two places enforce it: [the Makefile](../Makefile) target `cover-verify` and [the continuous integration workflow](../.github/workflows/ci.yml). A statement that cannot be reached must change or leave, because a dead branch drops the floor.

Each package carries its own tests next to the code. Reader and writer pairs have a round-trip test over a fixture under `internal/formats/<format>/testdata/`. Every shipped writer also appears in [the cross-check suite](../internal/formats/ass/crosscheck_test.go), which runs an ASS fixture through the writer and asserts its loss report.

Several suites sit above the package tests:

- [The cross-check chains](../internal/formats/ass/crosscheck_test.go) run one fixture per reader through a sequence of two or more formats and back. The readers and the writers therefore compose.
- [The integrity test](../pkg/sub/integrity_test.go) proves that a conversion through SubRip or SBV returns the exact document. It covers the ASS fixtures and the karaoke fixture.
- [The exchange suite](../pkg/sub/e2e_test.go) reads every format, writes the document as JSON1, and reads it back, so the lossless hop holds for every reader.
- [The voice span suite](../pkg/sub/voice_test.go) pins what each format does with a speaker name. It covers the formats that keep the name and the ones that report the loss.
- [The command line suite](../cmd/swag/main_test.go) walks every registered target through the real command and back, including the bare run and the default command form.
- [The renderer suite](../internal/tui/tui_test.go) proves the layout diff. An unchanged frame writes no bytes, a changed frame repaints only the changed rows, and a shorter frame clears the rows it leaves out.
- [The palette suite](../internal/tui/palette_test.go) covers the terminal palette probe. It covers the OSC 4, 10, and 11 queries, the reply forms, a partial answer, and the plain fallback when a terminal stays silent.
- [The interactive suite](../cmd/swag/interactive_test.go) drives the prompts and the frames through a scripted reader, so a whole conversion runs with no terminal. It covers both the line prompt and the key bindings.
- [The keybind suite](../internal/tui/keys_test.go) covers the token grammar, the commutative modifiers, the upper-case shorthand, the incremental matcher, and the prefix conflicts.
- [The key reader suite](../internal/tui/reader_test.go) covers the line prompt, the timeout sequence, the text fallback, and the echo.
- [The key source suite](../cmd/swag/keysource_test.go) covers the stream source, the terminal source, and the raw setup, with the terminal calls injected.
- [The batch and preview suites](../cmd/swag/batch_test.go) cover the directory walk, the target list, the output directory, and the preview rows.
- [The configuration suite](../internal/config/config_test.go) covers the location rule of every platform, the two environment overrides, the working directory file name, and the failure of a platform with no home directory. [The settings suite](../internal/config/settings_test.go) covers the decode, a broken document, a value of the wrong type, a duplicate key, an unknown setting, the merge of two files, the decode cache, the merge cache, and the file at the repository root. [The command line configuration suite](../cmd/swag/config_test.go) walks the whole precedence chain in one table: the command line flag, then the working directory file, then the global file, then the built-in default.

Tests reuse the shipped [default configuration file](../swag.toml) for the default case. [The settings suite](../internal/config/settings_test.go) decodes the embedded copy and the root copy, so neither drifts. A configuration variant that a test needs goes under `internal/config/testdata/`, so the repository root stays free of stray files. [The command line suite](../cmd/swag/main_test.go) points `SWAG_CONFIG_DIR` at a temporary directory, so the suite never reads the global configuration file of the machine that runs it.

## Fuzzing

Every reader has a fuzz target in [the fuzz file](../pkg/sub/fuzz_test.go). A target parses an arbitrary string, then renders the result as JSON1, so a reader must not panic and a document it accepts must render. The targets run their seed corpus during a normal test run.

A longer search runs in [the nightly fuzz workflow](../.github/workflows/fuzz.yml). The workflow runs each target for three minutes on a schedule, which spends about half an hour in total. It uploads a crash corpus on a failure. Fuzzing found a crash in the ASS reader: a `\t` tag with two arguments read past the end of its argument list. [The regression test](../internal/formats/ass/gaps_test.go) covers that input. A later short run of all nine targets, about twenty seconds each, found no further crash.

The configuration decoder and the keybind token parser carry targets too, so the workflow runs twelve in total. [The configuration fuzz file](../internal/config/fuzz_test.go) checks that a decoded document keeps no blank preferred name. [The keybind fuzz file](../internal/tui/fuzz_test.go) checks that a parsed token list names at least one key and that the keymap builder answers a binding lookup for an accepted override table.

Run one target by hand with:

```sh
go test ./pkg/sub/ -run=^$ -fuzz=^FuzzParseASS$ -fuzztime=30s
```

## Benchmarks

[The benchmark file](../pkg/sub/bench_test.go) builds a document with ten thousand cues and measures the parse, the render, and the conversion in every registered format. The conversion run covers the conversion into SubRip itself, so every cell of the conversion table carries a number.

```sh
make bench
```

[The configuration benchmark](../internal/config/bench_test.go) and [the keybind benchmark](../internal/tui/bench_test.go) cover the interactive startup and the key matching. Run every benchmark in the module with:

```sh
make bench-all
```

For a regression check, save a baseline on the same machine and compare a fresh run with `benchstat`:

```sh
make bench-save     # writes bench.txt
make bench-compare  # writes new-bench.txt and compares the two
```

`benchstat` reports the change of each benchmark and marks the significant rows. Compare runs from the same machine and the same Go version, because the numbers depend on the hardware.
