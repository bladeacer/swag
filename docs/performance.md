# Performance audit

This page records the performance audit of the v1.0.0 release. It names the tools, the test machine, the numbers, the bottlenecks, and the work that stays open. The audit covers the parse, the render, and the conversion of every registered format.

## Method

The benchmarks live in `pkg/sub/bench_test.go`. Each one builds a document of ten thousand cues, then parses it, renders it, or converts it. The benchmark results are the first evidence.

- `make bench` runs the full set with memory numbers.
- `make bench-all` runs the benchmarks of every package that carries one, including the configuration and the keybind engine.
- `make bench-save` writes a baseline to `bench.txt`, and `make bench-compare` compares a new run with it through `benchstat`.
- `make profile-cpu` writes `cpu.prof` from the benchmark run. Read it with `go tool pprof -top cpu.prof`.
- `make profile-trace` writes `trace.out`. Read it with `go tool trace trace.out`.

Three more tools cover the command itself, so the wall time, the syscalls, and the counters of one run are visible:

- `hyperfine` runs the release binary many times and reports the mean, the spread, and the range.
- `strace -c` counts the syscalls of one conversion and reports the time each name takes.
- `perf stat` reports the task-clock time, the page faults, and the hardware counters of the run.

The machine has two kinds of core, so `perf stat` reports `cpu_core` and `cpu_atom` counters side by side. The trace gives two derived profiles through `go tool trace -pprof=sched` and `-pprof=syscall`.

A benchmark number is noisy, because a single run can hit a scheduling pause. The tables below come from one full run on one machine. Treat a difference under ten percent as noise. The WebVTT fix below is far above that line.

## Test machine

| Item | Value |
|---|---|
| Go toolchain | Go 1.27.1 |
| Operating system | Linux 7.2.6 |
| Architecture | amd64 |
| CPU | 12th Gen Intel Core i7-1255U, 12 threads |
| Document size | 10 000 cues |

## The full run

Times are milliseconds per operation. The parse and the render run one format at a time, and the conversion runs the reader and the writer of each target. Every cell now carries a number, including the conversion into SubRip itself.

| Format | Parse | Render | Convert to the format |
|---|---:|---:|---:|
| `ass` | 8.21 ms | 9.62 ms | 19.65 ms |
| `json1` | 36.51 ms | 19.31 ms | 28.54 ms |
| `kdenlive` | 10.06 ms | 13.99 ms | 28.25 ms |
| `sbv` | 4.22 ms | 7.88 ms | 17.96 ms |
| `srt` | 6.16 ms | 9.94 ms | 20.01 ms |
| `srv3` | 15.32 ms | 23.88 ms | 33.10 ms |
| `ttml` | 28.13 ms | 11.63 ms | 20.80 ms |
| `vtt` | 7.23 ms | 9.09 ms | 18.88 ms |
| `ytt` | 15.48 ms | 23.77 ms | 33.08 ms |

## The bottlenecks

The CPU profile gives most of its samples to the Go runtime, and the allocation profile names the callers. The findings are:

1. The WebVTT reader and writer built a character reference replacer for every cue. A replacer holds a trie, so the build cost time and memory. This finding is fixed, and the section below carries the numbers.
2. Garbage collection and memory movement dominate the run. `runtime.memmove`, `runtime.memclrNoHeapPointers`, and `runtime.scanObjectsSmall` together take about a sixth of the CPU time. The allocation count is the cause.
3. `runtime.growslice` sits at the top of the cumulative profile, which reached one sixth of the samples before the reader fix. A reader appended into a slice that grew many times, and a writer still does. The reader fix below removes the intermediate span slice.
4. `internal/envelope.ReadLines` allocates a slice of lines for the whole file. It sits high in the allocation profile, and it serves every read of a plain format.
5. `model.RubyGroups` builds a fresh grouping on each call. The YTT and SRV3 writers call it per cue, so a document with ruby text pays the cost many times.
6. The JSON1 reader and writer lean on the encoding library, which shows in `encoding/json` on both profiles. JSON1 is a debugging format rather than a hot path, so the cost stays documented.
7. The trace puts the scheduler delay at `runtime.systemstack_switch`, which takes about three fifths of the delay samples. The garbage collector follows at about one quarter of the cumulative samples. The blocking profile of the trace shows one syscall of note, the write of the output, and it takes about six milliseconds.

## The fix

The WebVTT reader and writer called `strings.NewReplacer` inside `decodeEntities` and `encodeText`, so every cue payload built a replacer trie and dropped it at once. The fix moves the two replacers to package-level values, where they are built once. A test in `internal/formats/vtt/vtt_test.go` keeps the behaviour of both functions.

| Benchmark | Before | After |
|---|---:|---:|
| Parse, `vtt` | 19.40 ms, 330 046 allocs | 7.66 ms, 130 046 allocs |
| Render, `vtt` | 26.96 ms, 140 158 allocs | 8.48 ms, 70 041 allocs |

The parse is about two and a half times faster, and the render is about three times faster. The allocation count falls by more than half in both cases. The fix is the same change that the profile points at, and no test lost coverage.

## The reader fix

The plain readers and the shared YouTube reader built a fresh span slice for every text line. The SubRip reader also called the integer parser on every text line, and the parser built an error for each failed call. Four changes cut the cost:

- The SubRip span splitter appends into the span slice of the cue, so a line builds no intermediate slice.
- The SubRip counter test rejects a non-digit line before it calls the parser, so a text line builds no error.
- The SubRip and SBV timestamp parsers find the two colons instead of splitting the string, so a cue builds no slice for the parts.
- The YouTube reader pre-sizes its cue list and its three scratch slices, so the shared YTT and SRV3 path grows them at most once.

