# Install

## obflow

```sh
go install github.com/btraven00/obflow@latest
```

Or build from source:

```sh
git clone https://github.com/btraven00/obflow
cd obflow
go build -o obflow .
```

## obrun

A minimal companion CLI exposing just `run` and `use`:

```sh
go build -o obrun ./obrun
```

It shares `./.obflow/config.yaml` with `obflow`, so `obrun use` and
`obflow use` are interchangeable.

## Runtime dependencies

Required at runtime, not at build time:

| Tool | Used by | Install |
|---|---|---|
| [`git`](https://git-scm.com/) | everything | system package manager |
| [`uv`](https://docs.astral.sh/uv/) | `obflow run` (non-conda backends) | `curl -LsSf https://astral.sh/uv/install.sh \| sh` |
| [`pixi`](https://pixi.sh) | `obflow run` (conda backend) | `curl -fsSL https://pixi.sh/install.sh \| sh` |
| [`gh`](https://cli.github.com/) | `obflow dev --fork` | system package manager |

`obflow run` will print a copy-pastable install line if `uv` or `pixi` is
missing when you reach for it — you don't need to install everything up
front.
