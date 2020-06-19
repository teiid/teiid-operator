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

package maven

import (
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/assert"
)

const expectedPom = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" ` +
	`xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">
  <modelVersion>4.0.0</modelVersion>
  <groupId>org.apache.camel.k.integration</groupId>
  <artifactId>camel-k-integration</artifactId>
  <version>1.0.0</version>
  <packaging>jar</packaging>
  <parent>
    <groupId>org.basepom</groupId>
    <artifactId>basepom-oss</artifactId>
    <version>30</version>
  </parent>
  <dependencyManagement>
    <dependencies>
      <dependency>
        <groupId>org.apache.camel</groupId>
        <artifactId>camel-bom</artifactId>
        <version>2.22.1</version>
        <type>pom</type>
        <scope>import</scope>
      </dependency>
    </dependencies>
  </dependencyManagement>
  <dependencies>
    <dependency>
      <groupId>org.apache.camel.k</groupId>
      <artifactId>camel-k-runtime-jvm</artifactId>
      <version>1.0.0</version>
    </dependency>
  </dependencies>
  <repositories>
    <repository>
      <id>central</id>
      <url>https://repo.maven.apache.org/maven2</url>
      <snapshots>
        <enabled>false</enabled>
      </snapshots>
      <releases>
        <enabled>true</enabled>
        <updatePolicy>never</updatePolicy>
      </releases>
    </repository>
  </repositories>
  <pluginRepositories>
    <pluginRepository>
      <id>central</id>
      <url>https://repo.maven.apache.org/maven2</url>
      <snapshots>
        <enabled>false</enabled>
      </snapshots>
      <releases>
        <enabled>true</enabled>
        <updatePolicy>never</updatePolicy>
      </releases>
    </pluginRepository>
  </pluginRepositories>
  <build>
    <plugins></plugins>
  </build>
</project>`

func TestPomGeneration(t *testing.T) {
	project := Project{
		XMLName:           xml.Name{Local: "project"},
		XMLNs:             "http://maven.apache.org/POM/4.0.0",
		XMLNsXsi:          "http://www.w3.org/2001/XMLSchema-instance",
		XsiSchemaLocation: "http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd",
		ModelVersion:      "4.0.0",
		GroupID:           "org.apache.camel.k.integration",
		ArtifactID:        "camel-k-integration",
		Version:           "1.0.0",
		Packaging:         "jar",
		Parent: Parent{
			GroupID:    "org.basepom",
			ArtifactID: "basepom-oss",
			Version:    "30",
		},
		DependencyManagement: DependencyManagement{
			Dependencies: []Dependency{
				{
					GroupID:    "org.apache.camel",
					ArtifactID: "camel-bom",
					Version:    "2.22.1",
					Type:       "pom",
					Scope:      "import",
				},
			},
		},
		Dependencies: []Dependency{
			{
				GroupID:    "org.apache.camel.k",
				ArtifactID: "camel-k-runtime-jvm",
				Version:    "1.0.0",
			},
		},
		Repositories: []Repository{
			{
				ID:  "central",
				URL: "https://repo.maven.apache.org/maven2",
				Snapshots: RepositoryPolicy{
					Enabled: false,
				},
				Releases: RepositoryPolicy{
					Enabled:      true,
					UpdatePolicy: "never",
				},
			},
		},
		PluginRepositories: []Repository{
			{
				ID:  "central",
				URL: "https://repo.maven.apache.org/maven2",
				Snapshots: RepositoryPolicy{
					Enabled: false,
				},
				Releases: RepositoryPolicy{
					Enabled:      true,
					UpdatePolicy: "never",
				},
			},
		},
	}

	pom, err := EncodeXML(project)

	assert.Nil(t, err)
	assert.NotNil(t, pom)

	assert.Equal(t, expectedPom, pom)
}

func TestParseSimpleGAV(t *testing.T) {
	dep, err := ParseGAV("org.apache.camel:camel-core:2.21.1")

	assert.Nil(t, err)
	assert.Equal(t, dep.GroupID, "org.apache.camel")
	assert.Equal(t, dep.ArtifactID, "camel-core")
	assert.Equal(t, dep.Version, "2.21.1")
	assert.Equal(t, dep.Type, "jar")
	assert.Equal(t, dep.Classifier, "")
}

func TestParseGAVWithType(t *testing.T) {
	dep, err := ParseGAV("org.apache.camel:camel-core:war:2.21.1")

	assert.Nil(t, err)
	assert.Equal(t, dep.GroupID, "org.apache.camel")
	assert.Equal(t, dep.ArtifactID, "camel-core")
	assert.Equal(t, dep.Version, "2.21.1")
	assert.Equal(t, dep.Type, "war")
	assert.Equal(t, dep.Classifier, "")
}

