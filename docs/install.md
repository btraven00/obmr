# Install

## obmr

```sh
go install github.com/btraven00/obmr@latest
```

Or build from source:

```sh
git clone https://github.com/btraven00/obmr
cd obmr
go build -o obmr .
```

## Runtime dependencies

Required at runtime, not at build time:

| Tool | Used by | Install |
|---|---|---|
| [`git`](https://git-scm.com/) | everything | system package manager |
| [`uv`](https://docs.astral.sh/uv/) | `obmr run` (non-conda backends) | `curl -LsSf https://astral.sh/uv/install.sh \| sh` |
| [`pixi`](https://pixi.sh) | `obmr run` (conda backend) | `curl -fsSL https://pixi.sh/install.sh \| sh` |
| [`gh`](https://cli.github.com/) | `obmr dev --fork` | system package manager |

`obmr run` will print a copy-pastable install line if `uv` or `pixi` is
missing when you reach for it — you don't need to install everything up
front.
