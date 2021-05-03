# provider-nop
`provider-nop` does exactly what it is told to. It can be used for testing
functionality in core Crossplane without needing to account for the requirements
of external APIs. It supports a single managed resource that contains a spec
that allows for specifying how and when the status of the resource is changed.
It does not orchestrate any external system. 

## Developing

Run against a Kubernetes cluster:

```console
make run
```

Build, push, and install:

```console
make all
```

Build image:

```console
make image
```

Push image:

```console
make push
```

Build binary:

```console
make build
```