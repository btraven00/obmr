# Quickstart

Five minutes from a benchmark YAML to a running dev workflow.

## 1. Adopt a benchmark

```sh
cd ~/work
obflow use ~/lab/some-bench/bench.yaml
```

Writes `./.obflow/config.yaml` with the relative path to the plan. Every
subsequent `obflow` command run from this dir (or a child) defaults to
this plan, so you don't have to keep typing it.

## 2. Clone the modules

```sh
obflow init
```

Each module declared in the YAML is cloned as a sibling directory
(default parent: `../<benchdir>-modules`). A `.obflow.lock` records what
went where.

## 3. Switch to dev mode

```sh
obflow dev
```

Generates `bench.local.yaml` next to the canonical, with each module's
`url` rewritten to its local path and `commit` rewritten to its current
branch. Switches every clean module to `origin/HEAD` (typically `main`).

## 4. Run the benchmark

```sh
obflow run                  # ob run <local-yaml> --dirty
obflow run --prod           # canonical, no --dirty
obflow run -- --threads 8   # extra args pass through to ob
```

If `software_backend: conda` is set in your YAML, `obflow` builds a
[pixi](https://pixi.sh) env transparently. Otherwise `uv tool run` is
used.

## 5. Status

```sh
obflow status
```

Shows the active plan, dev/prod mode, divergence between local and
canonical (if any), and per-module branch/dirty state.

## What's next?

- [How to test a PR of omnibenchmark](howtos.md#test-a-pr-of-omnibenchmark)
- [How to push your edits back to canonical](howtos.md#promote-local-edits-to-canonical)
- [Full command reference](commands.md)
