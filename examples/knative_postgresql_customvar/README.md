# Binding an Imported Quarkus app deployed as Knative service with an In-cluster Operator Managed PostgreSQL Database

## Introduction

This scenario illustrates binding a Quarkus application deployed as Knative service with an in-cluster operated managed PostgreSQL Database.

## Actions to Perform by Users in 2 Roles

In this example there are 2 roles:

* Cluster Admin - Installs the operators and Serverless plugin to the OpenShift cluster
* Application Developer - Imports a Node.js application, creates a DB instance, creates a request to bind the application and DB (to connect the DB and the application).

### Cluster Admin

The cluster admin needs to install operators, knative serving and a builder image into the cluster:

* Service Binding Operator
* Backing Service Operator
* Serverless plugin
  * Service Mesh Opearator
  * Serverless Operator
  * Serverless UI
* Quarkus Native S2i Builder Image

A Backing Service Operator that is "bind-able," in other
words a Backing Service Operator that exposes binding information in secrets, config maps, status, and/or spec
attributes. The Backing Service Operator may represent a database or other services required by
applications. We'll use [postgresql-operator](https://github.com/operator-backing-service-samples/postgresql-operator) to
demonstrate a sample use case.

We can install all by running the following make target (note that the command will take up to ten minutes to run):

```shell
make install-all
```

or we can install one by one via the following six individual make targets.

#### Install the Service Binding Operator

```shell
make install-service-binding-operator
```

This makes the `ServiceBindingRequest` custom resource available, that the application developer will use later.

#### Install the backing DB (PostgreSQL) operator

```shell
make install-backing-db-operator
```

This makes the `Database` custom resource available, that the application developer will use later.

#### Install the Serverless plugin to OpenShift

The installation process for the Serverless plugin to OpenShift is documented [here](https://docs.openshift.com/container-platform/4.1/serverless/installing-openshift-serverless.html).

However there're several make targets for each step to make it easy for us.

##### Install the Service Mesh Operator

```shell
make install-service-mesh-operator
```

This makes the `ServiceMeshControlPlane` and `ServiceMeshMemberRoll` custom resourced available, that the application developer will use later.

##### Install the Serverless Operator

```shell
make install-serverless-operator
```

This makes the `KnativeServing` custom resource available, that the application developer will use later to deploy the application.

##### Install Serverless UI

```shell
make install-knative-serving
```

This enables `Serverless` view in the UI. Note that installing the Serverless features will require around seven minutes for the command to run.

#### Install the `ubi-quarkus-native-s2i` builder image

```shell
make install-quarkus-native-s2i-builder
```

This makes the `Ubi Quarkus Native S2i` builder image available for the application to be built when imported by the application developer.

When everything is installed, let's refresh the OpenShift Console Page to make the Serverless features visible.

### Application Developer

#### Create a namespace called `service-binding-demo`

The application and the DB needs a namespace to live in so let's create one for them:

```shell
make create-project
```

This creates a project/namespace called `service-binding-demo`.

#### Import an application

In this example we will import an arbitrary [Quarkus application](https://github.com/sbose78/using-spring-data-jpa-quarkus).

In the OpenShift Console switch to the Developer perspective. (We need to make sure we've selected the `service-binding-demo` project). Navigate to the `+Add` page from the menu and then click on the `[From Git]` button. Fill in the form with the following:

* `Git Repo URL` = `https://github.com/sbose78/using-spring-data-jpa-quarkus`
* `Builder Image` = `Ubi Quarkus Native S2i`
* `Application`->`Create Application`
* `Application Name` = `knative-app`
* `Name` = `knative-app`
* `Serverless`->`Enable scaling to zero when idle` = checked
* `Advanced Options`->`Create a route to the application` = checked

and click on the `[Create]` button.

Notice, that during the import no DB config was mentioned or requestd.

It take several minutes to build the application using the Quarkus native s2i builder image, we can check the running build progress in the Administrator's perspective under `Builds`->`Builds` view until the build status is `Complete`.

After the application is built we can check the `Services` under `Serverless` view to see the deployed application. The application should fail at this point with `Reason` to be the "connection refused" error. That indicates that the application is not connected to the DB.

#### Set labels on the application

Now we need to set arbitrary labels on the application's `Service` in order for the Service Binding Operator to be able to find the application.

The labels are:

* `connects-to=postgres` - indicates that the application needs to connect to a PostgreSQL DB
* `environment=demo` - indicates the demo environment - it narrows the search

```shell
oc label services.serving.knative.dev knative-app connects-to=postgres environment=demo --overwrite
```

Alternatively, we can perform the same task with this make command:

```shell
make set-labels-on-knative-app
```

#### Create a DB instance for the application

Now we utilize the DB operator that the cluster admin has installed. To create a DB instance just create a `Database` custom resource in the `service-binding-demo` namespace called `db-demo`:

```shell
cat <<EOS |kubectl apply -f -
---
apiVersion: postgresql.baiju.dev/v1alpha1
kind: Database
metadata:
  name: db-demo
  namespace: service-binding-demo
spec:
  image: docker.io/postgres
  imageName: postgres
  dbName: db-demo
EOS
```

Alternatively, we can perform the same task with this make command:

```shell
make create-backing-db-instance
```

#### Express an intent to bind the DB and the application together

Now, the only thing that remains is to connect the DB and the application. We let the Service Binding Operator to make the connection for us.

Create the following `ServiceBindingRequest`:

```shell
cat <<EOS |kubectl apply -f -
---
apiVersion: apps.openshift.io/v1alpha1
kind: ServiceBindingRequest
metadata:
  name: binding-request
  namespace: service-binding-demo
spec:
  applicationSelector:
    group: serving.knative.dev
    version: v1beta1
    resource: services
    matchLabels:
      connects-to: postgres
      environment: demo
  backingServiceSelector:
    group: postgresql.baiju.dev
    version: v1alpha1
    kind: Database
    resourceRef: db-demo
  customEnvVar:
    - name: JDBC_URL
      value: 'jdbc:postgresql://{{ .status.dbConnectionIP }}:{{ .status.dbConnectionPort }}/{{ .status.dbName }}'
    - name: DB_USER
      value: '{{ index .status.dbConfigMap "db.username" }}'
    - name: DB_PASSWORD
      value: '{{ index .status.dbConfigMap "db.password" }}'
EOS
```

Alternatively, we can perform the same task with this make command:

```shell
make create-service-binding-request
```

There are 2 parts in the request:

* `applicationSelector` - used to search for the application based on the labels that we set earlier and the `group`, `version` and `resource` of the application to be a knative `Service`.
* `backingServiceSelector` - used to find the backing service - our operator-backed DB instance called `db-demo`.

That causes the application to be re-deployed.

Once the new version is up, go to the application's route to check the UI. Now, it works!

When the `ServiceBindingRequest` was created the Service Binding Operator's controller injected the DB connection information into the
application as environment variables via an intermediate `Secret` called `binding-request`:

```yaml
spec:
  template:
    spec:
      containers:
        - envFrom:
          - secretRef:
              name: binding-request
```

#### ServiceBindingRequestStatus

`ServiceBindingRequestStatus` depicts the status of the Service Binding operator. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#spec-and-status

| Field | Description |
|-------|-------------|
| BindingStatus | The binding status of Service Binding Request |
| Secret | The name of the intermediate secret |

That's it, folks!
