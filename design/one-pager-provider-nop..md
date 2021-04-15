# Crossplane provider-nop

* Owner: Rahul Grover (@rahulgrover99)
* Reviewers: Crossplane Maintainers
* Status: Speculative

## Introduction

There is a lot of machinery around the composition engine that needs to be tested which raises demand for a provider on which could rely to behave the way we want it to. It can be used to mock the behavior of providers by creating fake objects whenever needed so that we could test the response of composition to the resource becoming ready or not. 

Following is a very simple possible test scenario we might use provider-nop and its objects for:
- test creates an XRD and a composition of 3 nop objects
  - each of those nop objects is told to become ready after 3 seconds
- test creates an XR instance of this XRD
- test waits and verifies that 3 nops objects are created, they all make it to the ready status, and the XR itself then makes it to the ready status because all 3 of its composed resources are now ready.

This provider does exactly what it is told to. It can be used for testing functionality in core Crossplane without needing to account for the requirements of external APIs. It supports a single managed resource that contains a spec that allows for specifying how and when the status of the resource is changed. It does not orchestrate any external system. 

This provider will be installed while running Crossplane end-to-end tests to satisfy any kind of composite infrastructure resource to simulate different scenarios. 

## Design

The provider will have a single managed resource `NopResource` that will reflect whatever is in the spec into the status. It will allow to define the resource to:
- be ready after this period
- be unhealthy after this period

The idea is to have an array of fields which will let you declare condition type and status of the resource at each time interval. 

This will be implemented by making use of three fields:
- `conditionType` and `conditionStatus` for declaring condtion of resource
- `timeAfter` for declaring the time elapsed after the creation of resource at which we need to set the specified condition.

The resource structure might look something like: 
```go
type ResourceConditionAfter struct {
  Time            string `json:"time"`
  ConditionType   string `json:"conditionType"`
  ConditionStatus string `json:"conditionStatus"`
}

type NopResourceParameters struct {
	ConditionAfter []ResourceConditionAfter `json:"conditionAfter"`
}
```

### `NopResource` Controller

The controller for this nop type will not call any external APIs during its reconcile loop and just perform the behaviour as specified by the user in the spec. 

#### Setup
Since default is too long for the testing use case, the sync period should be decreased to a small value like `100ms` using `WithPollInterval`in the Setup function. This will allow controller to set various conditions.

#### Connect
Currently, there is no requirement for having `ProviderConfig` since we don't have any external API client that we might want to connect. 
We would just create an external client with no service parameter passed. 

#### Observe
This is where the main logic will reside that would compare the time elapsed with the time interval passed in the spec. As the time elapsed becomes equal to the time passed it will change the condition of resource accordingly. 

#### Update
Currently there is no plan to update the NopResource so this function will just set condition to crossplane-runtime `Available`. 

#### Delete
This will set the condition of NopResource to crossplane-runtime `Deleting`. 


The config might look something like:

```yaml
apiVersion: sample.template.crossplane.io/v1alpha1
kind: NopResource
metadata:
  name: example
spec:
  forProvider:
    conditionAfter:
      - conditionType: "Synced"
        conditionStatus: "True"
        timeAfter: "5s"
      - conditionType: "Ready"
        conditionStatus: "False"
        timeAfter: "10s"
```
`ObservableField` is an arbitrary field in `NopResourceObservation` type can be useful in crossplane testing scenarios. For example, while testing bidirectional patching back to the composite resource from one resource to another which can be supported by this field. 

The idea here is just to wait for the time provided in the spec before allowing the resource to be ready/unhealthy. This could be achieved by adding the logic to the controllers. 

## Future Plans
- We might want to include `ProviderConfig` at some point for those resources which are not managed. 
- Array type in nop resource observation type like `ObservableArrays []string` can also be useful in patching.
- A `patchReceiverField` can added as an optional field in `NopResource` parameters that just gets patched value from `coolField` of XRD 
