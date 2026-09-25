# swag contributor rules

`swag` (Subtitles With A Gopher) is a clean-room Go tool for reading, writing, and converting subtitle files. We credit [YTSubConverter](https://github.com/arcusmaximus/YTSubConverter) as the inspiration for the supported feature set. We wrote all code from scratch. Read [the third-party notices](docs/third-party-notices.md) for the attribution rules.

The architecture and the intermediate representation live in [the architecture page](docs/architecture.md). The milestones live in [the roadmap](ROADMAP.md).

## Documentation language: use the vendored simple-english skill

1. Load the skill in [the skill entry point](skills/simple-english/SKILL.md) before you write or change any prose. Prose means: `README.md`, everything under `docs/`, CLI help text, comments that ship to users, commit messages, and pull request descriptions.
2. The skill is vendored in this repository under `skills/`. Do not fetch it from the network. Do not edit the upstream text except through the documented project override.
3. The project override in this repository is British English spelling. Apply [the British English spelling rules](skills/simple-english/references/spelling.md) to all prose. Keep identifiers, flags, format names, and quoted errors unchanged.
4. Follow the skill in two places: documents follow The Document rules, and your chat replies follow The Reply rules.
5. For changelogs, release notes, and error messages, read [the use cases reference](skills/simple-english/references/use-cases.md) before you draft.
6. When you check existing text, follow the CHECK mode in the skill. Quote the rule number from [the rule catalogue](skills/simple-english/references/rule-catalog.md) for each finding.
7. Do not write em-dashes or en-dashes anywhere. This rule covers code comments, documentation, commit messages, release notes, and the chat reply. Write two sentences, use a colon, or name the relation with a word such as "because".
8. Every markdown link carries a human-readable alias. Never use a bare file path, file name, or URL as the link text.

## Code rules

1. Keep all subtitle knowledge inside `internal/formats` and `pkg/sub`. Commands in `cmd/swag` wire packages together. They contain no format logic.
2. A new format is a package under `internal/formats/<format>` with a `Reader`, a `Writer`, and a `Name() string`. Register both in the format registry.
3. Every fix to platform quirks carries a test that fails without the fix. Reference the platform behaviour in a comment when the reason is not obvious.
4. Keep the public API of `pkg/sub` small. Prefer unexported fields with constructors and accessor methods when invariants exist.
5. Errors describe what failed and name the file or cue. Use `fmt.Errorf("parse %s: %w", path, err)` wrapping, never bare `errors.New` for wrapped causes.
6. Run `gofmt`, `go vet ./...`, and `go test ./...` before you report work as done.

## Tests and coverage

1. The project must keep 100% statement coverage across the module. Run `go test -cover ./...` to measure. The floors in the `Makefile` and in CI enforce it, and they must stay at 100%.
2. Table-driven tests are the default. Put one test file next to each package.
3. Reader and writer pairs get round-trip tests with fixture files under `internal/formats/<format>/testdata/`.
4. New code needs tests in the same change. Coverage must not drop below 100%. When a statement is impossible to reach, change the code until it is reachable or remove it. Do not leave a dead branch behind.
5. Every shipped writer is covered by the cross-check suite in `internal/formats/ass/crosscheck_test.go`. Add a new writer to that suite in the same change.

## Internationalisation

1. User-facing strings come from the message catalogue in `internal/i18n`. The CLI never formats display text inline.
2. The default locale is `en-GB`. New messages land in the catalogue in the same change that uses them.
3. Do not build sentences by concatenation. Give translators a whole string with one placeholder.

## Changelog, roadmap, and docs

1. Release notes live under `docs/changelogs/`, one file per release, with [the changelog index](docs/changelogs/index.md) as the index. They follow [the Keep a Changelog format](https://keepachangelog.com/en/1.1.0/) 1.1.0. The project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
2. Unreleased work lands in the file for the next planned version. The release commit moves the status to released, adds the date, and adds the link to the index.
3. [The roadmap](ROADMAP.md) carries the milestones. Tick a checkbox in the same change that completes the work behind it. Move a deferred item to the changelog when you defer it.
4. User-facing documentation lives under `docs/`, with [the documentation index](docs/index.md) as the index. One topic per file, linked from the index.
5. A release links to its changelog file. The GoReleaser release body and the `make tag` message both carry the link.

## Build and release

1. Release builds compile with `CGO_ENABLED=0` and Go linker flags `-s -w`. The configuration lives in [the GoReleaser configuration](.goreleaser.yml).
2. `go install github.com/bladeacer/swag/cmd/swag@latest` must keep working. Do not add build tags or dependencies that break a plain `go install`.
3. Local development uses air with the configuration in [the air configuration](.air.toml). `make watch` starts it.
4. CI must run `go vet`, `go test -cover ./...`, and `goreleaser check` on every pull request.
5. `make tag` suggests the highest version in `docs/changelogs/`. Keep that behaviour, and keep the override prompt.

## Licence and attribution

1. New source files carry the Apache-2.0 header comment.
2. Do not copy code, comments, or data tables from YTSubConverter or its forks. Behaviour can match, expression must be our own. See [the third-party notices](docs/third-party-notices.md).
3. Keep [the third-party notices](docs/third-party-notices.md) current when a dependency, a vendored work, or an inspiration source changes.
