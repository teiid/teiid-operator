module github.com/teiid/teiid-operator

require (
	//github.com/Azure/go-autorest/autorest v0.9.3 // indirect
	github.com/RHsyseng/operator-utils v0.0.0-20200417214513-7aac0c82a293
	github.com/coreos/prometheus-operator v0.38.0
	github.com/emicklei/go-restful v2.11.1+incompatible // indirect
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-logr/logr v0.1.0
	github.com/go-openapi/spec v0.19.4
	github.com/go-openapi/validate v0.19.5 // indirect
	github.com/hetesiistvan/go-pkcs12 v0.0.0-20191008101753-f02f3241080a
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/infinispan/infinispan-operator v0.0.0-20200421123311-ca42dade66d1
	github.com/jaegertracing/jaeger-operator v1.16.0
	github.com/magiconair/properties v1.8.1
	github.com/openshift/api v3.9.1-0.20190927182313-d4a64ec2cbd8+incompatible
	github.com/openshift/client-go v0.0.0-20190923180330-3b6373338c9b
	github.com/operator-framework/operator-sdk v0.17.0
	github.com/pkg/errors v0.9.1
	github.com/scylladb/go-set v1.0.2
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.4.0
	go.uber.org/zap v1.14.1
	golang.org/x/crypto v0.0.0-20200214034016-1d94cc7ab1c6 // indirect
	golang.org/x/sys v0.0.0-20200219091948-cb0a6d8edb6c // indirect
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.17.4
	k8s.io/apimachinery v0.17.4
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20191107075043-30be4d16710a
	sigs.k8s.io/controller-runtime v0.5.2
)

// Pinned to kubernetes-1.16.2
replace (
	k8s.io/api => k8s.io/api v0.0.0-20191016110408-35e52d86657a
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20191016113550-5357c4baaf65
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191004115801-a2eda9f80ab8
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20191016112112-5190913f932d
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20191016114015-74ad18325ed5
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191016111102-bec269661e48
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20191016115326-20453efc2458
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20191016115129-c07a134afb42
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20191004115455-8e001e5d1894
	k8s.io/component-base => k8s.io/component-base v0.0.0-20191016111319-039242c015a9
	k8s.io/cri-api => k8s.io/cri-api v0.0.0-20190828162817-608eb1dad4ac
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20191016115521-756ffa5af0bd
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20191016112429-9587704a8ad4
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20191016114939-2b2b218dc1df
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20191016114407-2e83b6f20229
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20191016114748-65049c67a58b
	k8s.io/kubectl => k8s.io/kubectl v0.0.0-20191016120415-2ed914427d51
	k8s.io/kubelet => k8s.io/kubelet v0.0.0-20191016114556-7841ed97f1b2
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20191016115753-cf0698c3a16b
	k8s.io/metrics => k8s.io/metrics v0.0.0-20191016113814-3b1a734dba6e
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20191016112829-06bb3c9d77c9
)

replace github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309 // Required by Helm

replace github.com/openshift/api => github.com/openshift/api v0.0.0-20190924102528-32369d4db2ad // Required until https://github.com/operator-framework/operator-lifecycle-manager/pull/1241 is resolved

go 1.13
