/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package virtualdatabase

import (
	"context"
	"encoding/xml"
	"errors"
	"strings"

	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/controller/virtualdatabase/constants"
	"github.com/teiid/teiid-operator/pkg/util/conf"
	"github.com/teiid/teiid-operator/pkg/util/envvar"
	"github.com/teiid/teiid-operator/pkg/util/kubernetes"
	"github.com/teiid/teiid-operator/pkg/util/maven"
	"github.com/teiid/teiid-operator/pkg/util/vdbutil"
	corev1 "k8s.io/api/core/v1"
)

func addDependency(project *maven.Project, sourceType string, cf conf.ConnectionFactory) error {
	for _, gav := range cf.Gav {
		dependency, err := maven.ParseGAV(gav)
		if err != nil {
			return err
		}
		if dependency.Version == "" && constants.Config.Drivers[sourceType] != "" {
			dependency.Version = constants.Config.Drivers[sourceType]
		} else if dependency.Version == "" && constants.Config.Drivers[sourceType] == "" && cf.JdbcSource {
			continue
		} else if dependency.Version == "" && strings.HasPrefix(dependency.GroupID, "org.teiid") {
			dependency.Version = constants.Config.TeiidSpringBootVersion
		} else {
			return errors.New("No version defined for Dependency " + gav + ". Please provide the version")
		}
		project.AddDependencies(dependency)
	}
	return nil
}

// GenerateVdbPom -- Generate the POM file based on the VDb provided
func GenerateVdbPom(vdb *v1alpha1.VirtualDatabase, sources []vdbutil.DatasourceInfo,
	includeAllDependencies bool, includeOpenAPIAdependency bool, includeIspnDependency bool) (maven.Project, error) {
	// do code generation.
	// generate pom.xml
	project := createMavenProject(vdb.ObjectMeta.Name)

	mavenRepos := constants.GetMavenRepositories(vdb)
	for k, v := range mavenRepos {
		project.AddRepository(maven.NewRepository(v + "@id=" + k))
		project.AddPluginRepository(maven.NewRepository(v + "@id=" + k))
	}

	// looking that the CRD we need to fill in the dependencies
	for _, str := range vdb.Spec.Build.Source.Dependencies {
		d, err := maven.ParseGAV(str)
		if err != nil {
			return project, err
		}
		project.AddDependencies(d)
	}

	if includeAllDependencies {
		for k, v := range constants.ConnectionFactories {
			if err := addDependency(&project, k, v); err != nil {
				return project, err
			}
		}
	} else {
		for _, s := range sources {
			if v, ok := constants.ConnectionFactories[s.Type]; ok {
				if err := addDependency(&project, s.Type, v); err != nil {
					return project, err
				}
			} else {
				log.Info("No predefined Connection Factory found for ", s, " Treating as custom source, dependency must de defined in YAML file")
			}
		}
	}

	if vdb.Spec.ExposeVia3Scale {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.teiid",
			ArtifactID: "spring-keycloak",
			Version:    constants.Config.TeiidSpringBootVersion,
		})
	}

	if includeIspnDependency {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.teiid",
			ArtifactID: "spring-data-infinispan-hotrod",
			Version:    constants.Config.TeiidSpringBootVersion,
		})
	}

	// add keyclock based security
	if includeAllDependencies || envvar.Get(vdb.Spec.Env, "KEYCLOAK_AUTH_SERVER_URL") != nil {
		log.Info("KEYCLOAK_AUTH_SERVER_URL found, enabling security module")
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.teiid",
			ArtifactID: "spring-keycloak",
			Version:    constants.Config.TeiidSpringBootVersion,
		})
	}

	if includeOpenAPIAdependency || includeAllDependencies {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.teiid",
			ArtifactID: "spring-openapi",
			Version:    constants.Config.TeiidSpringBootVersion,
		})
	}

	if !includeOpenAPIAdependency || includeAllDependencies {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.teiid",
			ArtifactID: "spring-odata",
			Version:    constants.Config.TeiidSpringBootVersion,
		})
	}

	project.AddDependencies(maven.Dependency{
		GroupID:    "me.snowdrop",
		ArtifactID: "narayana-spring-boot-starter",
		Version:    constants.Config.Drivers["narayana"],
	})

	return project, nil
}

// GenerateJarPom -- Generate the POM file based on the VDB provided
func GenerateJarPom(vdb *v1alpha1.VirtualDatabase) (maven.Project, error) {
	// do code generation.
	// generate pom.xml
	project := createPlainMavenProject(vdb.ObjectMeta.Name)

	mavenRepos := constants.GetMavenRepositories(vdb)
	for k, v := range mavenRepos {
		project.AddRepository(maven.NewRepository(v + "@id=" + k))
		project.AddPluginRepository(maven.NewRepository(v + "@id=" + k))
	}

	return project, nil
}

func addCopyPlugIn(vdbDependency maven.Dependency, artifactType string, targetName string, outputDirectory string, project *maven.Project) {
	// build the plugin to grab the VDB from maven repo and make it part of the package
	plugin := maven.Plugin{
		GroupID:    "org.apache.maven.plugins",
		ArtifactID: "maven-dependency-plugin",
		Executions: []maven.Execution{
			{
				ID:    "copy-vdb",
				Phase: "generate-sources",
				Goals: []string{
					"copy",
				},
				Configuration: maven.Configuration{
					ArtifactItems: []maven.ArtifactItem{
						{
							GroupID:             vdbDependency.GroupID,
							ArtifactID:          vdbDependency.ArtifactID,
							Version:             vdbDependency.Version,
							Type:                artifactType,
							DestinationFileName: targetName,
						},
					},
					OutputDirectory: outputDirectory,
				},
			},
		},
	}
	project.AddBuildPlugin(plugin)
}

