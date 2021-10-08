# Deprecation Note: 

This project is no longer being maintained.

# Teiid Operator

Teiid Operator for OpenShift/Kubernetes

[![Go Report](https://goreportcard.com/badge/github.com/teiid/teiid-operator)](https://goreportcard.com/report/github.com/teiid/teiid-operator)
[![Build Status](https://travis-ci.org/teiid/teiid-operator.svg?branch=master)](https://travis-ci.org/teiid/teiid-operator)

## Development Of Operator

### Requirements

- go v1.13+
- operator-sdk v0.17.0+ (will be installed automatically)
- buildah v1.14.2+
- golint
- yamllint
- Optional Docker Hub account

### SetUp the OpenShift 4.x Cluster (Code Ready Containers) on Laptop

Using Code Ready Containers, create OpenShift environment. Take look at [Code Ready Containers](https://developers.redhat.com/products/codeready-containers)

Once you downloaded the CRC follow the below steps. It is expected that your laptop has 12GB of memory, 8 CPUS, 60GB of disk space available for this setup

```bash
crc stop
crc delete
crc config set memory 12288
crc config set cpus 8
crc setup
# create a new instance
crc start
# shut down the instance to adjust disk size
crc stop
# default size is 30, up it to 60
sudo qemu-img resize ~/.crc/machines/crc/crc +30G
# verify the new size
sudo qemu-img info ~/.crc/machines/crc/crc | grep 'virtual size'
# we're ready, start again and give cluster admin to the developer account
crc start
eval $(crc oc-env)
oc login -u developer -p developer
oc adm policy --as system:admin add-cluster-role-to-user cluster-admin developer

# after startup
# The below is to increase the disk size to 30GB
ssh -i ~/.crc/machines/crc/id_rsa core@`crc ip`
sudo xfs_growfs /sysroot
# confirm the new size of sysroot
df -h
exit
```
Create a new project:
```bash
oc new-project {your-project-name}
```

If you do not have/want a Docker Hub account, then you need to target your crc registry:
```bash
# add access and login
oc extract secret/router-ca --keys=tls.crt -n openshift-ingress-operator
sudo mkdir -p /etc/docker/certs.d/default-route-openshift-image-registry.apps-crc.testing/ 
sudo cp tls.crt /etc/docker/certs.d/default-route-openshift-image-registry.apps-crc.testing/
sudo chmod 644 /etc/docker/certs.d/default-route-openshift-image-registry.apps-crc.testing/tls.crt
buildah login -u developer -p $(oc whoami -t) default-route-openshift-image-registry.apps-crc.testing
export REGISTRY=default-route-openshift-image-registry.apps-crc.testing/`oc project --short`
```

Otherwise if you have want to use your Docker Hub account note that the Makefile defaults to your local username as your dockerid.  
If your dockerid is differnt then do:
```bash
export REGISTRY={my-dockerid}
```

### Setup Teiid Operator Workspace

To set up your Teiid Operator workspace please do following tasks

```bash
git clone git@github.com:teiid/teiid-operator.git
cd teiid-operator

make install
```
This should install necessary `Go` libraries and tools.

### Build Teiid Operator

```bash
make build
```
When you run this command the Operator is built and a (docker) container image is created on your local machine under `{REGISTRY}/teiid-operator:{current-version}` using the `buildah` tool.

Before submitting PR, please be sure to generate, update-connections, vet, format, and test your code. This all can be done with one command.

```bash
make ci
```

### Updating the CRDs and Adding new data sources

If the Operator being released and if you need to update the CRD to a new version

```bash
make generate
```

This will generate the new CRDs


To pull the latest set of connection information for you version - generally only needed if you following a SNASHOT version - run

```bash
make update-connections
```

### Deploying Teiid Operator in OpenShift

To deploy the Operator to running Openshift that is installed above or to any Openshift cluster that you are already connected using the `oc` command, execute the following

```bash
make deploy
```
This will push the image to the regestry.  By default it will be your [Docker Hub](https://hub.docker.com/) account, then from there it will deploy into the available OpenShift instance.

NOTE: This is not going to use the OperatorHub, you are installing directly into the Namespace that you are connected to on the OpenShift. For OperatorHub see below.

### Cleanup

To remove the Operator from locally deployed instance run following

```bash
make clean
```
This will remove all the artifacts that have been installed as part of the previous step.

## OLM Notes

https://github.com/operator-framework/community-operators/blob/master/docs/contributing.md

## Operator Installing on Openshift 4.x using OLM (OperatorHub)

The directions here are very specific to testing the OLM based installation locally, to verify before it released to the OperatorHub.

The script below shows creating a CSV and then using it deploy to local OpenShift instance. At the end of the steps, you should be able to go to "OperatorHub" menu in the OpenShift 4.x and find the "Teiid" Operator inside it and install it.

### CSV Generation for OLM

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
```
This should generate couple files in `deploy/olm-catalog/teiid/{version}` folder that represents the CSV file. Make sure the image names match to the ones you are working with.

### Image Promotion to Quay.io & OpenShift

For this to work, one needs a `quay.io` account as OperatorHub does not work with Docker Hub. By default this pushes to the quay repository quay.io/teiid/teiid-operator.  If you have a different target repository set it via the QUAY_REPOSITORY env variable.

```bash
export QUAY_REPOSITORY=quay.io/my-org/teiid-operator
```

```bash
# push image to quay.io
$make quay-push

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
```
at this point after sometime, the OpenShift instance's OperatorHub menu should have listing for "Teiid" Operator.

### Opearator Score Card testing (TODO: This needs to be automated)

To see if the Operator CSV is up to the OLM team standards,run the tool and fix any issues that are found. 

NOTE: For this to work, one needs to install the Postgres Operator and install a database and populate schema for the `dv-customer` example, then run it. As below does deploy the example for verification purposes. 

```bash
# Run the scorecard test (optional, only for testing)
$operator-sdk scorecard --bundle deploy/olm-catalog/teiid/0.2.0-SNAPSHOT/
```
