# How-to

Task-oriented recipes. Each assumes you've run `obmr use <plan>` and
`obmr init`.

## Test a PR of omnibenchmark

Configure once, then `obmr run` uses the PR build:

```sh
obmr config omnibenchmark.pr 123
obmr run
```

When done, clear it:

```sh
obmr config --unset omnibenchmark.pr   # back to latest pypi
```

Resolution priority is `pr > branch > version > pypi-latest`. Setting
any of `branch`/`version` follows the same pattern.

## Test a PR on a benchmark module

`obmr` doesn't wrap PR fetching; do it per-module with `gh`:

```sh
cd ../some-bench-modules/<module>
gh pr checkout 456
cd -
obmr dev          # regenerate bench.local.yaml with the new branch
obmr run          # runs against the checked-out PR
```

`obmr status` will show the module on the PR's branch.

## Edit a YAML stage and push it back to canonical

Sketch in `bench.local.yaml` (uncomment a stage, tweak parameters,
whatever). When ready:

```sh
obmr status                # confirms divergence + hint
obmr plan promote          # writes canonical from local (urls/commits restored)
git -C $(dirname <plan>) diff   # review
```

`plan promote` refuses to write canonical if any local module has a
url that isn't a known remote (e.g. you added a new module without a
github URL — fix that first).

## Update canonical after PRs merge upstream

```sh
obmr pull            # fast-forward each module on its current branch
obmr plan pin        # rewrite canonical SHAs from origin/HEAD
obmr trim            # clean up merged local branches
```

`plan pin` is upstream-only (`git fetch origin && rev-parse origin/<ref>`),
so the SHAs it writes are always fetchable by anyone.

## Use your own forks instead of pushing to upstream

```sh
obmr dev --fork      # creates a `fork` remote per module via `gh repo fork`
obmr checkout feat-x -b
# ... edit, commit ...
obmr push            # auto-targets `fork` (falls back to `origin`)
```

The canonical YAML never references your forks; they're per-developer
plumbing.

## Branch across all modules at once

```sh
obmr checkout feat-x -b   # creates feat-x in every module
obmr status               # confirm
```

`obmr` deliberately creates the branch in every module, even ones you
may not touch. After the feature ships, `obmr trim` removes empty/merged
branches in one shot.

## Run an arbitrary command across all modules

```sh
obmr foreach -- git log -1 --oneline
obmr foreach -- pre-commit run --all-files
```

Per-repo prefix on each line; runs in parallel.

## Bring in canonical changes after someone else updates the spec

```sh
git pull              # in the canonical bench repo (you own it)
obmr init             # clones any new modules; existing clones untouched
obmr dev              # regenerate bench.local.yaml
```

## Fix a malformed plan YAML

```sh
obmr plan fmt         # in dev mode, formats local; otherwise canonical
obmr plan fmt --local # explicit
```

Requires the file to parse first. Parse errors are shown cargo-style with
context lines and a caret.

## See where I am

```sh
obmr status
```

Shows the active plan, dev/prod mode, local-edit divergence summary
(with hint to `obmr plan promote`), and per-module branch + dirty flag.
