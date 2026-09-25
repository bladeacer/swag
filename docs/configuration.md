# Configuration

`swag` resolves one configuration file per user, and it follows the convention of the platform. This page records where the file lives, how to move it, and which settings it carries. The tool reads the file on every run, and a missing file changes no behaviour.

## The location

| Platform | Directory | File |
|---|---|---|
| Linux and the other Unix systems | `$XDG_CONFIG_HOME/swag`, or `$HOME/.config/swag` when the variable is empty | `config.toml` |
| macOS | `~/Library/Application Support/swag` | `config.toml` |
| Windows | `%AppData%\swag` | `config.toml` |

The rules come from the freedesktop basedir specification on Linux and from the platform convention on macOS and Windows. [The basedir specification](https://specifications.freedesktop.org/basedir-spec/latest/) names the `$XDG_CONFIG_HOME` variable and its fallback.

## Overrides

Two environment variables move the file:

| Variable | Meaning |
|---|---|
| `SWAG_CONFIG_DIR` | The configuration directory. It wins over the platform rule. |
| `SWAG_CONFIG` | The configuration file. It wins over the directory and over the platform rule. |

A portable install therefore keeps its settings beside the binary:

```sh
SWAG_CONFIG_DIR=./config swag config
```

## Report the location

The `config` command prints the resolved path, whether the file is present, and the platform of the build:

```sh
swag config
```

```
 INFO  The configuration file is /home/user/.config/swag/config.toml.
 INFO  The file is absent, so the tool uses its own defaults.
 INFO  This build runs on linux/amd64.
```

## Write the default file

`swag config --init` writes the commented default file to the resolved path and creates the directory. The command refuses to replace an existing file, so a hand-edited file never disappears:

```sh
swag config --init
```

The written file holds every setting as a comment, so writing it changes no behaviour until you edit it.

## Precedence

One setting can arrive from four places. The first source that carries a value wins:

1. The command line.
2. The environment, for example `SWAG_LOCALE`.
3. The configuration file.
4. The built-in default.

## The file format

The file is TOML, and every setting is optional. An unknown key is an error, so a typo never passes in silence. The written file carries the schema:

```toml
# The swag configuration file.
#
# Every setting is optional. A setting that is absent keeps the default of the
# tool, so this file starts with every setting commented out.
#
# The command line wins over the environment, the environment wins over this
# file, and this file wins over the built-in default.

# The message locale. The shipped locales are en-GB and en-US.
# locale = "en-US"

# Print the conversion report, as the -v flag does.
# verbose = true

# Fail a conversion that drops a feature, as the -s flag does.
# strict = true

# Write no integrity block, as the -c flag does.
# strict_compat = true

# Replace the font of every style and span, as the -n flag does.
# font = "Verdana"

# The input format, as the -F flag does. An empty value detects the format.
# from = "ass"

# The target format, as the -f flag does. An empty value uses the output
# extension.
# format = "vtt"

# The target formats that a batch run writes when -f is absent. The same list
# orders the target question of the interactive mode. Every name must be a
# registered format.
# preferred = ["srt", "vtt"]

# The conversions that run at once in a batch run, as -j does. The value
# keeps two cores free when it is absent, and zero uses every core.
# jobs = 4

# The keys of the interactive mode. Each entry maps an action onto one key. An
# empty value removes the binding. A modifier joins the parts with +, so
# ctrl+shift+r is one key, and <leader> stands for the leader key. The actions
# are accept, help, and quit, and the leader names the leader itself.
# [keybinds]
# leader = "ctrl+x"
# accept = "<leader>a"
# help = "<leader>?"
# quit = "<leader>q"
```

| Setting | Type | Meaning |
|---|---|---|
| `locale` | string | The message locale, as `--locale` does. |
| `verbose` | bool | Print the conversion report, as `--verbose` does. |
| `strict` | bool | Fail a conversion that drops a feature, as `--strict` does. |
| `strict_compat` | bool | Write no integrity block, as `--strict-compat` does. |
| `font` | string | Replace the font, as `--font` does. |
| `from` | string | The input format, as `--from` does. |
| `format` | string | The target format, as `--format` does. The interactive command reads the same setting from `--target`. |
| `preferred` | list of strings | The target formats of a batch run with no `-f`, and the first choices of the interactive picker. |
| `jobs` | integer | The conversions that run at once in a batch run, as `--jobs` does. |
| `keybinds` | table | The keys of the interactive mode, as [the keybind section](#the-keys-of-the-interactive-mode) records. |

## The keys of the interactive mode

The interactive mode reads one answer per line, and a binding runs an action before an answer arrives. The `accept` action takes the default value of the question. The `help` action prints the bindings and asks the question again. The `quit` action ends the run and writes no file. The `leader` entry names the key that the other bindings start with, and it runs no action of its own.

The notation follows the terminals:

- A modifier joins the parts of a key with `+`, so `ctrl+shift+r` is one key.
- `<leader>` stands for the leader key, at the start of a key or on its own.
- The modifiers are `ctrl`, `alt`, `meta`, and `shift`. `meta` is another name for `alt`.
- The named keys are `esc`, `tab`, `space`, `backspace`, `up`, `down`, `left`, and `right`. Any other key is one character.
- `ctrl` takes a letter, or one of `@`, `[`, `\\`, `]`, `^`, `_`, and `space`. The terminals do not agree on the rest, so any other key after `ctrl` is an error. The shift of a control letter changes nothing, because a terminal sends the same byte either way.

An entry replaces the built-in binding of its action, and an empty value removes the binding. An action that the tool does not ship is an error, so a typo never leaves a binding inert.

| Binding | Default | Action |
|---|---|---|
| `leader` | `ctrl+x` | The key that starts the other bindings. |
| `accept` | `<leader>a` | Take the default value. |
| `help` | `<leader>?` | Print the bindings and ask again. |
| `quit` | `<leader>q` | End the run and write no file. |

[The internationalisation page](i18n.md) covers the locale settings, and [the usage page](usage.md) covers the flags. [The terminal palette page](terminal-palette.md) covers the theme source that sits behind the file.
