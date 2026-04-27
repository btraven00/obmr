# obmr

omnibenchmark monorepo helper.

See [DESIGN.md](DESIGN.md) for the design and milestones.

## Build

```sh
go build -o obmr .
```

## Usage (M1, in progress)

```sh
obmr list path/to/benchmark.yaml
obmr init path/to/benchmark.yaml --parent ../bench-modules
```
