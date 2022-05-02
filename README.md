# provider-nop

`provider-nop` does exactly what it is told to. It can be used for testing
functionality in core Crossplane without needing to account for the requirements
of external APIs. It supports a single managed resource that contains a spec
that allows for specifying how and when the status of the resource is changed.
It does not orchestrate any external system. 

## Getting Started
Install the provider by using the following command:

```
kubectl crossplane install provider crossplane/provider-nop:v0.1.1
```

See [examples](examples/) for creating a NopResource.
