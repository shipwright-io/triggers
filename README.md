Shipwright Triggers
-------------------

Work in progress implementation of [SHIP-0031][SHIP-0031].

## Contributing

To work on this project, consider the following `Makefile` targets:

```bash
# builds the application
make

# run unit-tests
make test-unit

# run end-to-end tests
make test-e2e

# deploy against the controller
make deploy IMAGE_BASE='ghcr.io/...'
```

To work on this project you need the following tools installed:

- GNU/Make
- Helm
- KO
- Kubernetes (`kubectl`)

[SHIP-0031]: https://github.com/shipwright-io/community/blob/main/ships/0031-shipwright-trigger.md
