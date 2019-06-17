package constants

import "github.com/teiid/teiid-operator/pkg/apis/vdb/v1alpha1"

const (
	// ImageStreamNamespace default namespace for the ImageStreams
	ImageStreamNamespace = "openshift"
	// ImageRegistry ...
	ImageRegistry = "docker.io"
	// ImageRepo ...
	ImageRepo = "fabric8"
	// ImageStreamTag default tag name for the ImageStreams
	ImageStreamTag = "latest" // "latest-java11"
)

// RuntimeImageDefaults ...
var RuntimeImageDefaults = map[v1alpha1.RuntimeType][]v1alpha1.Image{
	v1alpha1.KarafRuntimeType: {
		{
			ImageStreamName:      "s2i-karaf",
			ImageStreamNamespace: ImageStreamNamespace,
			ImageStreamTag:       ImageStreamTag,
			ImageRegistry:        ImageRegistry,
			ImageRepo:            ImageRepo,
			BuilderImage:         true,
		},
		{
			BuilderImage: false,
		},
	},
	v1alpha1.SpringbootRuntimeType: {
		{
			ImageStreamName:      "s2i-java",
			ImageStreamNamespace: ImageStreamNamespace,
			ImageStreamTag:       "latest-java11",
			ImageRegistry:        ImageRegistry,
			ImageRepo:            ImageRepo,
			BuilderImage:         true,
		},
		{
			BuilderImage: false,
		},
	},
}
