#!/bin/bash

# taken from https://stackoverflow.com/questions/5014632/how-can-i-parse-a-yaml-file-from-a-linux-shell-script
function parse_yaml {
   local prefix=$2
   local s='[[:space:]]*' w='[a-zA-Z0-9_]*' fs=$(echo @|tr @ '\034')
   sed -ne "s|^\($s\):|\1|" \
        -e "s|^\($s\)\($w\)$s:$s[\"']\(.*\)[\"']$s\$|\1$fs\2$fs\3|p" \
        -e "s|^\($s\)\($w\)$s:$s\(.*\)$s\$|\1$fs\2$fs\3|p"  $1 |
   awk -F$fs '{
      indent = length($1)/2;
      vname[indent] = $2;
      for (i in vname) {if (i > indent) {delete vname[i]}}
      if (length($3) > 0) {
         vn=""; for (i=0; i<indent; i++) {vn=(vn)(vname[i])("_")}
         printf("%s%s%s=%s\n", "'$prefix'",vn, $2, $3);
      }
   }'
}

OPENAPIGEN=openapi-gen
command -v ${OPENAPIGEN} > /dev/null
if [ $? != 0 ]; then
    if [ -n ${GOPATH} ]; then
        OPENAPIGEN="${GOPATH}/bin/openapi-gen"
    fi
fi

CLIENTGEN=client-gen
command -v ${CLIENTGEN} > /dev/null
if [ $? != 0 ]; then
    if [ -n ${GOPATH} ]; then
        CLIENTGEN="${GOPATH}/bin/client-gen"
    fi
fi

# generate the CRD(s)
operator-sdk generate crds
RT=$?
if [ ${RT} != 0 ]; then
    echo "Failed to generate CRDs."
    exit ${RT}
fi

# generate the schema validation (openapi) stubs
${OPENAPIGEN} --logtostderr=true -o "" -i ./pkg/apis/teiid/v1alpha1 -O zz_generated.openapi -p ./pkg/apis/teiid/v1alpha1 -h /dev/null -r "-"
RT=$?
if [ ${RT} != 0 ]; then
    echo "Failed to generate the openapi (schema validation) stubs."
    exit ${RT}
fi

# generate the Kubernetes stubs
operator-sdk generate k8s
RT=$?
if [ ${RT} != 0 ]; then
    echo "Failed to generate the Kubernetes stubs."
    exit ${RT}
fi

# generate the connection-factories.json file
eval $(parse_yaml ./build/conf/config.yaml)
sed "s|teiidSpringBootVersion|${teiidSpringBootVersion}|g" ./scripts/connections/pom.xml > pom-temp.xml
mvn -s ./scripts/connections/settings.xml -f pom-temp.xml exec:java
rm pom-temp.xml
