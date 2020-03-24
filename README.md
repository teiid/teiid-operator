# Teiid Operator

Teiid Operator for OpenShift/Kubernetes

[![Go Report](https://goreportcard.com/badge/github.com/teiid/teiid-operator)](https://goreportcard.com/report/github.com/teiid/teiid-operator)
[![Build Status](https://travis-ci.org/teiid/teiid-operator.svg?branch=master)](https://travis-ci.org/teiid/teiid-operator)

## Development Of Operator

### Requirements

- go v1.13+
- operator-sdk v0.15.0+
- buildah v1.14.2+

### SetUp the OpenShift 4.x

Using Code Ready Containers, create OpenShift environment. Take look at [Code Ready Containers](https://developers.redhat.com/products/codeready-containers)

Once you downloaded the CRC follow the below steps. It is expected that your laptop has 12GB of memory, 8 CPUS, 60GB of disk space available for this setup

```bash
crc stop
crc delete
crc config set memory 12288
crc config set cpus 8
sudo qemu-img resize ~/.crc/machines/crc/crc +30G
sudo qemu-img info ~/.crc/machines/crc/crc | grep 'virtual size'

crc start

oc login {find-addess-from-last-statement}

oc adm policy --as system:admin add-cluster-role-to-user cluster-admin developer

# after startup
# The below is to increase the disk size by 30GB
crc ip
ssh -i /home/${your-username}/.crc/machines/crc/id_rsa core@192.168.130.11
sudo xfs_growfs /sysroot
df -h
```

### Install

To set up you environment please do following

```bash
git clone git@github.com:teiid/teiid-operator.git
cd teiid-operator

make install
```

### Build

```bash
make build
```

Before submitting PR, please be sure to generate, vet, format, and test your code. This all can be done with one command.

```bash
make ci
```

### Deploy

To deploy the Operator to locally running Openshift or wherever the oc command connected instance

```bash
make deploy
```

### Cleanup

To remove the Operator from locally deployed instance run following

```bash
make clean
```

## OLM Notes

https://github.com/operator-framework/community-operators/blob/master/docs/contributing.md

## Operator Installing on Openshift 4.x using OLM (OperatorHub)

Here the script is showing to move to version `0.2.0` from `0.1.0` but note that either current version or your working version may be different

```bash
# Generate CSV from Code
$operator-sdk generate csv --csv-channel beta --csv-version 0.2.0 --from-version 0.1.0 --operator-name teiid

# fix couple of attributes in CSV File
containerImage: fix this
teiid-operator/Image  - fix this

# copy additional files into the olm-catalog directory
$cp crd into olm-catalog/teiid/{version}
$cp package olm-catalog/teiid/{version}

# push image to quay.io
$docker push quay.io/teiid/teiid-operator:0.2.0

# start OpenShift 4.x and give developer few rols
$oc adm policy --as system:admin add-cluster-role-to-user cluster-admin developer

# quay login for testing
$curl -sH "Content-Type: application/json" -XPOST https://quay.io/cnr/api/v1/users/login -d '
{
    "user": {
        "username": "'"${QUAY_USERNAME}"'",
        "password": "'"${QUAY_PASSWORD}"'"
    }
}'

# deploy operator in OpenShift 4.x
$operator-courier push deploy/olm-catalog/teiid/0.2.0 ${user-name} teiid 0.2.0 "basic token_from_login"

# run Operator source to push the Operator to local OperatorHub for OKD
oc apply -f deploy/olm-catalog/teiid/teiid-operatorsource.yaml

# Run the scorecard test (optional, only for testing)
$operator-sdk scorecard --bundle deploy/olm-catalog/teiid/0.2.0-SNAPSHOT/
```
