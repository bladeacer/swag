# Testing notes

The project holds statement coverage at 100 percent, runs a nightly search of every reader, and benchmarks a large document. This page records how each part runs.

## Coverage

Run `go test -cover ./...` for a package summary, or `make cover` for the per-function breakdown. The floor is 100 percent, and two places enforce it: [the Makefile](../Makefile) target `cover-verify` and [the continuous integration workflow](../.github/workflows/ci.yml). A statement that cannot be reached must change or leave, because a dead branch would drop the floor.

Each package carries its own tests next to the code. Reader and writer pairs have a round-trip test over a fixture under `internal/formats/<format>/testdata/`. Every shipped writer also appears in [the cross-check suite](../internal/formats/ass/crosscheck_test.go), which runs an ASS fixture through the writer and asserts its loss report.

Four suites sit above the package tests:

- [The cross-check chains](../internal/formats/ass/crosscheck_test.go) run one fixture per reader through a sequence of two or more formats and back, so the readers and the writers compose.
- [The integrity test](../pkg/sub/integrity_test.go) proves that a conversion through SubRip or SBV returns the exact document. It covers the ASS fixtures and the karaoke fixture.
- [The exchange suite](../pkg/sub/e2e_test.go) reads every format, writes the document as JSON1, and reads it back, so the lossless hop holds for every reader.
- [The voice span suite](../pkg/sub/voice_test.go) pins what each format does with a speaker name, from the formats that keep it to the ones that report the loss.
- [The command line suite](../cmd/swag/main_test.go) walks every registered target through the real command and back, including the bare run and the default command form.

## Fuzzing

Every reader has a fuzz target in [the fuzz file](../pkg/sub/fuzz_test.go). A target parses an arbitrary string, then renders the result as JSON1, so a reader must not panic and a document it accepts must render. The targets run their seed corpus during a normal test run.

A longer search runs in [the nightly fuzz workflow](../.github/workflows/fuzz.yml). The workflow runs each target for three minutes on a schedule, which spends about half an hour in total, and it uploads a crash corpus on a failure. Fuzzing found a crash in the ASS reader: a `\t` tag with two arguments read past the end of its argument list. [The regression test](../internal/formats/ass/gaps_test.go) covers that input.

Run one target by hand with:

```sh
go test ./pkg/sub/ -run=^$ -fuzz=^FuzzParseASS$ -fuzztime=30s
```

## Benchmarks

[The benchmark file](../pkg/sub/bench_test.go) builds a document with ten thousand cues and measures the parse, the render, and the conversion in every registered format.

```sh
make bench
```

For a regression check, save a baseline on the same machine and compare a fresh run with `benchstat`:

```sh
make bench-save     # writes bench.txt
make bench-compare  # writes new-bench.txt and compares the two
```

`benchstat` reports the change of each benchmark and marks the significant rows. Compare runs from the same machine and the same Go version, because the numbers depend on the hardware.
