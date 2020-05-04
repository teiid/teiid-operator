#!/bin/sh
IMAGE=$1

sed "s|quay\.io\/teiid\/teiid-operator\:8\.0\.0|${IMAGE}|g" deploy/operator.yaml > deploy/operator-`whoami`.yaml

for FILE in deploy/crds/teiid.io_virtualdatabases_crd.yaml deploy/role.yaml deploy/service_account.yaml deploy/role_binding.yaml deploy/operator-`whoami`.yaml
do
	oc apply -f ${FILE}
done
rm deploy/operator-`whoami`.yaml