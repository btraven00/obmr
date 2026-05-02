# obflow

an opinionated omnibenchmark monorepo helper.

## obrun

A no-frills `ob` runner: just `run` and `use`.

Grab the prebuilt binary from [releases](TBD) and drop it on your `PATH`.

```sh
obrun use ~/lab/some-bench/bench.yaml   # remember the active plan
obrun                                   # ob run <plan>
obrun -- --threads 8                    # extra args pass through to ob
```

`obrun use` writes `./.obflow/config.yaml`, the same file `obflow use`
writes, so the two tools share state.

You'll need one of:

- [`pixi`](https://pixi.sh) — for the `conda` software backend.
- [`uv`](https://docs.astral.sh/uv/) — for `apptainer`, `podman`, or `envmodules`.

## obflow

Manages a benchmark and its module repos as a workspace of sibling
clones, driven by the benchmark YAML. See [`docs/`](docs/) (build with
`mkdocs serve`) for install, quickstart, and full command reference.
