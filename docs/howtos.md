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

## Iterate on a single module

When debugging one module's logic (e.g. tweaking `pca.py`), `obmr run`
+ Snakemake is too coarse. `obmr enter` drops you into a pixi shell
scoped to that module, with upstream inputs from the latest
successful run preloaded as env vars and a `runit` helper on PATH that
echoes-then-runs the module's entrypoint with all required flags
prefilled:

```sh
obmr enter pca-scanpy
# inside the shell — plan-declared parameters are used as defaults:
runit
# + python pca.py --output_dir /tmp/... --name datasets \
#     --normalized.h5 /.../datasets_normalized.h5 \
#     --selected.genes /.../datasets_selected.txt.gz \
#     --pca_type scanpy_arpack --n_components 50 --random_seed 42
# (then it runs)

# override any plan default by passing it after `runit`:
runit --pca_type scanpy_randomized --random_seed 7
```

`runit` reads the **first** entry of the module's plan-declared
`parameters:` block; for cartesian expansions like
`selection_type: [a, b]`, it picks the first value. User-supplied flags
override (argparse takes the last occurrence).

You can also use the env vars directly:

```sh
echo $NORMALIZED_H5      # /.../datasets_normalized.h5
echo $SELECTED_GENES     # /.../datasets_selected.txt.gz
echo $OBMR_OUTPUT_DIR    # /tmp/obmr-enter-pca-scanpy-<rand>
python pca.py --output_dir $OBMR_OUTPUT_DIR --name $OBMR_NAME \
  --normalized.h5 $NORMALIZED_H5 --selected.genes $SELECTED_GENES \
  --pca_type scanpy_arpack --n_components 50 --random_seed 0
```

Env-var names are derived from the plan's input ids (e.g.
`normalized.h5` → `NORMALIZED_H5`). The upstream chain is anchored on
the deepest matching candidate so all inputs come from one coherent
DAG path.

If you don't want a sub-shell, source the exports into your current
shell:

```sh
eval "$(obmr enter pca-scanpy --print)"
runit --pca_type scanpy_arpack --n_components 50 --random_seed 0
```

To debug interactively, add `ipython` / `ipdb` to the module's
`pixi.toml` and pass `--env <name>` if you put them under a feature.

### Requirements

- The benchmark must be in dev mode (run `obmr dev` first — `obmr enter`
  loads the local YAML).
- At least one successful prior `obmr run` so the upstream stages have
  produced files under `out/`. If a producer has never run, `obmr enter`
  errors with the missing input id and a pointer to `obmr run`.
- The module needs a `pixi.toml` (its environment) and an
  `omnibenchmark.yaml` declaring its entrypoints (so `runit` can
  resolve the correct script).
