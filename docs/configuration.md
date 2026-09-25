# Configuration

`swag` resolves one configuration file per user, and it follows the convention of the platform. This page records where the file lives and how to move it. [The roadmap](../ROADMAP.md) carries the file format, which lands with v0.9.0. Until then the tool reads no configuration and uses its own defaults.

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

The command creates nothing. A directory appears only when a later release writes the file.

## The file format

The file is TOML. [The roadmap](../ROADMAP.md) scopes the schema, its defaults, and its comments to v0.9.0. The planned settings carry feature parity with the command line: the default flags, the flag options, the locale, the preferred formats, the vim inspired keybinds, and the leader key.

[The internationalisation page](i18n.md) covers the locale settings, and [the usage page](usage.md) covers the flags that the file will carry.