| Benchmark | Before | After |
|---|---:|---:|
| Parse, `srt` | 8.52 ms, 140 038 allocs, 14.9 MB | 6.16 ms, 70 038 allocs, 10.1 MB |
| Parse, `sbv` | 4.86 ms, 50 046 allocs, 9.9 MB | 4.22 ms, 30 046 allocs, 8.9 MB |
| Parse, `ytt` | 17.00 ms, 200 123 allocs, 17.8 MB | 15.48 ms, 190 105 allocs, 15.1 MB |
| Parse, `srv3` | 17.36 ms, 200 123 allocs, 17.8 MB | 15.32 ms, 190 105 allocs, 15.1 MB |

The SubRip parse allocates half as many objects. The SBV parse allocates about two fifths fewer. The YouTube pair allocates about five percent fewer objects and fifteen percent fewer bytes. The YTT and SRV3 readers share one path, so one change serves both.

## The interactive path

The interactive mode reads its configuration and builds its keymap before the first question. Both steps are small, and the table records them so a later change is visible.

| Benchmark | Result |
|---|---:|
| Decode of the default configuration | 49.2 µs, 223 allocs |
| Cached load of an unchanged file | 618 ns, 2 allocs |
| Cached merge of two unchanged files | 1.12 µs, 4 allocs |
| Keymap build | 1.86 µs, 41 allocs |
| Whole-line key match | 8.50 ns, no alloc |
| One key-sequence step | 14.79 ns, no alloc |

The decode of the configuration costs about fifty microseconds on a startup that reads a file. The default file now carries an active entry for every fixed setting, so it is a little larger and the decode is a little heavier than the first measurement. The keymap builds once per interactive session. The match and the step of the matcher allocate nothing, so a typed key costs no garbage collection.

[The settings suite](../internal/config/settings_test.go) keeps both caches honest: an edit between two loads reaches the caller. The decode cache serves the second and later loads of an unchanged path in about six tenths of a microsecond. The merge cache layers two unchanged files in about one microsecond, so a run that resolves the pair twice skips the merge. The first decode of a path still carries the full cost, and a lighter reader stays a candidate for a later release.

## The command line

The command runs one conversion of the ten-thousand-cue SubRip document in about twenty-five milliseconds. The binary is a release build, so the linker flags `-s -w` and `CGO_ENABLED=0` are in force.

| Command | Mean | Spread | Range |
|---|---:|---:|---:|
| Convert `srt` to `srt` | 26.1 ms | 2.7 ms | 21.9 to 31.9 ms |
| Convert `srt` to `vtt` | 24.3 ms | 2.8 ms | 19.1 to 28.8 ms |
| Bare run | 3.1 ms | 0.9 ms | 2.0 to 4.8 ms |

`hyperfine` ran each command twenty times after three warm-up runs. The bare run opens the banner and exits, so it measures the startup alone. The startup is about one tenth of a conversion, and it covers the flag parse, the message catalogue, and the configuration load.

`perf stat` reports the counters of a conversion:

| Counter | Value |
|---|---:|
| Task clock | 35.07 ms |
| Page faults | 3 425 |
| Context switches | 0 |
| CPU migrations | 0 |
| Cycles, `cpu_core` | 73.7 M |
| Instructions, `cpu_core` | 233.5 M |
| Cache misses, `cpu_core` | 158 015 |
| Cycles, `cpu_atom` | 75.0 M |
| Instructions, `cpu_atom` | 187.2 M |
| Cache misses, `cpu_atom` | 191 239 |

The table carries the mean of five runs. The command runs on one core and never migrates. The instruction count is about two to three times the cycle count on each core, which shows the superscalar width of the machine. The cache misses are low, because the document fits in the last-level cache.

`strace -c` counts 1 098 syscalls in the traced run, and 50 of them report an error. The count moves between runs, because the Go runtime starts a different number of threads. The tracer slows every syscall, so the times below name the shape of the calls rather than the cost of the untraced command:

| Syscall | Share of the traced time | Calls |
|---|---:|---:|
| `futex` | 70.4 percent | 322 |
| `nanosleep` | 19.0 percent | 160 |
| `tgkill` | 1.9 percent | 39 |
| `sched_yield` | 0.8 percent | 54 |

Almost all of the traced time sits in the scheduler and the sleep of the Go runtime, which is the shape of a short-lived process. The file work is small: the run reads one input and writes one output.

## Open items

The remaining findings stay documented rather than fixed, because each one needs a larger change than the v1.0.0 window allows. They now carry their own milestone, so the work is tracked rather than lost. [The roadmap](../ROADMAP.md) records [the v1.1.0 items](../ROADMAP.md#v110-performance-and-the-smaller-fixes). The list is:

- The writers append into slices that grow many times. A pre-sized buffer or a reused scratch slice can remove the growth. The reader fix above shows the shape of the change.
- `envelope.ReadLines` builds a full slice of lines. A streaming reader can remove that allocation, and it changes the shape of the block handling.
- `model.RubyGroups` builds a fresh grouping per call. A cached grouping on the document, or a grouping that writes into a caller slice, can remove the repeated work.
- The first decode of a configuration path carries its full cost. A lighter reader can cut the startup of a run that reads a file.
- `perf` and `strace` on the command show a startup cost from the registry and the configuration load. The cost is small beside a conversion, so it stays measured rather than removed.

A later audit can measure each item with the same tools. The `bench.txt` baseline makes a regression visible.
