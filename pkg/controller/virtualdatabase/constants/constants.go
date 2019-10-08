package constants

import "github.com/teiid/teiid-operator/pkg/apis/vdb/v1alpha1"
import "github.com/teiid/teiid-operator/pkg/util/conf"

const (
	// Version --
	Version = "0.0.1"
	// BuilderImageTargetName target build image name
	BuilderImageTargetName = "virtualdatabase-builder"
	// SpringBoot --
	SpringBoot = "SpringBoot"
)

// Config from /conf/config.yml file
var Config = conf.GetConfiguration()

// SpringBootRuntime --
var SpringBootRuntime = v1alpha1.RuntimeType{
	Type:    SpringBoot,
	Version: Config.TeiidSpringBootVersion,
}

// RuntimeImageDefaults ...
var RuntimeImageDefaults = map[string][]v1alpha1.Image{
	SpringBootRuntime.Type: {
		{
			ImageStreamName:      Config.BuildImage.Name,
			ImageStreamNamespace: Config.BuildImage.Namespace,
			ImageStreamTag:       Config.BuildImage.Tag,
			ImageRegistry:        Config.BuildImage.Registry,
			ImageRepository:      Config.BuildImage.Repo,
			BuilderImage:         true,
		},
		{
			BuilderImage: false,
		},
	},
}
