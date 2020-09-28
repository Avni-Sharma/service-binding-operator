# Best Practices for Making a Kubernetes Resource Bindable

## Introduction

The goals of the Service Binding Operator is to make it easier for
applications developers to bind applications with needed backing
services, without having to perform manual configuration of `secrets`,
`configmaps`, etc. and to assist operator providers in promoting and
expanding the adoption of their operators.

When a `ServiceBinding` is created the Service Binding Operator
collects binding information and shares it with application. The
Service Binding Operator's controller injects the binding information
into the application's `DeploymentConfig`, `Deployment` or `Replicaset`
as environment variables via an intermediate Secret.
The binding also works with Knative `Services` as it works with any API which has the podspec defined in the its jsonpath as
"spec.template.spec.containers".

This document provides "best practices" guidelines for the development of
Operators that manage backing services to be bound together with applications
by the Service Binding Operator.

## Making a Kubernetes Resource Bindable

In order to make backing service bindable, the Kubernetes resource representing the same needs to be annotated as a means to express what is "interesting" for applications.

Example, in an `Ingress` or `Route` resource representing a backing service, the `host` and `port` among other things would be "interesting" for applications for connecting to the backing service.

To do so, the Kubernetes resource may be annotated to meaningfully convey what is "interesting" for binding.

``` yaml
kind: Route
apiVersion: route.openshift.io/v1
metadata:
  name: example
  namespace: service-binding-demo
  annotations:
    openshift.io/host.generated: 'true'
    service.binding/host: 'path={.spec.host}' #annotate here.
spec:
  host: example-sbo.apps.ci-ln-smyggvb-d5d6b.origin-ci-int-aws.dev.rhcloud.com
  path: /
  to:
    kind: Service
    name: example
    weight: 100
  port:
    targetPort: 80
  wildcardPolicy: None
```

## Making a Helm-Chart managed Backing Service Bindable

Resources created from a Helm Chart are treated the same way as any other Kubernetes resource. The Kubernetes resource may be annotated the way in the chart templates to denote what is "interesting" for binding.


## Making an Operator Managed Backing Service Bindable

In order to make a service bindable, the operator provider needs to express
the information needed by applications to bind with the services provided by
the operator. In other words, the operator provider must express the
information that is “interesting” to applications.

There are three methods for making Operator Managed Backing Service Bindable:

