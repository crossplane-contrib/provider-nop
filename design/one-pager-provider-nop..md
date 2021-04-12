# Crossplane provider-nop

* Owner: Rahul Grover (@rahulgrover99)
* Reviewers: Crossplane Maintainers
* Status: Speculative

## Introduction

There is a lot of machinery around the composition engine that needs to be tested which raises demand for a provider on which could rely to behave the way we want it to. It can be used to mock the behavior of providers by creating fake objects whenever needed so that we could test the response of composition to the resource becoming ready or not. 

This provider does exactly what you tell it to. It can be used for testing functionality in core Crossplane without needing to account for the requirements of external APIs. It supports a single managed resource that contains a spec that allows for specifying how and when the status of the resource is changed. It does not orchestrate any external system. 

This provider will be installed while running Crossplane end-to-end tests to satisfy any kind of composite infrastructure resource by doing nothing to simulate different scenarios. 

## Design

The provider will have a single managed resource `NopResource` that will reflect whatever is in the spec into the status. It will let you define to:
- become ready after this period
- become unhealthy after this period

The config might look something like:

```yaml
spec:
    condtionAfter:
        time: 5s
        condition: ready
```

The idea here is just to wait for the time provided in the spec before allowing the resource to be ready/unhealthy. This could be achieved by adding the logic to the controllers. 

