package constants

import (
	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/util/conf"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	// Version --
	Version = "0.3.0"
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

// GetMavenRepositories --
func GetMavenRepositories(vdb *v1alpha1.VirtualDatabase) map[string]string {

	repos := make(map[string]string)
	// configure default repositories
	if len(vdb.Spec.Build.Source.MavenRepositories) != 0 {
		for k, v := range vdb.Spec.Build.Source.MavenRepositories {
			repos[k] = v
		}
	} else {
		if len(Config.MavenRepositories) != 0 {
			for k, v := range Config.MavenRepositories {
				repos[k] = v
			}
		}
	}
	return repos
}

// GetComputingResources --
func GetComputingResources(vdb *v1alpha1.VirtualDatabase) corev1.ResourceRequirements {
	// resources for the container
	if &vdb.Spec.Resources == nil {
		return corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				"memory": resource.MustParse("512Mi"),
				"cpu":    resource.MustParse("1.0"),
			},
			Requests: corev1.ResourceList{
				"memory": resource.MustParse("256Mi"),
				"cpu":    resource.MustParse("0.2"),
			},
		}
	}
	return vdb.Spec.Resources
}
