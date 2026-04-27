# obmr

omnibenchmark monorepo helper. Manages a benchmark and its module repos
as a workspace of sibling clones, driven by the benchmark YAML.

See [docs/spec.md](docs/spec.md) for the full workflow and design rules.

## Build

```sh
go build -o obmr .
```

## Quick start

```sh
cd ~/work
obmr use ~/lab/some-bench/bench.yaml   # remember the active plan
obmr init                              # clone modules to ../some-bench-modules
obmr dev                               # write bench.local.yaml; switch modules to main
obmr run                               # uv-invoke `ob run` against bench.local.yaml
```

After `use`, the YAML arg can be omitted from every command.

## Day-to-day workflow

```sh
obmr checkout feat-x -b   # create the same branch in every module
# ... edit code in any module, run benchmark with bench.local.yaml ...
obmr status               # branch + dirty per module
obmr push                 # push to fork if present, else origin (skips clean)
# (open PRs; wait for upstream merge)
obmr pull                 # ff-only on each module's current branch
obmr pin                  # rewrite canonical SHAs from origin/HEAD
obmr trim                 # delete merged local branches
```

## Commands

| Command | Purpose |
|---|---|
| `obmr use <plan>` | Set default plan in `./.obmr/config.yaml`. |
| `obmr config [key] [value]` | Get/set config (git-config style). |
| `obmr run [--prod]` | Invoke `ob run` via `uv`; default adds `--dirty` (dev mode), `--prod` skips it. |
| `obmr list` | Print modules in the canonical YAML. |
| `obmr init [--parent DIR]` | Clone modules; write `.obmr.lock`. |
| `obmr dev [--fork]` | Write `bench.local.yaml`; switch modules to `origin/HEAD`; with `--fork`, ensure a `fork` remote per module. |
| `obmr status` | Branch + dirty per module. |
| `obmr checkout <branch> [-b]` | Concerted checkout. |
| `obmr foreach -- <cmd>` | Run a shell command in every module. |
| `obmr pull` | `git pull --ff-only` per module. |
| `obmr push` | Push current branch (fork if present, else origin). |
| `obmr pin [--ref REF]` | Rewrite canonical commit SHAs from `origin/<ref>`. |
| `obmr trim [--branch NAME] [--force]` | Delete merged local branches. |

## Layout

```
~/lab/some-bench/
  bench.yaml               # canonical (URL + SHA), git-tracked
  bench.local.yaml         # generated, gitignored
  .obmr.lock               # resolved paths
  .obmr/config.yaml        # default plan
~/lab/some-bench-modules/
  module-a/                # origin = upstream; fork (optional) = your fork
  module-b/
  ...
```

## Requirements

- `git` on PATH.
- `uv` on PATH for `obmr run`.
- `gh` on PATH and authenticated for `obmr dev --fork`.
