# obflow

A small CLI for managing an [omnibenchmark](https://github.com/omnibenchmark/omnibenchmark)
and its module repos as a workspace of sibling clones — driven by the
benchmark YAML.

## What it does

- Clones every module declared in a benchmark YAML as a sibling directory.
- Generates a dev-mode `bench.local.yaml` whose URLs point at your clones.
- Fans out git operations (`checkout`, `push`, `pull`, ...) across modules.
- Round-trips edits between the local and canonical YAML (`plan promote`,
  `plan pin`).
- Invokes `ob run` via `uv` (or `pixi` for the conda backend).

## Mental model

The canonical `bench.yaml` is the spec: upstream URL + SHA per module,
git-tracked, reproducible. Dev mode generates a `bench.local.yaml` with
local paths so you can edit modules in place. Branches and forks never
appear in canonical.

[Install →](install.md){ .md-button }
[Quickstart →](quickstart.md){ .md-button .md-button--primary }
