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
	"os"
	"path"

	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/util/defaults"
	"github.com/teiid/teiid-operator/pkg/util/maven"
	"github.com/teiid/teiid-operator/pkg/util/tar"
)

func NewCodeGenerationAction() Action {
	return &codeGenerationAction{}
}

type codeGenerationAction struct {
	baseAction
}

// Name returns a common name of the action
func (action *codeGenerationAction) Name() string {
	return "code generation"
}

// CanHandle tells whether this action can handle the virtualdatabase
func (action *codeGenerationAction) CanHandle(vdb *v1alpha1.VirtualDatabase) bool {
	return vdb.Status.Phase == v1alpha1.PublishingPhaseCodeGeneration
}

// Handle handles the virtualdatabase
func (action *codeGenerationAction) Handle(ctx context.Context, vdb *v1alpha1.VirtualDatabase) error {
	// update the status
	target := vdb.DeepCopy()

	// do code generation.
	// generate pom.xml
	project := createMavenProject(vdb)

	//TODO: looking that the CRD we need to fill in the dependencies
	project.AddDependencies(maven.Dependency{
		GroupID:    "org.postgresql",
		ArtifactID: "postgresql",
	})

	target.Status.Phase = v1alpha1.PublishingPhaseCodeGenerationCompleted
	pom, err := maven.GeneratePomContent(project)
	if err != nil {
		target.Status.Phase = v1alpha1.PublishingPhaseInitial
		target.Status.Failure = "Failed to generate the pom.xml"
		action.Log.Info("Failed to generate the pom.xml", "phase", target.Status.Phase)
	} else {
		v := ctx.Value(v1alpha1.BuildStatusKey)
		_, err := createTarFileForBuild((v.(v1alpha1.BuildStatus)).TarFile, pom, vdb)
		if err != nil {
			target.Status.Phase = v1alpha1.PublishingPhaseInitial
			target.Status.Failure = "Failed to build Tar file for build"
			action.Log.Info("Failed to build Tar file for build", "phase", target.Status.Phase)
		}
		// // Need to do the build tomorrow.
		// action.client.Delete(ctx)
	}
	action.Log.Info("VDB state transition", "phase", target.Status.Phase)
	return action.client.Status().Update(ctx, target)
}

func createTarFileForBuild(tarFileName string, pom string, vdb *v1alpha1.VirtualDatabase) (*tar.Appender, error) {
	tarFileDir := path.Dir(tarFileName)
	err := os.MkdirAll(tarFileDir, 0777)
	if err != nil {
		return nil, err
	}

	tarAppender, err := tar.NewAppender(tarFileName)
	if err != nil {
		return nil, err
	}
	defer tarAppender.Close()
	tarAppender.AddData([]byte(pom), "pom.xml")
	tarAppender.AddData([]byte(vdb.Spec.Content), "src/main/resources/teiid-vdb.ddl")
	return tarAppender, nil
}

func createMavenProject(vdb *v1alpha1.VirtualDatabase) maven.Project {
	project := maven.Project{
		XMLName:           xml.Name{Local: "project"},
		XMLNs:             "http://maven.apache.org/POM/4.0.0",
		XMLNsXsi:          "http://www.w3.org/2001/XMLSchema-instance",
		XsiSchemaLocation: "http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd",
		ModelVersion:      "4.0.0",
		GroupID:           "org.teiid.virtualdatabase",
		ArtifactID:        vdb.ObjectMeta.Name,
		Version:           "1.0.0",
		DependencyManagement: maven.DependencyManagement{
			Dependencies: []maven.Dependency{
				{
					GroupID:    "org.teiid",
					ArtifactID: "teiid-spring-boot-starter-parent",
					Version:    defaults.TeiidVersion,
					Type:       "pom",
					Scope:      "import",
				},
			},
		},
		Dependencies: []maven.Dependency{
			{
				GroupID:    "org.teiid",
				ArtifactID: "teiid-spring-boot-starter",
			},
			{
				GroupID:    "org.teiid",
				ArtifactID: "spring-odata",
			},
			{
				GroupID:    "org.springframework.boot",
				ArtifactID: "spring-boot-starter-actuator",
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
		},
		Build: maven.Build{
			Plugins: []maven.Plugin{
				{
					GroupID:    "org.springframework.boot",
					ArtifactID: "spring-boot-maven-plugin",
					Version:    defaults.SpringBootVersion,
					Executions: []maven.Execution{
						{
							Goals: []string{
								"repackage",
							},
						},
					},
				},
				{
					GroupID:    "org.teiid",
					ArtifactID: "vdb-codegen-plugin",
					Version:    defaults.TeiidVersion,
					Executions: []maven.Execution{
						{
							Goals: []string{
								"vdb-codegen",
							},
						},
					},
				},
			},
		},
	}

	return project
}
