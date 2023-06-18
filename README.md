# provider-nop

`provider-nop` is a Crossplane provider that does nothing. It provides one
managed resource - a `NopResource` that does not orchestrate any external
system. Each `NopResource` can be configured to emit arbitrary status conditions
after a specified period of time. A `NopResource` can also emit arbitrary
connection details.

The main value of a `NopResource` is that it can be used to create a Crossplane
`Composition` that can satisfy any kind of composite resource by doing nothing.
This can be useful for systems that automatically create a real composite
resource (one that composes real cloud infrastructure) when running in
production, but that wish to avoid creating real infrastructure when running in
development. It can also be useful for developing and testing Crossplane itself.

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
            # The NopResource spec.forProvider.fields is an arbitrary,
            # schemaless object. Use it to patch to and from.
            # status.atProvider.fields works the same.
            fields:
              integerField: 42
              stringField: "cool"
              objectField:
                stringField: "cool"
              arrayField:
              - stringField: "cool"
            # This NopResource will set its 'Ready' status condition to 'True'
            # after 10 seconds.
            conditionAfter:
            - time: 10s
              conditionType: Ready
              conditionStatus: "True"
            # The NopResource will emit whatever connection details it is told
            # to have. These are all plaintext - for testing only.
            connectionDetails:
            - name: username
              value: fakeuser
            - name: password
              value: verysecurepassword
            - name: endpoint
              value: 127.0.0.1
          # Like all managed resources the NopResource allows you to configure a
          # provider config. It ignores the configured value.
          providerConfigRef:
            name: default
          # Simulating connection details (see connectionDetails above) works
          # only when the NopResource writes a connection secret.
          writeConnectionSecretToRef:
            namespace: crossplane-system
            name: nop-example-resource
      patches:
        # You can use the schemaless 'fields' objects to patch to and from.
        - type: FromCompositeFieldPath
          fromFieldPath: spec.parameters.storageGB
          toFieldPath: spec.forProvider.fields.storageGB
          transforms:
            - type: string
              string:
                fmt: "%d"
        - type: ToCompositeFieldPath
          fromFieldPath: status.atProvider.fields.health
          toFieldPath: status.health
```
