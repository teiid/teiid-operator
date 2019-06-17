# Teiid Operator
Teiid Operator for OpenShift/Kubernetes

[![Go Report](https://goreportcard.com/badge/github.com/teiid/teiid-operator)](https://goreportcard.com/report/github.com/teiid/teiid-operator)

## Requirements

- go v1.11+
- dep v0.5.x
- operator-sdk v0.7.0

## Install

```shell
oc tag --source=docker docker.io/fabric8/s2i-java:latest-java11 openshift/s2i-java:latest-java11 -n openshift
oc new-app -e POSTGRESQL_USER=user -e POSTGRESQL_PASSWORD=mypassword -e POSTGRESQL_DATABASE=sampledb postgresql:9.5
oc apply -f deploy/crs/vdb_v1alpha1_virtualdatabase_cr.yaml
```
