# Commands

Pass `--help` to any command for full flag reference.

## Basics

| Command | Purpose |
|---|---|
| `obmr use <plan>` | Set default plan in `./.obmr/config.yaml`. Walks up parents like `.git`. |
| `obmr init [--parent DIR]` | Clone modules as siblings; write `.obmr.lock`. |
| `obmr status` | Active plan, mode, local/canonical divergence, per-module branch + dirty flag. |
| `obmr run [--prod] [-- ob-args]` | Invoke `ob run` via `uv` (or `pixi` for conda backend). |
| `obmr dev [--fork]` | Write `bench.local.yaml`; switch clean modules to `origin/HEAD`; with `--fork`, ensure a `fork` remote per module. |
| `obmr list [--origin]` | Print modules. Reads local YAML in dev mode; `--origin` reads canonical. |

## Git fan-out

| Command | Purpose |
|---|---|
| `obmr checkout <branch> [-b]` | Concerted checkout (`-b` to create). |
| `obmr pull` | `git pull --ff-only` per module. |
| `obmr push` | Push current branch to `fork` if present else `origin`; skip clean modules. |
| `obmr foreach -- <cmd>` | Run a shell command in every module dir. |
| `obmr trim [--branch NAME] [--force]` | Delete local branches merged into `origin/HEAD`. |

## Plan YAML (`obmr plan ...`)

| Command | Purpose |
|---|---|
| `obmr plan fmt [path] [--local]` | Reformat a plan YAML in place (preserves comments). |
| `obmr plan pin [--ref REF]` | Rewrite canonical commit SHAs from `origin/<ref>`. Default ref: `origin/HEAD`. |
| `obmr plan promote` | Copy local YAML edits back into canonical (urls/commits restored). |

## Configuration

`obmr config` reads/writes `./.obmr/config.yaml` git-config-style.

| Key | Meaning |
|---|---|
| `default.plan` | Path to the active benchmark YAML. Set by `obmr use`. |
| `omnibenchmark.pr` | PR number on `omnibenchmark/omnibenchmark` to install (highest priority). |
| `omnibenchmark.branch` | Branch on `omnibenchmark/omnibenchmark` to install. |
| `omnibenchmark.version` | Pinned PyPI version. |

```sh
obmr config                              # list everything
obmr config omnibenchmark.branch         # print value
obmr config omnibenchmark.branch dev     # set value
obmr config --unset omnibenchmark.branch # clear
```

Resolution priority for `omnibenchmark.*`: `pr > branch > version > pypi-latest`.
