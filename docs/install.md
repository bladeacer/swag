# Install

This page covers the ways to get `swag` and the ways to check that it works. [The usage page](usage.md) covers the command line.

## Requirements

- Go 1.24 or newer, for a build from source and for `go install`.
- No compiler at run time and no cgo. Release builds set `CGO_ENABLED=0`, so the binary runs on a machine with no Go toolchain.

The tool reads and writes plain files. It opens no network connection and needs no service.

## Install the command with Go

```sh
go install github.com/bladeacer/swag/cmd/swag@latest
```

The command lands in the bin directory of the module cache prefix, which is `$GOBIN` or `$HOME/go/bin` by default. Add that directory to `PATH` when it is not there already.

A plain `go install` adds no build tag and no code generation step, so it keeps working for every release.

## Install a release archive

Every release carries prebuilt archives for Linux, macOS, and Windows. Open [the releases page](https://github.com/bladeacer/swag/releases), download the archive for your system, and unpack it. The archive holds the `swag` binary, the licence, and the readme. Each release links its changelog file.

The release build strips the symbol table and the DWARF data with the linker flags `-s -w`, so the archive is smaller than a local build. [The GoReleaser configuration](../.goreleaser.yml) carries the build settings.

## Build from source

```sh
git clone https://github.com/bladeacer/swag.git
cd swag
make build      # writes bin/swag
make install    # runs go install on the module
```

A direct build works too:

```sh
go build -o bin/swag ./cmd/swag
```

## Check the install

```sh
swag --version          # the version, the commit, and the build date
swag --help             # every flag
swag -i in.srt -o out.sbv -v
```

A bare `swag` prints the banner and the first step, and it exits 0. That is what the development loop runs, so `air` and `make run` open the banner instead of failing on the missing input flag.

## Development tools

`make tools` installs the tools this repository uses:

- [air](https://github.com/air-verse/air) for the hot reload loop
- [goreleaser](https://goreleaser.com) for the release build
- [go-test-coverage](https://github.com/vladopajic/go-test-coverage) for the coverage badge
- [benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat) for the benchmark comparison

The daily loop:

```sh
make watch          # air rebuilds and runs bin/swag on every save
make test           # every test with a coverage summary
make cover-verify   # fails when module coverage drops below 100 percent
make bench          # the ten-thousand-cue benchmarks
```

[The testing notes](testing.md) cover the coverage floor, the fuzz targets, and the benchmarks in detail. The air settings live in [the air configuration](../.air.toml).

## Troubleshooting

| Symptom | Cause and fix |
|---|---|
| `missing flags: --input=STRING` and exit code 1 | An older build. A bare run prints the banner and exits 0 from v0.7.0. |
| `build.bin is deprecated` from air | Air renamed the key. [The air configuration](../.air.toml) uses `entrypoint`. |
| `make run` exits 1 | Same cause. `make run` passes a bare separator, which now counts as a bare run. |
| `swag` is not found after `go install` | The bin directory of the module cache prefix is not on `PATH`. |
