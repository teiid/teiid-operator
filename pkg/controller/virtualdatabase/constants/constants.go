package constants

import (
	"github.com/teiid/teiid-operator/pkg/apis/vdb/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/util/conf"
)

const (
	// Version --
	Version = "0.2.0"
	// BuilderImageTargetName target build image name
	BuilderImageTargetName = "virtualdatabase-builder"
	// TSB --
	TSB = "teiid-spring-boot"
)

// Config from /conf/config.yml file
var Config = conf.GetConfiguration()

// SpringBootRuntime --
var SpringBootRuntime = v1alpha1.RuntimeType{
	Type:    TSB,
	Version: Config.TeiidSpringBootVersion,
}
