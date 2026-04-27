# obmr — workflow spec

`obmr` manages an omnibenchmark and its module repos as a workspace of
sibling clones. This doc describes the real end-to-end workflow,
including the uses that motivate the design.

## Mental model in one paragraph

The canonical `bench.yaml` is the spec: upstream URL + SHA per module,
git-tracked, reproducible. To develop, you clone the modules as
siblings and run the benchmark against a generated `bench.local.yaml`
whose URLs are rewritten to those local paths. You push module work to
your fork (or directly to upstream if you have access). After PRs are
merged, you pin upstream SHAs back into canonical. The canonical never
references your fork; branches never appear in the canonical.

## The workflow

### 1. Adopt a benchmark

```sh
cd ~/work
obmr use ~/lab/some-bench/bench.yaml      # writes ./.obmr/config.yaml
obmr init                                  # clones modules to ../some-bench-modules
```

`init` clones each module's canonical URL into a sibling directory
(default `../<benchdir>-modules`). Each clone's `origin` = upstream.
A `.obmr.lock` next to the YAML records what was resolved where.

### 2. Switch to dev mode

```sh
obmr dev               # shared-repo workflow (origin is writable)
obmr dev --fork        # fork workflow (creates a fork remote per module)
```

`dev` writes `bench.local.yaml` next to the canonical, with each
module's `repository.url` rewritten to its local path. The benchmark
runner consumes `bench.local.yaml` during dev. Comments in the
canonical are preserved.

`--fork` additionally runs `gh repo fork --remote=fork --clone=false`
per module, so each clone has both:
- `origin` — upstream (read; write if you have access)
- `fork`   — your writable fork (push target when present)

`gh` must be on PATH and authenticated. Idempotent: safe to re-run.

### 3. Branch across modules

```sh
obmr checkout feat-x -b
```

Creates `feat-x` in every module's working tree. The canonical YAML is
not touched — branches are a working-tree concept only.

### 4. Edit and run

Edit code in any of the sibling clones. Run the benchmark against
`bench.local.yaml`; since it points at paths, whatever you have
checked out is what runs. Iterate freely.

### 5. Push

```sh
obmr push
```

Per module: `git push -u <fork-or-origin> <current-branch>`, where
target is `fork` if that remote exists, else `origin`. Modules with no
commits ahead of upstream are skipped.

### 6. Open PRs (deferred — for now use `gh pr create` per dir)

Future `obmr pr` will fan out `gh pr create` from `fork:feat-x` to
`origin:default` for each pushed module.

### 7. After merge: pull, pin, trim

```sh
obmr pull              # git pull --ff-only on each module's current branch
obmr pin               # rewrite canonical commit SHAs from origin/HEAD
obmr pin --ref release-2.0   # or pin from a different upstream ref
obmr trim              # delete merged local branches per module
```

`pin` is **upstream-only**: it does `git fetch origin` and reads
`origin/<ref>` SHAs — never working-tree HEAD. This guarantees the
canonical's pinned SHAs are fetchable by anyone, anywhere.

`pin` rewrites `bench.yaml` in place. Review with `git diff`. It warns
if the new SHA is not a descendant of the existing one (a rewind).

`trim` per module: if the working tree is clean, delete every local
branch whose tip is merged into `origin/HEAD`. With `--branch <name>`,
delete only that branch across modules. With `--force`, use `git branch
-D` (delete unmerged branches too). Modules whose current branch would
be deleted are switched to the upstream default branch first.

## Caveats and rules baked into the design

- **Canonical never references your fork.** Forks are per-developer
  plumbing; serializing them would break shareability of the YAML.
- **Branches never go in canonical.** A branch name in `commit:` would
  break reproducibility. Use `bench.local.yaml` + working trees for
  branch experiments.
- **Two files, not a toggle.** `bench.local.yaml` is gitignored and
  always regenerable; this prevents accidentally committing a
  local-mode YAML.
- **Pin is post-merge, not post-push.** Pinning a fork-only SHA into
  canonical would produce a spec referencing a commit no one else can
  fetch.
- **Comments in canonical are preserved** across `pin`'s in-place
  rewrites (yaml.v3 Node round-trip).
- **Concerted commands fan out in parallel** per module.

## Layout

```
~/lab/some-bench/
  bench.yaml                # canonical (URL + SHA), git-tracked
  bench.local.yaml          # generated, gitignored
  .obmr.lock                # resolved paths
  .obmr/config.yaml         # default plan for commands run from here
~/lab/some-bench-modules/
  module-a/                 # origin = upstream; fork (optional) = your fork
  module-b/
  ...
```

## Command reference

| Command | Purpose |
|---|---|
| `obmr use <plan>` | Set default plan in `./.obmr/config.yaml`. Walks up parents like `.git`. |
| `obmr list` | Print modules declared in canonical. |
| `obmr init [--parent DIR]` | Clone modules as siblings; write `.obmr.lock`. |
| `obmr dev [--fork]` | Write `bench.local.yaml`; with `--fork`, also ensure a `fork` remote per module. |
| `obmr status` | Per-module branch + dirty flag. |
| `obmr checkout <branch> [-b]` | Concerted checkout. |
| `obmr pull` | `git pull --ff-only` per module. |
| `obmr push` | Push current branch to `fork` if present else `origin`; skip clean modules. |
| `obmr foreach -- <cmd>` | Escape hatch: run a shell command in every module. |
| `obmr pin [--ref REF]` | Post-merge: rewrite canonical SHAs from `origin/<ref>`. Default ref: `origin/HEAD`. |
| `obmr trim [--branch NAME] [--force]` | Delete local branches that are merged into `origin/HEAD` (per module). Skips dirty trees. `--branch` scopes to one branch; `--force` uses `-D`. |

Deferred: `obmr pr` (per-module `gh pr create`).

## Note on lazy branch creation

`obmr checkout <branch> -b` deliberately creates the branch in **every**
module, even those you may never touch. This is predictable and matches
how `git checkout -b` works locally. The cleanup story is `trim`:
after a feature merges (or you abandon it), `obmr trim` removes empty
or merged branches across all modules in one shot. We chose this over a
lazy/conditional `checkout` to keep the per-command behavior simple
and uniform.
