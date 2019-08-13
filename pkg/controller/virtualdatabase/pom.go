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
	"github.com/teiid/teiid-operator/pkg/util/maven"
)

// GeneratePom -- Generate the POM file based on the VDb provided
func GeneratePom(vdb *v1alpha1.VirtualDatabase, includeAllDependencies bool, includeOpenAPIAdependency bool) (string, error) {
	// do code generation.
	// generate pom.xml
	project := createMavenProject(vdb.ObjectMeta.Name)

	ddl := vdb.Spec.Build.Source.DDL

	// looking that the CRD we need to fill in the dependencies
	for _, str := range vdb.Spec.Build.Source.Dependencies {
		d, err := maven.ParseGAV(str)
		if err != nil {
			return "", err
		}
		project.AddDependencies(d)
	}

	// be smarter and look for implicit dependencies?
	lowerDDL := strings.ToLower(ddl)
	if includeAllDependencies || strings.Contains(lowerDDL, "postgresql") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.postgresql",
			ArtifactID: "postgresql",
			Version:    constants.PostgreSQLVersion,
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "mysql") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "mysql",
			ArtifactID: "mysql-connector-java",
			Version:    constants.MySQLVersion,
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "mongodb") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.mongodb",
			ArtifactID: "mongo-java-driver",
			Version:    constants.MongoDBVersion,
		})
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.teiid",
			ArtifactID: "spring-data-mongodb",
			Version:    constants.TeiidSpringBootVersion,
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "google") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.teiid",
			ArtifactID: "spring-data-google",
			Version:    constants.TeiidSpringBootVersion,
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "salesforce") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.teiid",
			ArtifactID: "spring-data-salesforce",
			Version:    constants.TeiidSpringBootVersion,
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "excel") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.teiid",
			ArtifactID: "spring-data-excel",
			Version:    constants.TeiidSpringBootVersion,
		})
	}

	if includeAllDependencies || strings.Contains(lowerDDL, "rest") || strings.Contains(lowerDDL, "ws") {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.teiid",
			ArtifactID: "spring-data-rest",
			Version:    constants.TeiidSpringBootVersion,
		})
	}

	if includeAllDependencies || vdb.Spec.ExposeVia3Scale {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.teiid",
			ArtifactID: "spring-keycloak",
			Version:    constants.TeiidSpringBootVersion,
		})
	}

	if includeOpenAPIAdependency {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.teiid",
			ArtifactID: "spring-openapi",
			Version:    constants.TeiidSpringBootVersion,
		})
	} else {
		project.AddDependencies(maven.Dependency{
			GroupID:    "org.teiid",
			ArtifactID: "spring-odata",
			Version:    constants.TeiidSpringBootVersion,
		})
	}

	return maven.GeneratePomContent(project)
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
				Version:    constants.TeiidSpringBootVersion,
			},
			{
				GroupID:    "org.springframework.boot",
				ArtifactID: "spring-boot-starter-actuator",
				Version:    constants.SpringBootVersion,
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
					GroupID:    "org.teiid",
					ArtifactID: "vdb-codegen-plugin",
					Version:    constants.TeiidSpringBootVersion,
					Executions: []maven.Execution{
						{
							Goals: []string{
								"vdb-codegen",
							},
							ID:    "codegen",
							Phase: "generate-sources",
						},
					},
				},
				{
					GroupID:    "org.springframework.boot",
					ArtifactID: "spring-boot-maven-plugin",
					Version:    constants.SpringBootVersion,
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