func addVdbCodeGenPlugIn(project *maven.Project, vdbFilePath string, materializationEnable bool, version string) {
	plugin := maven.Plugin{
		GroupID:    "org.teiid",
		ArtifactID: "vdb-codegen-plugin",
		Version:    constants.Config.TeiidSpringBootVersion,
		Executions: []maven.Execution{
			{
				Goals: []string{
					"vdb-codegen",
				},
				ID:    "codegen",
				Phase: "generate-sources",
				Configuration: maven.Configuration{
					VdbFile:               vdbFilePath,
					MaterializationType:   "infinispan-hotrod",
					MaterializationEnable: materializationEnable,
					VdbVersion:            version,
				},
			},
		},
	}
	project.PrependBuildPlugin(plugin)
}

func createMavenProject(name string) maven.Project {
	project := maven.Project{
		XMLName:           xml.Name{Local: "project"},
		XMLNs:             "http://maven.apache.org/POM/4.0.0",
		XMLNsXsi:          "http://www.w3.org/2001/XMLSchema-instance",
		XsiSchemaLocation: "http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd",
		ModelVersion:      "4.0.0",
		GroupID:           "io.integration",
		ArtifactID:        name,
		Version:           "1.0.0",
		Packaging:         "jar",

		Dependencies: []maven.Dependency{
			{
				GroupID:    "org.teiid",
				ArtifactID: "teiid-spring-boot-starter",
				Version:    constants.Config.TeiidSpringBootVersion,
			},
			{
				GroupID:    "org.springframework.boot",
				ArtifactID: "spring-boot-starter-actuator",
				Version:    constants.Config.SpringBootVersion,
			},
			{
				GroupID:    "io.opentracing.contrib",
				ArtifactID: "opentracing-spring-jaeger-web-starter",
				Version:    "1.0.1",
			},
			{
				GroupID:    "com.h2database",
				ArtifactID: "h2",
				Version:    "1.4.199",
			},
		},
		Repositories: []maven.Repository{
			{
				ID:  "snapshots-repo",
				URL: "https://oss.sonatype.org/content/repositories/snapshots",
				Snapshots: maven.RepositoryPolicy{
					Enabled: true,
				},
				Releases: maven.RepositoryPolicy{
					Enabled: false,
				},
			},
		},
		PluginRepositories: []maven.Repository{
			{
				ID:  "snapshots-repo",
				URL: "https://oss.sonatype.org/content/repositories/snapshots",
				Snapshots: maven.RepositoryPolicy{
					Enabled: true,
				},
				Releases: maven.RepositoryPolicy{
					Enabled: false,
				},
			},
		},
		Build: maven.Build{
			Plugins: []maven.Plugin{
				{
					GroupID:    "org.springframework.boot",
					ArtifactID: "spring-boot-maven-plugin",
					Version:    constants.Config.SpringBootVersion,
					Executions: []maven.Execution{
						{
							Goals: []string{
								"repackage",
							},
							ID:    "repackage",
							Phase: "package",
						},
					},
				},
			},
		},
	}
	return project
}

func createPlainMavenProject(name string) maven.Project {
	project := maven.Project{
		XMLName:           xml.Name{Local: "project"},
		XMLNs:             "http://maven.apache.org/POM/4.0.0",
		XMLNsXsi:          "http://www.w3.org/2001/XMLSchema-instance",
		XsiSchemaLocation: "http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd",
		ModelVersion:      "4.0.0",
		GroupID:           "io.integration",
		ArtifactID:        name,
		Version:           "1.0.0",
		Packaging:         "pom",

		Repositories: []maven.Repository{
			{
				ID:  "snapshots-repo",
				URL: "https://oss.sonatype.org/content/repositories/snapshots",
				Snapshots: maven.RepositoryPolicy{
					Enabled: true,
				},
				Releases: maven.RepositoryPolicy{
					Enabled: false,
				},
			},
		},
		PluginRepositories: []maven.Repository{
			{
				ID:  "snapshots-repo",
				URL: "https://oss.sonatype.org/content/repositories/snapshots",
				Snapshots: maven.RepositoryPolicy{
					Enabled: true,
				},
				Releases: maven.RepositoryPolicy{
					Enabled: false,
				},
			},
		},
		Build: maven.Build{
			Plugins: []maven.Plugin{},
		},
	}
	return project
}

func readMavenSettingsFile(ctx context.Context, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase, pom maven.Project) (string, error) {
	settingsContent, err := maven.EncodeXML(maven.NewDefaultSettings(pom.Repositories))
	if vdb.Spec.Build.Source.MavenSettings.ConfigMapKeyRef != nil || vdb.Spec.Build.Source.MavenSettings.SecretKeyRef != nil {
		settingsContent, err = kubernetes.ResolveValueSource(ctx, r.client, vdb.ObjectMeta.Namespace, &vdb.Spec.Build.Source.MavenSettings)
	} else if kubernetes.HasConfigMap(ctx, r.client, "teiid-maven-settings", vdb.ObjectMeta.Namespace) {
		selector := &corev1.ConfigMapKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: "teiid-maven-settings",
			},
			Key: "settings.xml",
		}
		settingsContent, err = kubernetes.GetConfigMapRefValue(ctx, r.client, vdb.ObjectMeta.Namespace, selector)
	}
	return settingsContent, err
}
