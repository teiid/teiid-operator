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
	// S2IBuildImageName --
	S2IBuildImageName = "s2i-java"
	//S2IBuildImageTag --
	S2IBuildImageTag = "latest-java11"
	// SpringBoot --
	SpringBoot = "SpringBoot"
)

// SpringBootRuntime --
var SpringBootRuntime = v1alpha1.RuntimeType{
	Type:    SpringBoot,
	Version: TeiidSpringBootVersion,
}

// RuntimeImageDefaults ...
var RuntimeImageDefaults = map[string][]v1alpha1.Image{
	SpringBootRuntime.Type: {
		{
			ImageStreamName:      S2IBuildImageName,
			ImageStreamNamespace: ImageStreamNamespace,
			ImageStreamTag:       S2IBuildImageTag,
			ImageRegistry:        ImageRegistry,
			ImageRepository:      ImageRepo,
			BuilderImage:         true,
		},
		{
			BuilderImage: false,
		},
	},
}
