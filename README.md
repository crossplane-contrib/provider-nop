# provider-nop

`provider-nop` is a Crossplane infrastructure provider that does nothing. It
provides one managed resource - a `NopResource` that does not orchestrate any
external system. Each `NopResource` can be configured to emit arbitrary status
conditions after a specified period of time.

The main value of a `NopResource` is that it can be used to create a Crossplane
`Composition` that can satisfy any kind of composite resource by doing nothing.
This can be useful for systems that automatically create a real composite
resource (one that composes real cloud infrastructure) when running in
production, but that wish to avoid creating real infrastructure when running in
development. It can also be useful for developing and testing Crossplane's
support for Composition itself.

The below `Composition` satisfies the `SQLInstance` composite resource kind by
by composing a `NopResource`. When an `SQLInstance` is created it will become
ready and write fake data to a connection secret.

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: nop.sqlinstances.example.org
spec:
  writeConnectionSecretsToNamespace: crossplane-system
  compositeTypeRef:
    apiVersion: example.org/v1alpha1
    kind: SQLInstance
  resources:
    - name: nop
      base:
        apiVersion: nop.crossplane.io/v1alpha1
        kind: NopResource
        spec:
          forProvider:
            # This NopResource will set its 'Ready' status condition to 'True'
            # after 10 seconds.
            conditionAfter:
            - time: 10s
              conditionType: Ready
              conditionStatus: "True"
          # Like all managed resources the NopResource allows you to configure a
          # provider config. It ignores the configured value.
          providerConfigRef:
            name: default
          # Simulating connection details (see connectionDetails below) works
          # only when the NopResource writes a connection secret. The supplied
          # connection secret will be written, but empty.
          writeConnectionSecretToRef:
            namespace: crossplane-system
            name: nop-example-resource
      patches:
        # You can patch a NopResource, but it doesn't have any spec fields of
        # interest. In this case we copy the SQLInstance's spec fields to
        # annotations of the NopResource.
        - fromFieldPath: spec.parameters.engineVersion
          toFieldPath: metadata.annotations[nop.crossplane.io/engineVersion]
        - fromFieldPath: spec.parameters.storageGB
          toFieldPath: metadata.annotations[nop.crossplane.io/storageGB]
          transforms:
            - type: string
              string:
                fmt: "%d"
      # You can simulate connection details being returned by the NopResource by
      # simply providing fake, static details.
      connectionDetails:
        - name: username
          value: fakeuser
        - name: password
          value: verysecurepassword
        - name: endpoint
          value: 127.0.0.1
```
