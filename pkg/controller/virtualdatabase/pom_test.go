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
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/controller/virtualdatabase/constants"
	"github.com/teiid/teiid-operator/pkg/util/conf"
	"github.com/teiid/teiid-operator/pkg/util/maven"
	"github.com/teiid/teiid-operator/pkg/util/vdbutil"
	"gopkg.in/yaml.v2"
)

func TestParsingDataSources(t *testing.T) {

	contents, _ := ioutil.ReadFile("../../../deploy/crds/vdb_from_ddl.yaml")
	var vdb v1alpha1.VirtualDatabase
	err := yaml.Unmarshal(contents, &vdb)
	assert.Nil(t, err)

	dsInfo := vdbutil.ParseDataSourcesInfoFromDdl(vdb.Spec.Build.Source.DDL)

	assert.Equal(t, 1, len(dsInfo))
	assert.Equal(t, "sampledb", dsInfo[0].Name)
	assert.Equal(t, "postgresql", dsInfo[0].Type)
}

func TestPomGeneration(t *testing.T) {
	contents, _ := ioutil.ReadFile("../../../deploy/crds/vdb_from_ddl.yaml")
	var vdb v1alpha1.VirtualDatabase
	err := yaml.Unmarshal(contents, &vdb)
	assert.Nil(t, err)

	dsInfo := vdbutil.ParseDataSourcesInfoFromDdl(vdb.Spec.Build.Source.DDL)

	project, err := GenerateVdbPom(&vdb, dsInfo, false, false)
	assert.Nil(t, err)
	assert.True(t, hasDependency(project, "org.postgresql", "postgresql"))
	assert.True(t, hasDependency(project, "org.teiid", "teiid-spring-boot-starter"))
	assert.True(t, hasDependency(project, "org.springframework.boot", "spring-boot-starter-actuator"))
	assert.True(t, hasDependency(project, "io.opentracing.contrib", "opentracing-spring-jaeger-web-starter"))
	assert.True(t, hasDependency(project, "org.teiid", "spring-odata"))
	assert.True(t, hasDependency(project, "me.snowdrop", "narayana-spring-boot-starter"))
}

func hasDependency(project maven.Project, groupID string, artifactID string) bool {
	for _, d := range project.Dependencies {
		if d.GroupID == groupID && d.ArtifactID == artifactID && len(d.Version) > 0 {
			return true
		}
	}
	return false
}

func TestMinimalGavPomGeneration(t *testing.T) {
	contents, _ := ioutil.ReadFile("../../../deploy/crds/vdb_from_ddl.yaml")
	var vdb v1alpha1.VirtualDatabase
	err := yaml.Unmarshal(contents, &vdb)
	assert.Nil(t, err)

	dsInfo := vdbutil.ParseDataSourcesInfoFromDdl(vdb.Spec.Build.Source.DDL)
	dsInfo = append(dsInfo, vdbutil.DatasourceInfo{
		Name: "foo",
		Type: "foo",
	})

	constants.ConnectionFactories["foo"] = conf.ConnectionFactory{
		Name:           "foo",
		TranslatorName: "bar",
	}

	project, err := GenerateVdbPom(&vdb, dsInfo, false, false)
	assert.Nil(t, err)
	assert.True(t, hasDependency(project, "org.postgresql", "postgresql"))
	assert.True(t, hasDependency(project, "org.teiid", "teiid-spring-boot-starter"))
	assert.True(t, hasDependency(project, "org.springframework.boot", "spring-boot-starter-actuator"))
	assert.True(t, hasDependency(project, "io.opentracing.contrib", "opentracing-spring-jaeger-web-starter"))
	assert.True(t, hasDependency(project, "org.teiid", "spring-odata"))
	assert.True(t, hasDependency(project, "me.snowdrop", "narayana-spring-boot-starter"))
}
