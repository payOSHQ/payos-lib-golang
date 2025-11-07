# Contributing

## Setting up the environment

```bash
./scripts/bootstrap.sh
```

This will install all the required dependencies.

## Adding and running examples

You can run, modify and add new examples in `examples/` directory.

```bash
cd examples/
go run <example>.go
```

## Linting and formatting

To ensure code quality and consistency, we use `gofmt` for formatting.

### Linting

To run the linter, use the following command:

```bash
./scripts/lint.sh
```

### Formatting

To format your code, use the following command:

```bash
./scripts/format.sh
```

## Publishing and release

TO publish a new version of the package, create new tags and push to remote repository:

```bash
git tag vX.Y.Z
git push origin vX.Y.Z
./scripts/trigger-pkg.sh vX.Y.Z
```
