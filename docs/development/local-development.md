# Local Development Guide

## Building

### Golang

This project assumes you have golang 1.17 or higher installed on your machine.

### Container Image

This project uses [ko](https://github.com/google/ko) to build the container image with the
controller manager and deploy it to a Kubernetes cluster. To change the destination container
registry, provide the `IMAGE_REPO` variable to any make target:

```sh
$ make container-build IMAGE_REPO=quay.io/myusername
```

Similarly, use the `TAG` variable to modify the tag for the built image.
By default, this repository builds and pushes the controller image to
`ghcr.io/shipwright-io/triggers/triggers:latest`.

ko by default generates a Software Bill of Materials (SBOM) alongside the container image, which
can be pushed to a container registry.
Not all container registries support SBOM images - to disable this, set the `SBOM` variable to
`none`:

```sh
$ make container-push IMAGE_REPO=quay.io/myusername SBOM=none
```

If you need finer control over how ko builds and pushes images, provide all necessary ko
arguments to the `KO_OPTS` variable:

```sh
$ make container-build IMAGE_REPO=quay.io/myusername/kubebuilder-plus KO_OPTS="--bare --tag=mytag"
```

By default the following arguments are passed to ko:

- `-B`: this strips the md5 hash from the image name
- `-t ${TAG}`: sets the tag for the image. Defaults to the `TAG` argument (`latest`)
- `--sbom=${SBOM}`: configures SBOM behavior. Defaults to the `SBOM` argument (`spdx`)

## Deploying

Use the following instructions to deploy Triggers on a local Kubernetes cluster.

### KinD

To deploy on a KinD cluster, do the following:

1. Install Tekton Pipelines from the latest supported version for Shipwright:

   ```sh
   $ kubectl apply --filename https://storage.googleapis.com/tekton-releases/pipeline/previous/v0.34.1/release.yaml
   ```

2. Install Shipwright Build from the most recent release:
   ```sh
   $ kubectl apply --filename https://github.com/shipwright-io/build/releases/download/v0.9.0/sample-strategies.yaml
   ```

3. Deploy Shipwright Triggers using the `kind.local` image repo:

   ```sh
   $ make deploy IMAGE_REPO=kind.local
   ```

### CodeReady Containers

To deploy on an OpenShift CodeReady Containers cluster, do the following:

1. Install Shipwright and Tekton using the Shipwright Operator in the Community Operators catalog. Make sure that a `ShipwrightBuild` instance is created and reports itself `Ready=True` in its status conditions.

2. Use the following options for `make deploy`:

   ```sh
   $ make deploy IMAGE_REPO=default-route-openshift-image-registry.apps-crc.testing/shipwright-build KO_OPTS="-B -t latest --insecure-registry"
   ```

## Testing

### Unit and Integration Testing

TODO - add instructions on unit and integration tests when these are ready.

### End to End Testing

The end to end test suite is invoked by running `make test-e2e` against a cluster with Triggers deployed.