* [Non-OLM Operator and Resource Annotations](#Non-OLM-Operator-and-Resource-Annotations)
* [Operator Providing Metadata in OLM](#operator-providing-metadata-in-olm)
* [Operator Not Providing Metadata](#operator-not-providing-metadata)

### Non-OLM Operator and Resource Annotations

This feature enables operator providers who do not use OLM (Operator Lifecycle
Manager) to provide metadata outside of an OLM descriptor. In this method,
the binding information is provided as annotations in the CRD of the operator
that manages the backing service or the CR. The Service Binding Operator extracts the
annotations to bind the application together with the backing service.

An intermediate Secret is created with the data exposed by the services via annotations. To handle the majority of existing resources and CRDs, `Secret` generation needs to support the following behaviours:

1.  Extract a string from a resource
1.  Extract an entire `ConfigMap`/`Secret` refrenced from a resource
1.  Extract a specific entry in a `ConfigMap`/`Secret` referenced from a resource
1.  Extract entries from a collection of objects, mapping keys and values from entries in a `ConfigMap`/`Secret` referenced from a resource
1.  Map each value to a specific key

While the syntax of the generation strategies are specific to the system they are annotating, they are based on a common data model.

| Model | Description
| ----- | -----------
| `path` | A template represention of the path to an element in a Kubernetes resource.  The value of `path` is specified as [JSONPath](https://kubernetes.io/docs/reference/kubectl/jsonpath/).  Required.
| `objectType` | Specifies the type of the object selected by the `path`.  One of `ConfigMap`, `Secret`, or `string` (default).
| `elementType` | Specifies the type of object in an array selected by the `path`.  One of `sliceOfMaps`, `sliceOfStrings`, `string` (default).
| `sourceKey` | Specifies a particular key to select if a `ConfigMap` or `Secret` is selected by the `path`.  Specifies a value to use for the key for an entry in a binding `Secret` when `elementType` is `sliceOfMaps`.
| `sourceValue` | Specifies a particular value to use for the value for an entry in a binding `Secret` when `elementType` is `sliceOfMaps`

Let's take a look at the annotations in a more descriptive manner.
The following examples refer to this resource definition.

```yaml
apiVersion: apps.kube.io/v1beta1
kind: Database
metadata:
  name: my-cluster
spec:
  ...

status:
  bootstrap:
  - type: plain
    url: myhost2.example.com
    name: hostGroup1
  - type: tls
    url: myhost1.example.com:9092,myhost2.example.com:9092
    name: hostGroup2
  data:
    dbConfiguration: database-config     # ConfigMap
    dbCredentials: database-cred-Secret  # Secret
    url: db.stage.redhat.com
```

1.  Mount an entire `Secret` as the binding `Secret`
    ```plain
    “service.binding":
      ”path={.status.data.dbCredentials},objectType=Secret”
    ```
1.  Mount an entire `ConfigMap` as the binding `Secret`
    ```plain
    service.binding”:
      "path={.status.data.dbConfiguration},objectType=ConfigMap”
    ```
1.  Mount an entry from a `ConfigMap` into the binding `Secret`
    ```plain
    “service.binding/certificate”:
      "path={.status.data.dbConfiguration},objectType=ConfigMap,sourceKey=certificate"
    ```
1.  Mount an entry from a `ConfigMap` into the binding `Secret` with a different key
    ```plain
    “service.binding/timeout”:
      “path={.status.data.dbConfiguration},objectType=ConfigMap,sourceKey=db_timeout”
    ```
1.  Mount a resource definition value into the binding `Secret`
    ```plain
    “service.binding/uri”:
      "path={.status.data.url}"
    ```
1.  Mount a resource definition value into the binding `Secret` with a different key
    ```plain
    “service.binding/uri":
      "path={.status.data.connectionURL}”
    ```
1.  Mount the entries of a collection into the binding `Secret` selecting the key and value from each entry
    ```plain
    “service.binding/endpoints”:
      "path={.status.bootstrap},elementType=sliceOfMaps,sourceKey=type,sourceValue=url"
    ```

### Operator Providing Metadata in OLM

This feature enables operator providers to specify binding information an
operator's OLM (Operator Lifecycle Manager) descriptor. The Service Binding
Operator extracts to bind the application together with the backing service.
OLM Operators are configured by setting the `specDescriptor` and `statusDescriptor` entries in the [ClusterServiceVersion](https://docs.openshift.com/container-platform/4.4/operators/operator_sdk/osdk-generating-csvs.html) with mapping descriptors.

The following examples refer to this resource definition.

```yaml
apiVersion: apps.kube.io/v1beta1
kind: Database
metadata:
  name: my-cluster
spec:
  ...

status:
  bootstrap:
  - type: plain
    url: myhost2.example.com
    name: hostGroup1
  - type: tls
    url: myhost1.example.com:9092,myhost2.example.com:9092
    name: hostGroup2
  data:
    dbConfiguration: database-config     # ConfigMap
    dbCredentials: database-cred-Secret  # Secret
    url: db.stage.redhat.com
```

1.  Mount an entire `Secret` as the binding `Secret`

    ```yaml
    - path: data.dbCredentials
      x-descriptors:
      - urn:alm:descriptor:io.kubernetes:Secret
      - service.binding
    ```

1.  Mount an entire `ConfigMap` as the binding `Secret`

	```yaml
    - path: data.dbConfiguration
      x-descriptors:
      - urn:alm:descriptor:io.kubernetes:ConfigMap
      - service.binding
    ```

1.  Mount an entry from a `ConfigMap` into the binding `Secret`

    ```yaml
    - path: data.dbConfiguration
      x-descriptors:
      - urn:alm:descriptor:io.kubernetes:ConfigMap
      - service.binding:certificate:sourceKey=certificate
    ```

1.  Mount an entry from a `ConfigMap` into the binding `Secret` with a different key

    ```yaml
    - path: data.dbConfiguration
      x-descriptors:
      - urn:alm:descriptor:io.kubernetes:ConfigMap
      - service.binding:timeout:sourceKey=db_timeout
    ```

1.  Mount a resource definition value into the binding `Secret`

    ```yaml
    - path: data.uri
      x-descriptors:
      - service.binding:uri
    ```

1.  Mount a resource definition value into the binding `Secret` with a different key

    ```yaml
    - path: data.connectionURL
      x-descriptors:
      - service.binding:uri
    ```

1.  Mount the entries of a collection into the binding `Secret` selecting the key and value from each entry

    ```yaml
    - path: bootstrap
      x-descriptors:
      - service.binding:endpoints:elementType=sliceOfMaps:sourceKey=type:sourceValue=url
    ```

### Operator Not Providing Metadata

This feature enables operators that manage backing services but which don't
have any metadata in their CSV to use the Service Binding Operator to bind
together the service and applications. The Service Binding Operator binds all
sub-resources defined in the backing service CR by populating the binding
secret with information from Routes, Services, ConfigMaps, and Secrets owned
by the backing service CR.

[This is how resource and sub-resource relationships are set in
Kubernetes.](https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/#owners-and-dependents)

The binding is initiated by the introduction of this API option in the backing service CR:
``` yaml
detectBindingResources : true
```
When this API option is set to true, the Service Binding Operator
automatically detects Routes, Services, ConfigMaps, and Secrets owned by
the backing service CR.

## Reference Operators

Reference backing service operators are available [here.](https://github.com/operator-backing-service-samples)

A set of examples, each of which illustrates a usage scenario for the
Service Binding Operator, is being developed in parallel with the Operator.
Each example makes use of one of the reference operators and includes
instructions for deploying the reference operators to a cluster, either
through the command line or client web console UI. The examples are
available [here.](https://github.com/redhat-developer/service-binding-operator/blob/master/README.md#example-scenarios)
