package constants

import (
	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/util/conf"
)

const (
	// Version --
	Version = "0.2.0"
	// BuilderImageTargetName target build image name
	BuilderImageTargetName = "virtualdatabase-builder"
	// TSB --
	TSB = "teiid-spring-boot"

	// KeystoreLocation --
	KeystoreLocation = "/etc/tls/private"
	// KeystorePassword --
	KeystorePassword = "changeit"
	// KeystoreName --
	KeystoreName = "keystore.pkcs12"
	// TruststoreName --
	TruststoreName = "truststore.pkcs12"
)

// Config from /conf/config.yml file
var Config = conf.GetConfiguration()

// SpringBootRuntime --
var SpringBootRuntime = v1alpha1.RuntimeType{
	Type:    TSB,
	Version: Config.TeiidSpringBootVersion,
}
