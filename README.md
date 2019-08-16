# Teiid Operator
Teiid Operator for OpenShift/Kubernetes

[![Go Report](https://goreportcard.com/badge/github.com/teiid/teiid-operator)](https://goreportcard.com/report/github.com/teiid/teiid-operator)
[![Build Status](https://travis-ci.org/teiid/teiid-operator.svg?branch=master)](https://travis-ci.org/teiid/teiid-operator)

## Requirements

- go v1.11+
- dep v0.5.x
- operator-sdk v0.7.0

## Build

```bash
make
```

## Upload to a container registry

e.g.

```bash
docker push quay.io/teiid/teiid-operator:<version>
```

## Deploy to OpenShift 4 using OLM

To install this operator on OpenShift 4 for end-to-end testing, make sure you have access to a quay.io account to create an application repository. Follow the [authentication](https://github.com/operator-framework/operator-courier/#authentication) instructions for Operator Courier to obtain an account token. This token is in the form of "basic XXXXXXXXX" and both words are required for the command.

Push the operator bundle to your quay application repository as follows:

```bash
operator-courier push deploy/courier/0.0.1 teiid teiid 0.0.1 "basic XXXXXXXXX"
```

If pushing to another quay repository, replace _teiid_ with your username or other namespace. Also note that the push command does not overwrite an existing repository, and it needs to be deleted before a new version can be built and uploaded. Once the bundle has been uploaded, create an [Operator Source](https://github.com/operator-framework/community-operators/blob/master/docs/testing-operators.md#linking-the-quay-application-repository-to-your-openshift-40-cluster) to load your operator bundle in OpenShift.

```bash
oc create -f deploy/courier/teiid-operatorsource.yaml
```

Remember to replace _registryNamespace_ with your quay namespace. The name, display name and publisher of the operator are the only other attributes that may be modified.

It will take a few minutes for the operator to become visible under the _OperatorHub_ section of the OpenShift console _Catalog_. It can be easily found by filtering the provider type to _Custom_.

## VirtualDatabase Deployment

Use the OLM console to subscribe to the `Teiid Operators` Operator Catalog Source within your namespace. Once subscribed, use the console to `Create VirtualDatabase` or create one manually as seen below.

```shell
$ oc tag --source=docker docker.io/fabric8/s2i-java:latest-java11 openshift/s2i-java:latest-java11 -n openshift
$ oc new-app -e POSTGRESQL_USER=user -e POSTGRESQL_PASSWORD=mypassword -e POSTGRESQL_DATABASE=sampledb postgresql:9.5
$ oc apply -f deploy/crs/vdb_v1alpha1_virtualdatabase_cr.yaml
virtualdatabase.teiid.io/rdbms-springboot created
```

### Clean up a VirtualDatabase deployment

```bash
oc delete vdb rdbms-springboot
```

## Development

Change log level at runtime w/ the `DEBUG` environment variable. e.g. -

```bash
make dep
make clean
DEBUG="true" operator-sdk up local --namespace=<namespace>
```

Before submitting PR, please be sure to generate, vet, format, and test your code. This all can be done with one command.

```bash
make test
make
make deploy
```

## OLM Notes
https://github.com/operator-framework/community-operators/blob/master/docs/contributing.md