func TestParseGAVWithClassifierAndType(t *testing.T) {
	dep, err := ParseGAV("org.apache.camel:camel-core:war:test:2.21.1")

	assert.Nil(t, err)
	assert.Equal(t, dep.GroupID, "org.apache.camel")
	assert.Equal(t, dep.ArtifactID, "camel-core")
	assert.Equal(t, dep.Version, "2.21.1")
	assert.Equal(t, dep.Type, "war")
	assert.Equal(t, dep.Classifier, "test")
}

func TestParseGAVMvnNoVersion(t *testing.T) {
	dep, err := ParseGAV("mvn:org.apache.camel/camel-core")

	assert.Nil(t, err)
	assert.Equal(t, dep.GroupID, "mvn")
	assert.Equal(t, dep.ArtifactID, "org.apache.camel/camel-core")
}

func TestParseGAVErrorNoColumn(t *testing.T) {
	dep, err := ParseGAV("org.apache.camel.k.camel-k-runtime-noop-0.2.1-SNAPSHOT.jar")

	assert.EqualError(t, err, "GAV must match <groupId>:<artifactId>[:<packagingType>[:<classifier>]]:(<version>|'?')")
	assert.Equal(t, Dependency{}, dep)
}

func TestNewRepository(t *testing.T) {
	r := NewRepository("http://nexus/public")
	assert.Equal(t, "", r.ID)
	assert.Equal(t, "http://nexus/public", r.URL)
	assert.True(t, r.Releases.Enabled)
	assert.False(t, r.Snapshots.Enabled)
}

func TestNewRepositoryWithSnapshots(t *testing.T) {
	r := NewRepository("http://nexus/public@snapshots")
	assert.Equal(t, "", r.ID)
	assert.Equal(t, "http://nexus/public", r.URL)
	assert.True(t, r.Releases.Enabled)
	assert.True(t, r.Snapshots.Enabled)
}

func TestNewRepositoryWithSnapshotsAndID(t *testing.T) {
	r := NewRepository("http://nexus/public@snapshots@id=test")
	assert.Equal(t, "test", r.ID)
	assert.Equal(t, "http://nexus/public", r.URL)
	assert.True(t, r.Releases.Enabled)
	assert.True(t, r.Snapshots.Enabled)
}

func TestNewRepositoryWithID(t *testing.T) {
	r := NewRepository("http://nexus/public@id=test")
	assert.Equal(t, "test", r.ID)
	assert.Equal(t, "http://nexus/public", r.URL)
	assert.True(t, r.Releases.Enabled)
	assert.False(t, r.Snapshots.Enabled)
}

func TestMetadata(t *testing.T) {

	contents := `<metadata modelVersion="1.1.0">
	<groupId>org.teiid.examples</groupId>
	<artifactId>postgresql-maven</artifactId>
	<version>1.0-SNAPSHOT</version>
	<versioning>
	  <snapshot>
	     <timestamp>20200506.095522</timestamp>
	     <buildNumber>1</buildNumber>
	  </snapshot>
	  <lastUpdated>20200506095522</lastUpdated>
	  <snapshotVersions>
	     <snapshotVersion>
	       <extension>vdb</extension>
	       <value>1.0-20200506.095522-1</value>
	       <updated>20200506095522</updated>
	     </snapshotVersion>
	     <snapshotVersion>
	        <extension>pom</extension>
	        <value>1.0-20200506.095522-1</value>
	        <updated>20200506095522</updated>
	     </snapshotVersion>
	   </snapshotVersions>
	</versioning>
	</metadata>`

	m, err := parseMavenMetadata([]byte(contents))
	assert.Nil(t, err)
	assert.NotNil(t, m)

	assert.Equal(t, "org.teiid.examples", m.GroupID)
	assert.Equal(t, "postgresql-maven", m.ArtifactID)
	assert.Equal(t, "1.0-SNAPSHOT", m.Version)
	assert.Equal(t, 2, len(m.Versioning.SnapshotVersions))
	assert.Equal(t, "vdb", m.Versioning.SnapshotVersions[0].Extension)
	assert.Equal(t, "20200506095522", m.Versioning.SnapshotVersions[0].Updated)
	assert.Equal(t, "1.0-20200506.095522-1", m.Versioning.SnapshotVersions[0].Value)
}
