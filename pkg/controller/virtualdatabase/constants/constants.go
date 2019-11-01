package constants

import "github.com/teiid/teiid-operator/pkg/apis/vdb/v1alpha1"
import "github.com/teiid/teiid-operator/pkg/util/conf"

const (
	// Version --
	Version = "0.1.0"
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
