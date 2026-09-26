# swag documentation

Documentation for `swag` (Subtitles With A Gopher), a clean-room Go tool for reading, writing, and converting subtitles. Every page follows the vendored `simple-english` skill with British English spelling.

Getting started:

- [the install page](install.md): the ways to install the command, the platform support, and the tools for the development loop
- [the usage page](usage.md): the flags, the interactive mode, the batch conversion, the preview, the format detection, and the exit codes
- [the configuration page](configuration.md): where the configuration file lives on each platform, how to move it, and the available settings
- [the default configuration file](../swag.toml): the file that `swag config --init` writes, with the built-in keybinds as active entries
- [the library guide](library.md): the public API with a worked example, and the v1.0.0 stability guarantee
- [the migration guide](migration.md): how to move a YTSubConverter workflow onto `swag`

Formats and fidelity:

- [the format notes](formats.md): the support of every format, its specification, and its caveats and limits
- [the compatibility page](compatibility.md): the release targets, the format matrix, and the players that read each format
- [the loss report review](loss-report.md): the degradation of every format at a glance
- [the ASS support page](ass-support.md): the ASS feature list mapped onto the code and the tests
- [the file integrity page](integrity.md): the block that keeps a plain SubRip or SBV file lossless inside `swag`
- [the JSON1 page](json1.md): the lossless exchange format, its versions, and its migrations

Reference and governance:

- [the architecture page](architecture.md): packages, the intermediate representation, the format matrix, and the ASS tag tiers
- [the internationalisation page](i18n.md): the message catalogue, the locales, and how to add a language
- [the terminal palette page](terminal-palette.md): the OSC query for the terminal colours, what the prototype found, and the plan for theming
- [the performance audit](performance.md): the tools, the numbers of every format, the bottlenecks, and the fixes
- [the browser demo](demo/index.html): the WebAssembly build that converts a document in the browser
- [the testing notes](testing.md): coverage, fuzzing, and benchmarks
- [the roadmap](../ROADMAP.md): scope and the ordered milestones to v1.0.0
- [the contributor rules](../AGENTS.md): conventions for agent contributors, including the documentation language
- [the third-party notices](third-party-notices.md): the works this project builds on and their licences

Release notes, in Keep a Changelog form:

- [the changelog index](changelogs/index.md): every release
- [the v1.0.0 notes](changelogs/v1.0.0.md): the stable release, the frozen API, the browser demo, and the performance audit
- [the v0.9.0 notes](changelogs/v0.9.0.md): the configuration file, the strict compatibility flag, the interactive keybinds, the system locale, and the parallel batch
- [the v0.8.0 notes](changelogs/v0.8.0.md): the interactive command, the layout-diffing renderer, and the command line ergonomics
- [the v0.7.0 notes](changelogs/v0.7.0.md): the conversion API, the plain format integrity block, and the reference docs
- [the v0.6.0 notes](changelogs/v0.6.0.md): the ASS tag closure and the new formats
- [the v0.5.0 notes](changelogs/v0.5.0.md): the ASS writer release
- [the v0.4.0 notes](changelogs/v0.4.0.md): the ASS reader release
- [the v0.3.0 notes](changelogs/v0.3.0.md): the YouTube pair release
- [the v0.2.0 notes](changelogs/v0.2.0.md): the plain formats release
- [the v0.1.0 notes](changelogs/v0.1.0.md): the foundation release

[The browser demo](demo/index.html) runs the conversion of a document through the WebAssembly build. Build the assets with `make wasm`.
