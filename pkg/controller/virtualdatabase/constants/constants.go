package constants

import "github.com/teiid/teiid-operator/pkg/apis/vdb/v1alpha1"

const (
	// Version --
	Version = "0.0.1-SNAPSHOT"

	// TeiidSpringBootVersion --
	TeiidSpringBootVersion = "1.2.0-SNAPSHOT"

	// SpringBootVersion --
	SpringBootVersion = "2.1.3.RELEASE"

	// PostgreSQLVersion --
	PostgreSQLVersion = "42.1.4"
	// MySQLVersion --
	MySQLVersion = "5.1.40"
	// MongoDBVersion --
	MongoDBVersion = "3.6.3"

	// ImageStreamNamespace default namespace for the ImageStreams
	ImageStreamNamespace = "openshift"
	// ImageRegistry ...
	ImageRegistry = "docker.io"
	// ImageRepo ...
	ImageRepo = "fabric8"
	// ImageStreamTag default tag name for the ImageStreams
	ImageStreamTag = "latest" // "latest-java11"
	// BuilderImageTargetName target build image name
	BuilderImageTargetName = "virtualdatabase-builder"
)

// RuntimeImageDefaults ...
var RuntimeImageDefaults = map[v1alpha1.RuntimeType][]v1alpha1.Image{
	v1alpha1.SpringbootRuntimeType: {
		{
			ImageStreamName:      "s2i-java",
			ImageStreamNamespace: ImageStreamNamespace,
			ImageStreamTag:       "latest-java11",
			ImageRegistry:        ImageRegistry,
			ImageRepository:      ImageRepo,
			BuilderImage:         true,
		},
		{
			BuilderImage: false,
		},
	},
}
