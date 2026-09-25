# Compatibility

This page records where `swag` runs, which players read the formats it writes, and how each claim was reached. The v0.9.0 milestone asks for the matrix, so the page carries the evidence beside the claim. [The install page](install.md) covers the ways to install the tool.

## The release targets

| Operating system | Architecture | Archive | Evidence |
|---|---|---|---|
| Linux | `amd64` | `tar.gz` | The whole test suite runs here. Continuous integration builds the binary, runs `go vet`, runs `go test -cover ./...` on the module, and runs `goreleaser check` on the release configuration. |
| Linux | `arm64` | `tar.gz` | The release snapshot builds the archive, and the toolchain compiles every package for the target. |
| macOS | `amd64` | `tar.gz` | The release snapshot builds the archive. |
| macOS | `arm64` | `tar.gz` | The release snapshot builds the archive. |
| Windows | `amd64` | `zip` | The release snapshot builds the archive. |
| Windows | `arm64` | `zip` | The release snapshot builds the archive. |
| `js/wasm` | `wasm` | none | The build id is skipped. The browser demo lands with v1.0.0, and [the roadmap](../ROADMAP.md) carries the item. |

Six archives come out of a snapshot. The checksums file covers each one, and the source archive carries the whole module.

The platform list is not a limit on where the code builds. The module carries no cgo dependency and no build tag, so the Go toolchain builds it for any target it knows:

```sh
GOOS=freebsd GOARCH=arm64 go build -o swag ./cmd/swag
```

The same property keeps `go install` working, which is the install path that [the install page](install.md) recommends for a Go user.

## The formats

Every registered format reads and writes. This table names the specification that each reader follows. It shows where the tool holds to a standard and where it follows an observed behaviour.

| Format | Extensions | The reader follows | Evidence in this repository |
|---|---|---|---|
| SubRip `srt` | `srt` | The common shape, because the format has no specification | [The format notes](formats.md) and the round trip fixtures |
| YouTube SBV `sbv` | `sbv` | [The YouTube caption support page](https://support.google.com/youtube/answer/2734698) | The round trip fixtures |
| YouTube Timed Text `ytt` | `ytt` | The YouTube upload quirks, because YouTube publishes no schema | [The format notes](formats.md) list each quirk and its reason |
| YouTube SRV3 `srv3` | `srv3` | The same dialect as `ytt` | One reader and one pen model for the pair |
| Advanced SubStation Alpha `ass` | `ass`, `ssa` | [The ASS override tag reference](https://aegisub.org/docs/latest/ass_tags/) and [libass](https://github.com/libass/libass) | [The ASS support page](ass-support.md) maps each tag onto its field and its test |
| WebVTT `vtt` | `vtt` | [The WebVTT specification](https://www.w3.org/TR/webvtt1/) | The round trip fixtures and the cross-check suite |
| TTML `ttml` | `ttml`, `dfxp` | [TTML2](https://www.w3.org/TR/ttml2/) for the write, and the YouTube dialect for the read | The round trip fixtures |
| Kdenlive JSON `kdenlive` | `kdenlive` | The fields that Kdenlive writes, because the format has no schema | [The format notes](formats.md) |
| Lossless JSON `json1` | `json1` | [The JSON1 page](json1.md), which is the specification | The version policy and the migration tests |

Two properties hold for every pair:

- A conversion keeps every feature that the target can carry, and the writer reports every feature that it cannot. [The loss report review](loss-report.md) lists the entries.
- A lossy write to SubRip, SBV, WebVTT, or TTML appends [the integrity block](integrity.md), so a later `swag` read restores the whole document. A tool outside `swag` ignores the block. Pass `--strict-compat` when a target tool must never meet it.

## The players

The format decides which player reads a file. This table names the players that document the format as an input. The project runs no player test suite. The column is a statement about the format and about the evidence above, and not a per-player test result.

| Player | Formats it documents | Notes |
|---|---|---|
| VLC | `srt`, `sbv`, `ass`, `ssa`, `vtt`, `ttml` | It reads the plain and the styled cues of each. |
| mpv | `srt`, `ass`, `ssa`, `vtt` | It renders ASS through libass. |
| Aegisub | `ass`, `ssa`, `srt` | It is the reference editor for the ASS tag set. |
| A web browser | `vtt` | A `<track>` element takes WebVTT. |
| YouTube | `sbv`, `ytt`, `srv3`, `ttml` | The upload path takes those four. |
| Kdenlive | `kdenlive` | It keeps a subtitle track in its own JSON shape. |
| `swag` itself | every registered format | JSON1 carries the whole document. |

Three limits follow from the list, and each one has a workaround:

- A player outside the list reads none of the formats. A conversion to a plain format in the list, with `--strict-compat`, is the portable answer.
- The YouTube path rewrites some values, so a write is not byte-identical to its source. [The format notes](formats.md) name each rewrite.
- Kdenlive holds no integrity block, so a round trip through Kdenlive keeps the cues and reports the rest as a loss. A conversion through JSON1 keeps the content.

## Make sure that a conversion carries every feature

The `--strict` flag turns a dropped feature into a failure, so it answers the question "does this target carry everything?" for one file:

```sh
swag -i in.ass -o out.vtt --strict
```

The command exits 1 and names every dropped feature when the target is too narrow. Drop the flag and the same conversion succeeds with a report, which `-v` prints.

These three commands work on every platform:

```sh
swag --version     # the version, the commit, and the build date
swag config        # the configuration path and the platform of the build
swag -i in.srt -o out.vtt -v   # a conversion with its loss report
```

The version line names the platform through `swag config`, which reports the values that the Go toolchain stamped into the binary. A release archive therefore says which build it is without a separate table.
