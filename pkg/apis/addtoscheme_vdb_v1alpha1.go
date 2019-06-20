package apis

import (
	oappsv1 "github.com/openshift/api/apps/v1"
	buildv1 "github.com/openshift/api/build/v1"
	oimagev1 "github.com/openshift/api/image/v1"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/teiid/teiid-operator/pkg/apis/vdb/v1alpha1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes,
		v1alpha1.SchemeBuilder.AddToScheme,
		oappsv1.AddToScheme,
		routev1.AddToScheme,
		oimagev1.AddToScheme,
		buildv1.AddToScheme,
	)
}
