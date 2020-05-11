package apis

import (
	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	oappsv1 "github.com/openshift/api/apps/v1"
	buildv1 "github.com/openshift/api/build/v1"
	consolev1 "github.com/openshift/api/console/v1"
	oimagev1 "github.com/openshift/api/image/v1"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"

	"github.com/teiid/teiid-operator/pkg/util/openshift"
	jaegerv1 "github.com/teiid/teiid-operator/pkg/util/opentracing/client/scheme"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes,
		v1alpha1.SchemeBuilder.AddToScheme,
		oappsv1.AddToScheme,
		routev1.AddToScheme,
		oimagev1.AddToScheme,
		buildv1.AddToScheme,
		monitoringv1.AddToScheme,
		jaegerv1.AddToScheme,
	)
	if err := openshift.ConsoleYAMLSampleExists(); err == nil {
		AddToSchemes = append(AddToSchemes, consolev1.Install)
	}
}
