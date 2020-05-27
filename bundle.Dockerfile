FROM scratch

LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/
LABEL operators.operatorframework.io.bundle.package.v1=teiid-operator
LABEL operators.operatorframework.io.bundle.channels.v1=0.x
LABEL operators.operatorframework.io.bundle.channel.default.v1=0.x

COPY deploy/olm-catalog/teiid-operator/manifests /manifests/
COPY deploy/olm-catalog/teiid-operator/metadata/annotations.yaml /metadata/annotations.yaml
