<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->
Building and Testing Shipwright Triggers
----------------------------------------

This document describes how to build and test this project, to work on this project you need the following tools installed:

- [GNU/Make][gnuMake]
- [Helm][helmInstall]
- [KO][koBuild]

 The automation necessary to accomplish the objective is concentrated on the [`Makefile`](../Makefile), the organization of each task is defined by the targets. Targets are the entrypoint for the project automation tasks, the most important are described below.

# Building

To compile the project run `make` without any arguments, i.e.:

```bash
make
```

# Testing

Project testing is subdivided as [unit](#unit) and [end-to-end](#e2e) tests, in order to run all tests in the project use the `test` target, i.e.:

```bash
make test
```

Please consider the section below to better understand requirements for end-to-end tests.

## Unit

Unit tests will be exercised by the `test-unit` target. For instance:

```bash
make test-unit
```

During development you might want to narrow down the scope, this can be achieved using the `ARGS` variable, the flags informed are appended on the [`go test`](goTest) command.

For example, narrowing down `TestInventory` function:

```bash
make test-unit ARGS='-run=^"TestInventory$$"'
```

## E2E

End-to-end tests (E2E) will assert the project features against the *real* dependencies, before running the `test-e2e` target you will need to provide the Shipwright Build instance, including its dependencies. Please follow this [documentation section](shpTryIt) first.

With the dependencies in place, run:

```bash
make test-e2e
```

Please consider the [GitHub Actions](#github-actions) section below to have a more practical way of running E2E tests in your own laptop.

## GitHub Actions

Continuous integration (CI) [tests](../.github/workflows/test.yaml) are managed by GitHub Actions, to run these jobs locally please consider [this documentation section][shpSetupContributing] which describes the dependencies and settings required.

After you're all set, run the following target to execute all jobs:

```bash
make act
```

 The CI jobs will exercise [unit](#unit) and [end-to-end](#e2e) tests sequentially, alternatively you may want to run a single suite at the time, that can be achieved using `ARGS` (on `act` target).
 
 Using `ARGS` you can run a single `job` passing flags to [act][nektosAct] command-line. For instance, in order to only run [unit tests](#unit) (`make test-unit`):

```bash
make act ARGS="--job=test-unit"
```

[gnuMake]: https://www.gnu.org/software/make
[goTest]: https://pkg.go.dev/cmd/go/internal/test
[helmInstall]: https://helm.sh/docs/intro/quickstart/#install-helm
[koBuild]: https://github.com/ko-build/ko
[nektosAct]: https://github.com/nektos/act
[shpSetupContributing]: https://github.com/shipwright-io/setup/blob/main/README.md#contributing
[shpTryIt]: https://github.com/shipwright-io/build#try-it
