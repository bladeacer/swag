# Performance audit

This page records the performance audit of the v1.0.0 release. It names the tools, the test machine, the numbers, the bottlenecks, and the work that stays open. The audit covers the parse, the render, and the conversion of every registered format.

## Method

The benchmarks live in `pkg/sub/bench_test.go`. Each one builds a document of ten thousand cues, then parses it, renders it, or converts it. The benchmark results are the first evidence.

- `make bench` runs the full set with memory numbers.
- `make bench-all` runs the benchmarks of every package that carries one, including the configuration and the keybind engine.
- `make bench-save` writes a baseline to `bench.txt`, and `make bench-compare` compares a new run with it through `benchstat`.
- `make profile-cpu` writes `cpu.prof` from the benchmark run. Read it with `go tool pprof -top cpu.prof`.
- `make profile-trace` writes `trace.out`. Read it with `go tool trace trace.out`.
- `strace -c` and `perf stat` run over a release build of the command, so the syscall and kernel costs of a conversion are visible.

A benchmark number is noisy, because a single run can hit a scheduling pause. The tables below come from one full run on one machine. Treat a difference under ten percent as noise. The WebVTT fix below is far above that line.

## Test machine

| Item | Value |
|---|---|
| Go toolchain | Go 1.27.1 |
| Operating system | Linux 7.2.6 |
| Architecture | amd64 |
| CPU | 12th Gen Intel Core i7-1255U, 12 threads |
| Document size | 10 000 cues |

## Results before the fix

Times are nanoseconds per operation. The parse and the render run one format at a time, and the conversion runs the reader and the writer of each target.

| Format | Parse | Render | Convert to the format |
|---|---:|---:|---:|
| `ass` | 8.34 ms | 10.11 ms | 23.83 ms |
| `json1` | 37.86 ms | 20.71 ms | 41.66 ms |
| `kdenlive` | 10.64 ms | 13.75 ms | 33.29 ms |
| `sbv` | 4.80 ms | 7.79 ms | 22.53 ms |
| `srt` | 8.59 ms | 9.64 ms | not measured |
| `srv3` | 17.21 ms | 22.83 ms | 38.04 ms |
| `ttml` | 29.00 ms | 10.95 ms | 25.26 ms |
| `vtt` | 19.40 ms | 26.96 ms | 45.63 ms |
| `ytt` | 17.10 ms | 23.20 ms | 37.89 ms |

## The bottlenecks

The CPU profile gives most of its samples to the Go runtime, and the allocation profile names the callers. The findings are:

1. The WebVTT reader and writer built a character reference replacer for every cue. A replacer holds a trie, so the build cost time and memory. This finding is fixed, and the section below carries the numbers.
2. Garbage collection and memory movement dominate the run. `runtime.memmove`, `runtime.scanObjectsSmall`, and `runtime.growslice` together take about a quarter of the CPU time. The cause is the allocation count, which reaches 330 046 allocations for the parse of one WebVTT document.
3. `internal/envelope.ReadLines` allocates a slice of lines for the whole file. It sits high in the allocation profile, and it serves every read of a plain format.
4. `model.RubyGroups` builds a fresh grouping on each call. The YTT and SRV3 writers call it per cue, so a document with ruby text pays the cost many times.
5. The JSON1 reader and writer lean on the encoding library, which shows in `encoding/json` on both profiles. JSON1 is a debugging format rather than a hot path, so the cost stays documented.

## The fix

The WebVTT reader and writer called `strings.NewReplacer` inside `decodeEntities` and `encodeText`, so every cue payload built a replacer trie and dropped it at once. The fix moves the two replacers to package-level values, where they are built once. A test in `internal/formats/vtt/vtt_test.go` keeps the behaviour of both functions.

| Benchmark | Before | After |
|---|---:|---:|
| Parse, `vtt` | 19.40 ms, 330 046 allocs | 7.66 ms, 130 046 allocs |
| Render, `vtt` | 26.96 ms, 140 158 allocs | 8.48 ms, 70 041 allocs |

The parse is about two and a half times faster, and the render is about three times faster. The allocation count falls by more than half in both cases. The fix is the same change that the profile points at, and no test lost coverage.

## The interactive path

The interactive mode reads its configuration and builds its keymap before the first question. Both steps are small, and the table records them so a later change is visible.

| Benchmark | Result |
|---|---:|
| Decode of the default configuration | 39.8 µs, 135 allocs |
| Keymap build | 1.82 µs, 41 allocs |
| Whole-line key match | 8.21 ns, no alloc |
| One key-sequence step | 14.30 ns, no alloc |

The decode of the configuration costs about 40 microseconds on every startup that reads a file. The keymap builds once per interactive session. The match and the step of the matcher allocate nothing, so a typed key costs no garbage collection. The first finding is a candidate for a later change: a cached schema, or a lighter reader, can remove part of the decode cost.

## Open items

The remaining findings stay documented rather than fixed, because each one needs a larger change than the v1.0.0 window allows. They are candidates for a later release:

- The allocation count of the readers is high. A reader that reuses a line buffer and a span slice can cut the garbage collection cost. The YTT and SRV3 reader pair shares one parse path, so one change serves both.
- `envelope.ReadLines` builds a full slice of lines. A streaming reader can remove that allocation, and it changes the shape of the block handling.
- `model.RubyGroups` builds a fresh grouping per call. A cached grouping on the document, or a grouping that writes into a caller slice, can remove the repeated work.
- `strace` and `perf` on the command show a startup cost from the registry and the configuration load. The cost is small beside a conversion, so it stays.

A later audit can measure each item with the same tools. The `bench.txt` baseline makes a regression visible.
