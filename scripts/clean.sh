#!/bin/sh
IMAGE=$1

sed "s|\$IMAGE_LOCATION|${IMAGE}|g" deploy/operator.yaml > deploy/operator-`whoami`.yaml

for FILE in deploy/operator-`whoami`.yaml deploy/role_binding.yaml deploy/service_account.yaml deploy/role.yaml  deploy/crds/teiid.io_virtualdatabases_crd.yaml
do
	oc delete -f ${FILE}
done
rm deploy/operator-`whoami`.yaml