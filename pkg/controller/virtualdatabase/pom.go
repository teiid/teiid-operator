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
	"encoding/xml"
	"strings"

	"github.com/teiid/teiid-operator/pkg/apis/vdb/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/controller/virtualdatabase/constants"
	"github.com/teiid/teiid-operator/pkg/util/envvar"
	"github.com/teiid/teiid-operator/pkg/util/maven"
)

// GenerateVdbPom -- Generate the POM file based on the VDb provided
func GenerateVdbPom(vdb *v1alpha1.VirtualDatabase, ddl string, includeAllDependencies bool, includeOpenAPIAdependency bool) (maven.Project, error) {
	// do code generation.
	// generate pom.xml
	project := createMavenProject(vdb.ObjectMeta.Name)

	mavenRepos := vdb.Spec.Build.Source.MavenRepositories
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

	if includeAllDependencies || strings.Contains(lowerDDL, "google") {
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

	if includeOpenAPIAdependency {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.teiid",
			ArtifactID: "spring-openapi",
			Version:    constants.Config.TeiidSpringBootVersion,
		})
	} else {
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

	mavenRepos := vdb.Spec.Build.Source.MavenRepositories
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
