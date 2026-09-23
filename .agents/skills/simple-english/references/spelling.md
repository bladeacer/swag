# British English spelling (project override)

This file belongs to the project override of the vendored SimpleEnglish skill. It replaces the spelling instruction of rule 1.14 for every document written in the swag repository.

## Spelling table

Use the British spelling. Common software words:

| American | British |
|---|---|
| color | colour |
| normalize | normalise |
| serialization | serialisation |
| organize | organise |
| recognize | recognise |
| optimize | optimise |
| analyze | analyse |
| catalog | catalogue |
| dialog | dialogue (as a noun; the file format name `.ass` "dialog" in specs stays quoted) |
| license (noun) | licence (the noun; "license" stays as the verb, and in file names such as `LICENSE`) |
| center | centre |
| meter (unit) | metre |
| behavior | behaviour |
| favorite | favourite |
| traveled | travelled |
| canceled | cancelled |
| modeling | modelling |
| labeled | labelled |
| fulfill | fulfil |
| enrollment | enrolment |
| alphanumerics | alphanumerics (no change) |
| gray | grey |
| default (no change) | default |

Words that stay unchanged because they are identifiers, file names, standard names, or quoted text: `color` inside Go code, CSS colour values, the `--color` flag, `SansSerif` font names, ` behavioural` inside quoted library output.

## Grammar and style

1. Collective nouns take a plural verb when the group acts as members: "the team write their reports". Use a singular verb when the group acts as one unit: "the team is large".
2. Dates use the format `4 October 2026` in prose. Use `2026-10-04` in tables, logs, and changelog headings.
3. Quotation marks are single for quotes inside prose, and double only for quoted text inside quoted text.
4. Words such as "programme", "cheque", and "storey" follow the standard British form. Technical nouns from software ("program" as a computer term is acceptable British usage; keep "program" for code, use "programme" for a schedule or broadcast).

## Scope

The override applies to prose in documentation, changelogs, commit messages, CLI help text, and comments that ship to users. It does not apply to code identifiers, JSON keys, or output that a machine reads.
