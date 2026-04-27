# obmr — omnibenchmark monorepo helper

A small Go CLI to manage an omnibenchmark and its module repos as a workspace
of sibling clones, driven by the benchmark YAML.

## Goals
- Parse an omnibenchmark YAML and discover its module repos.
- Clone each module into a sibling directory (default: `../bench-modules/`).
- Run concerted operations across all module repos (status, checkout, pull, push).
- Switch the YAML between "remote refs" and "local paths" without hand edits.
- Wrap `gh` for per-module PR creation tied to a shared branch.

## Non-goals (for v1)
- True monorepo history (would need `subtree`/`josh`; out of scope).
- Replacing `git submodules` for downstream consumers — `obmr` is a dev workflow tool.
- Hosting / CI integration beyond shelling out to `gh`.

## Layout
Given a benchmark file at `~/lab/<bench>/bench.yaml`, `obmr init` produces:

```
~/lab/<bench>/
  bench.yaml                 # canonical, remote-referenced
  bench.local.yaml           # generated, paths rewritten to local clones
  .obmr.lock                 # resolved modules, pinned SHAs, parent dir
~/lab/bench-modules/         # default sibling parent (configurable)
  module-a/                  # one clone per module
  module-b/
  ...
```

- Parent dir is **not** auto-namespaced per benchmark; multiple benchmarks
  sharing modules will reuse the same clone. `obmr` detects an existing clone
  and verifies its remote URL matches before reusing.
- All tool state lives in `.obmr.lock` next to the YAML. No `~/.obmr/`,
  no per-module dotfiles inside the modules.

## Commands (v1 surface)
- `obmr init <bench.yaml> [--parent ../bench-modules]`
  Parse YAML, clone missing modules, write `.obmr.lock` and `bench.local.yaml`.
- `obmr status`
  Per-module: branch, ahead/behind, dirty flag. Colored, parallel.
- `obmr foreach -- <cmd...>`
  Run a shell command in each module dir, in parallel, with per-repo prefix.
- `obmr checkout <branch> [-b]`
  Checkout (or create) the same branch in every module.
- `obmr sync`
  `git pull --ff-only` in each module, on its current branch.
- `obmr render --mode {remote,local}`
  Rewrite the YAML's module sources to remote URLs or local paths.
- `obmr lock`
  Update `.obmr.lock` with current SHAs per module.
- `obmr pr [--title ... --body ...]`
  For each dirty/ahead module: push branch, `gh pr create`. Print URLs.

## Manifest model
```go
type Module struct {
    Name   string // logical id from YAML
    Remote string // e.g. https://github.com/org/module.git
    Ref    string // branch or tag from YAML; default "main"
    Path   string // resolved local path (relative to bench dir)
}

type Lock struct {
    ParentDir string             // e.g. "../bench-modules"
    Modules   map[string]LockMod // keyed by Name
}

type LockMod struct {
    Remote string
    SHA    string
    Path   string
}
```

## YAML mode toggle
- **Remote mode (canonical)**: module entries reference git URLs + refs.
- **Local mode (generated)**: same YAML, but each module's `source`/`url`
  field rewritten to its resolved local path; `bench.local.yaml`.
- `obmr render` is pure: it never modifies the canonical file in place.
- Pipelines/tools that read the YAML are pointed at `bench.local.yaml`
  during development.

## Iterative milestones
1. **M1 — parse + clone**
   - YAML loader for the omnibenchmark schema (subset that names modules).
   - `obmr init` clones modules into `--parent` (default `../bench-modules`).
   - Writes `.obmr.lock`. No render yet.
2. **M2 — fanout**
   - `obmr status`, `obmr foreach`, parallel execution, per-repo log prefix.
3. **M3 — branch ops**
   - `obmr checkout`, `obmr sync`. Safe handling of dirty trees.
4. **M4 — render**
   - `obmr render --mode local|remote`, `bench.local.yaml`.
5. **M5 — gh integration**
   - `obmr pr`, push + `gh pr create` per module, print URLs (and optionally
     open a tracking issue linking them).
6. **M6 — polish**
   - Concurrency limits, cleaner errors, shell completion, tests on a fixture
     benchmark.

## Open questions
- Exact omnibenchmark YAML schema fields for module name + repo + ref?
  Need a real example to lock the parser.
- Should `obmr init` refuse to run if `--parent` is inside a git repo
  (to avoid accidental nesting)? Lean: yes.
- Do we need a notion of "active benchmark" to support multiple YAMLs
  reusing modules, or is one-YAML-per-cwd enough for v1? Lean: v1 = one YAML.
- `gh` auth: assume preconfigured; fail clearly if not.

## Dependencies (proposed)
- `github.com/spf13/cobra` — CLI.
- `gopkg.in/yaml.v3` — YAML.
- `github.com/go-git/go-git/v5` *or* shelling out to `git` — leaning toward
  shelling out: simpler, matches what users see, avoids go-git quirks.
- `gh` CLI on PATH for PR ops.
