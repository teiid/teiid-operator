#!/bin/sh
for FILE in deploy/crds/virtualdatabase.crd.yaml deploy/role.yaml deploy/service_account.yaml deploy/role_binding.yaml deploy/operator.yaml
do
	oc apply -f ${FILE}
done