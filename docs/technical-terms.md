# Technical terms

This page fixes the meaning of each term that the project uses, in the documentation, the code, and the commit messages. One term carries one meaning (rule 1.11 of the vendored skill). When you write about the tool, use the term in the table below.

## The terms

| Term | Meaning |
|---|---|
| configuration | The values that the tool reads at startup. Write configuration, not config, settings, or options. |
| setting | One named value in the configuration, for example `font` or `jobs`. |
| flag | A command line option, for example `--font`. |
| environment variable | A value that the operating system passes to the process, for example `SWAG_LOCALE`. |
| global file | The configuration file in the user configuration directory. |
| working directory file | The file named `swag.toml` in the directory that the command runs in. |
| default file | The file that `swag config --init` writes. Every fixed setting is active with its built-in default value. |
| locale | A language tag that selects a message catalogue, for example `en-GB`. |
| message catalogue | The map from a message key to its text for one locale. |
| format | A subtitle file form, for example SubRip or WebVTT. |
| registry | The table of the registered formats. |
| reader | The part of a format package that parses a file into a document. |
| writer | The part of a format package that renders a document into a file. |
| parse | Read a file into a document. |
| render | Write a document into a file. |
| convert | Parse a source file and render the document into the target format. |
| document | The full subtitle content in the intermediate representation. |
| cue | One timed subtitle entry. |
| span | One run of text inside a cue, with its own styling. |
| style | The default look of a cue, for example the font and the size. |
| pen | A YouTube Timed Text drawing style that attaches to a run. |
| window | A YouTube Timed Text region that carries a vertical mode or a direction. |
| position | The placement and the size of a cue on the video. |
| layout | The position and the alignment of a cue. |
| karaoke | The timing of the spans inside a cue. |
| ruby | A reading that sits over or under a base text. |
| base text | The text that a ruby reading annotates. |
| annotation | The ruby span that follows a base span. |
| intermediate representation | The document model that every reader and writer shares. The short form is IR. |
| loss report | The list of features that a target format cannot carry. |
| integrity block | The block at the end of a plain file that holds the whole document. |
| envelope | The code that writes and reads the integrity block. |
| batch conversion | A conversion of every subtitle file in one directory. |
| interactive mode | The mode that asks for the input, the target, and the output. |
| binding | The key sequence that runs one interactive action. |
| chord | One key with its modifiers, for example `Ctrl+x`. |
| leader | The key that starts the other bindings. |
| key token | One item in a binding list, for example `Ctrl` or `a`. |
| source format | The format of the input file. |
| target format | The format of the output file. |
| fixture | A sample file that a test reads. |
| coverage | The share of statements that the tests run. The floor is 100 percent. |
| benchmark | A Go test that measures the time and the allocations of an operation. |
| profile | The output of the Go profiler for a benchmark or a run. |
| performance audit | The page that records the tools, the numbers, and the bottlenecks. |
| STE | Simplified Technical English, the rule set of the vendored skill. |

## Terms we do not use

The left column carries the word that the project refuses. The right column carries the word that replaces it.

| Do not write | Write |
|---|---|
| config, settings, options | configuration |
| check, verify, confirm, validate, ensure | make sure that |
| should, would, may, might, could | must or can, or delete the word |
| e.g., i.e., etc. | for example, that is, or name the items |
| in order to | to |
| prior to | before |
| utilise, leverage | use |
| functionality | function or feature |
| out of the box | by default |

## Spelling and style

[The simple English skill](../skills/simple-english/SKILL.md) carries the rules. This project overrides rule 1.14: use British English spelling. [The spelling table](../skills/simple-english/references/spelling.md) lists the common software words. Write no em-dash and no en-dash. Write two sentences instead, or name the relation.

[The contributor rules](../AGENTS.md) record when to use this page. [The documentation index](index.md) links every page.
