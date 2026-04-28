# Quickstart

Five minutes from a benchmark YAML to a running dev workflow.

## 1. Adopt a benchmark

```sh
cd ~/work
obmr use ~/lab/some-bench/bench.yaml
```

Writes `./.obmr/config.yaml` with the relative path to the plan. Every
subsequent `obmr` command run from this dir (or a child) defaults to
this plan, so you don't have to keep typing it.

## 2. Clone the modules

```sh
obmr init
```

Each module declared in the YAML is cloned as a sibling directory
(default parent: `../<benchdir>-modules`). A `.obmr.lock` records what
went where.

## 3. Switch to dev mode

```sh
obmr dev
```

Generates `bench.local.yaml` next to the canonical, with each module's
`url` rewritten to its local path and `commit` rewritten to its current
branch. Switches every clean module to `origin/HEAD` (typically `main`).

## 4. Run the benchmark

```sh
obmr run                  # ob run <local-yaml> --dirty
obmr run --prod           # canonical, no --dirty
obmr run -- --threads 8   # extra args pass through to ob
```

If `software_backend: conda` is set in your YAML, `obmr` builds a
[pixi](https://pixi.sh) env transparently. Otherwise `uv tool run` is
used.

## 5. Status

```sh
obmr status
```

Shows the active plan, dev/prod mode, divergence between local and
canonical (if any), and per-module branch/dirty state.

## What's next?

- [How to test a PR of omnibenchmark](howtos.md#test-a-pr-of-omnibenchmark)
- [How to push your edits back to canonical](howtos.md#promote-local-edits-to-canonical)
- [Full command reference](commands.md)
