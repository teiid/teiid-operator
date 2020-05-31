#!/bin/sh
IMAGE=$1
CRC_IMAGE=$2
CRC_EXTERNAL=default-route-openshift-image-registry.apps-crc.testing


if [[ $IMAGE =~ ^$CRC_EXTERNAL ]]
then
  IMAGE=$CRC_IMAGE  
fi

sed "s|quay\.io\/teiid\/teiid-operator\:latest|${IMAGE}|g" deploy/operator.yaml > deploy/operator-`whoami`.yaml

for FILE in deploy/crds/teiid.io_virtualdatabases_crd.yaml deploy/role.yaml deploy/cluster_role.yaml deploy/service_account.yaml deploy/role_binding.yaml deploy/cluster_role_binding.yaml deploy/operator-`whoami`.yaml
do
	oc apply -f ${FILE}
done
rm deploy/operator-`whoami`.yaml
