# cloudfirewall

Starter repository scaffold for the cloudfirewall.io core IP written in Go.

## Included

- policy authoring model and normalized IR
- normalization, resolution, validation, compilation, simulation, and artifact package skeletons
- engine CLI stub
- agent stub
- JSON test fixtures
- starter tests

## Quick start

```bash
make test
make cli
./bin/engine-cli validate --policy testdata/policies/public-web-server.json
```
