# obflow

an opinionated omnibenchmark monorepo helper.

Manages a benchmark and its module repos as a workspace of sibling clones,
driven by the benchmark YAML.

Full docs in [`docs/`](docs/) (build with `mkdocs serve`).

## Build

```sh
go build -o obflow .
```

## Quick start

```sh
cd ~/work
obflow use ~/lab/some-bench/bench.yaml   # remember the active plan
obflow init                              # clone modules to ../some-bench-modules
obflow dev                               # write bench.local.yaml; switch modules to main
obflow run                               # uv-invoke `ob run` against bench.local.yaml
```

After `use`, the YAML arg can be omitted from every command.

## Day-to-day workflow

```sh
obflow checkout feat-x -b   # create the same branch in every module
# ... edit code in any module, run benchmark with bench.local.yaml ...
obflow status               # branch + dirty per module
obflow push                 # push to fork if present, else origin (skips clean)
# (open PRs; wait for upstream merge)
obflow pull                 # ff-only on each module's current branch
obflow pin                  # rewrite canonical SHAs from origin/HEAD
obflow trim                 # delete merged local branches
```

## Commands

| Command | Purpose |
|---|---|
| `obflow use <plan>` | Set default plan in `./.obflow/config.yaml`. |
| `obflow config [key] [value]` | Get/set config (git-config style). |
| `obflow run [--prod]` | Invoke `ob run` via `uv`; default adds `--dirty` (dev mode), `--prod` skips it. |
| `obflow list` | Print modules in the canonical YAML. |
| `obflow init [--parent DIR]` | Clone modules; write `.obflow.lock`. |
| `obflow dev [--fork]` | Write `bench.local.yaml`; switch modules to `origin/HEAD`; with `--fork`, ensure a `fork` remote per module. |
| `obflow enter <module-id> [--print]` | Open a pixi shell for a module with upstream inputs preloaded as env vars. `--print` emits `export` lines for `eval`. |
| `obflow status` | Branch + dirty per module. |
| `obflow checkout <branch> [-b]` | Concerted checkout. |
| `obflow foreach -- <cmd>` | Run a shell command in every module. |
| `obflow pull` | `git pull --ff-only` per module. |
| `obflow push` | Push current branch (fork if present, else origin). |
| `obflow plan fmt` | Reformat a plan YAML in place (preserves comments). |
| `obflow plan pin [--ref REF]` | Rewrite canonical commit SHAs from `origin/<ref>`. |
| `obflow plan promote` | Copy local YAML edits back into canonical. |
| `obflow trim [--branch NAME] [--force]` | Delete merged local branches. |

## Layout

```
~/lab/some-bench/
  bench.yaml               # canonical (URL + SHA), git-tracked
  bench.local.yaml         # generated, gitignored
  .obflow.lock               # resolved paths
  .obflow/config.yaml        # default plan
~/lab/some-bench-modules/
  module-a/                # origin = upstream; fork (optional) = your fork
  module-b/
  ...
```

## Requirements

- `git` on PATH.
- `uv` on PATH for `obflow run`, or `pixi` if conda is used.
`- `gh` on PATH and authenticated for `obflow dev --fork`.
