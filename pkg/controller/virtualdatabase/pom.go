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
	"strings"

	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/controller/virtualdatabase/constants"
	"github.com/teiid/teiid-operator/pkg/util/envvar"
	"github.com/teiid/teiid-operator/pkg/util/kubernetes"
	"github.com/teiid/teiid-operator/pkg/util/maven"
	corev1 "k8s.io/api/core/v1"
)

// GenerateVdbPom -- Generate the POM file based on the VDb provided
func GenerateVdbPom(vdb *v1alpha1.VirtualDatabase, ddl string, includeAllDependencies bool, includeOpenAPIAdependency bool) (maven.Project, error) {
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

	// be smarter and look for implicit dependencies?
	lowerDDL := strings.ToLower(ddl)
	if includeAllDependencies || strings.Contains(lowerDDL, "postgresql") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.postgresql",
			ArtifactID: "postgresql",
			Version:    constants.Config.Drivers["postgresql"],
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "mysql") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "mysql",
			ArtifactID: "mysql-connector-java",
			Version:    constants.Config.Drivers["mysql"],
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "h2") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "com.h2database",
			ArtifactID: "h2",
			Version:    constants.Config.Drivers["h2"],
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "sqlserver") || strings.Contains(lowerDDL, "mssql-server") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "com.microsoft.sqlserver",
			ArtifactID: "mssql-jdbc",
			Version:    constants.Config.Drivers["sqlserver"],
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "amazon-athena") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "com.syncron.amazonaws",
			ArtifactID: "simba-athena-jdbc-driver",
			Version:    constants.Config.Drivers["athena"],
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "db2") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "com.ibm.db2",
			ArtifactID: "jcc",
			Version:    constants.Config.Drivers["db2"],
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "hana") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "com.sap.cloud.db.jdbc",
			ArtifactID: "ngdbc",
			Version:    constants.Config.Drivers["hana"],
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "hbase") || strings.Contains(lowerDDL, "phoenix") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.apache.phoenix",
			ArtifactID: "phoenix-queryserver-client",
			Version:    constants.Config.Drivers["hbase"],
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "hive") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.apache.hive",
			ArtifactID: "hive-jdbc",
			Version:    constants.Config.Drivers["hive"],
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "hsql") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.hsqldb",
			ArtifactID: "hsqldb",
			Version:    constants.Config.Drivers["hsql"],
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "informix") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "com.ibm.informix",
			ArtifactID: "jdbc",
			Version:    constants.Config.Drivers["informix"],
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "ingres") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "com.ingres.jdbc",
			ArtifactID: "iijdbc",
			Version:    constants.Config.Drivers["ingres"],
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "jtds") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "net.sourceforge.jtds",
			ArtifactID: "jtds",
			Version:    constants.Config.Drivers["jtds"],
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "ucanacess") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "net.sf.ucanaccess",
			ArtifactID: "ucanaccess",
			Version:    constants.Config.Drivers["ucanacess"],
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "mongodb") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.mongodb",
			ArtifactID: "mongo-java-driver",
			Version:    constants.Config.Drivers["mongodb"],
		})
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.teiid",
			ArtifactID: "spring-data-mongodb",
			Version:    constants.Config.TeiidSpringBootVersion,
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "google-spreadsheet") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.teiid",
			ArtifactID: "spring-data-google",
			Version:    constants.Config.TeiidSpringBootVersion,
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "salesforce") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.teiid",
			ArtifactID: "spring-data-salesforce",
			Version:    constants.Config.TeiidSpringBootVersion,
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "excel") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.teiid",
			ArtifactID: "spring-data-excel",
			Version:    constants.Config.TeiidSpringBootVersion,
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "rest") || strings.Contains(lowerDDL, "ws") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.teiid",
			ArtifactID: "spring-data-rest",
			Version:    constants.Config.TeiidSpringBootVersion,
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "soap") || strings.Contains(lowerDDL, "ws") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.teiid",
			ArtifactID: "spring-data-soap",
			Version:    constants.Config.TeiidSpringBootVersion,
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "openapi") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.teiid",
			ArtifactID: "spring-data-openapi",
			Version:    constants.Config.TeiidSpringBootVersion,
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "infinispan-hotrod") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.teiid",
			ArtifactID: "spring-data-infinispan",
			Version:    constants.Config.TeiidSpringBootVersion,
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "amazon-s3") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.teiid",
			ArtifactID: "spring-data-amazon-s3",
			Version:    constants.Config.TeiidSpringBootVersion,
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "odata4") ||
		strings.Contains(lowerDDL, "sap-gateway") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.teiid",
			ArtifactID: "spring-data-rest",
			Version:    constants.Config.TeiidSpringBootVersion,
		})
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.teiid.connectors",
			ArtifactID: "translator-odata4",
			Version:    constants.Config.TeiidVersion,
		})
	}

	if includeAllDependencies || vdb.Spec.ExposeVia3Scale {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.teiid",
			ArtifactID: "spring-keycloak",
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

// GenerateJarPom -- Generate the POM file based on the VDb provided
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

func addVdbCodeGenPlugIn(project *maven.Project, vdbName string) {
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
					VdbFile: vdbName,
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
				ID:  "central",
				URL: "https://repo.maven.apache.org/maven2",
				Snapshots: maven.RepositoryPolicy{
					Enabled: false,
				},
				Releases: maven.RepositoryPolicy{
					Enabled:      true,
					UpdatePolicy: "never",
				},
			},
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
				ID:  "central",
				URL: "https://repo.maven.apache.org/maven2",
				Snapshots: maven.RepositoryPolicy{
					Enabled: false,
				},
				Releases: maven.RepositoryPolicy{
					Enabled:      true,
					UpdatePolicy: "never",
				},
			},
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
				ID:  "central",
				URL: "https://repo.maven.apache.org/maven2",
				Snapshots: maven.RepositoryPolicy{
					Enabled: false,
				},
				Releases: maven.RepositoryPolicy{
					Enabled:      true,
					UpdatePolicy: "never",
				},
			},
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
				ID:  "central",
				URL: "https://repo.maven.apache.org/maven2",
				Snapshots: maven.RepositoryPolicy{
					Enabled: false,
				},
				Releases: maven.RepositoryPolicy{
					Enabled:      true,
					UpdatePolicy: "never",
				},
			},
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
