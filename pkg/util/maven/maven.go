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
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/teiid/teiid-operator/pkg/util"
	"github.com/teiid/teiid-operator/pkg/util/logs"
)

// Log --
var Log = logs.GetLogger("maven")

// EncodeXML generate a pom.xml file from the given project definition
func EncodeXML(content interface{}) (string, error) {
	w := &bytes.Buffer{}
	w.WriteString(xml.Header)

	e := xml.NewEncoder(w)
	e.Indent("", "  ")

	err := e.Encode(content)
	if err != nil {
		return "", err
	}

	return w.String(), nil
}

// CreateStructure --
func CreateStructure(buildDir string, project Project) error {
	Log.Infof("write project: %+v", project)

	pom, err := EncodeXML(project)
	if err != nil {
		return err
	}

	err = util.WriteFileWithContent(buildDir, "pom.xml", pom)
	if err != nil {
		return err
	}

	return nil
}

// Run --
func Run(buildDir string, args ...string) error {
	mvnCmd := "mvn"
	if c, ok := os.LookupEnv("MAVEN_CMD"); ok {
		mvnCmd = c
	}

	args = append(args, "--batch-mode")

	cmd := exec.Command(mvnCmd, args...)
	cmd.Dir = buildDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	Log.Infof("execute: %s", strings.Join(cmd.Args, " "))

	return cmd.Run()
}

// ParseGAV decode a maven artifact id to a dependency definition.
//
// The artifact id is in the form of:
//
//     <groupId>:<artifactId>[:<packagingType>[:<classifier>]]:(<version>|'?')
//
func ParseGAV(gav string) (Dependency, error) {
	// <groupId>:<artifactId>[:<packagingType>[:<classifier>]]:(<version>|'?')
	dep := Dependency{}
	rex := regexp.MustCompile("([^: ]+):([^: ]+)(:([^: ]*)(:([^: ]+))?)?(:([^: ]+))?")
	res := rex.FindStringSubmatch(gav)

	fmt.Println(res, len(res))

	if res == nil || len(res) < 9 {
		return Dependency{}, errors.New("GAV must match <groupId>:<artifactId>[:<packagingType>[:<classifier>]]:(<version>|'?')")
	}

	dep.GroupID = res[1]
	dep.ArtifactID = res[2]
	dep.Type = "jar"

	cnt := strings.Count(gav, ":")
	switch cnt {
	case 2:
		dep.Version = res[4]
	case 3:
		dep.Type = res[4]
		dep.Version = res[6]
	default:
		dep.Type = res[4]
		dep.Classifier = res[6]
		dep.Version = res[8]
	}

	return dep, nil
}

// DownloadDependency --
func DownloadDependency(d Dependency, filepath string, mavenRepos map[string]string) (string, error) {
	// check if this is SNAPSHOT release, if yes build the version timestamp
	snapshot := strings.Contains(d.Version, "SNAPSHOT")
	versionTimestamped := ""
	if snapshot {
		versionTimestamped = getSnapshotVersion(d, mavenRepos)
		if versionTimestamped == "" {
			return "", errors.New("Failed to download the SNAPSHOT version of the artifact")
		}
	}

	parts := append(strings.Split(d.GroupID, "."), d.ArtifactID)
	artifactName := strings.Join(append(parts, d.Version), "/") + "/" + fileName(d, versionTimestamped)

	for _, v := range mavenRepos {
		url := v + artifactName
		if !strings.HasSuffix(v, "/") {
			url = v + "/" + artifactName
		}
		err := downloadFile(filepath, url)
		if err == nil {
			return filepath, nil
		}
	}
	return "", errors.New("Failed to download the artifact from configured maven repositories")
}

func getSnapshotVersion(a Dependency, mavenRepos map[string]string) string {
	for _, v := range mavenRepos {
		url := v
		if !strings.HasSuffix(v, "/") {
			url = url + "/"
		}
		group := strings.Replace(a.GroupID, ".", "/", -1)
		url = url + group + "/" + a.ArtifactID + "/" + a.Version + "/maven-metadata.xml"
		err := downloadFile("/tmp/maven-metadata.xml", url)
		if err == nil {
			xmlFile, err := ioutil.ReadFile("/tmp/maven-metadata.xml")
			if err == nil {
				m, err := parseMavenMetadata(xmlFile)
				if err == nil {
					for _, sv := range m.Versioning.SnapshotVersions {
						if sv.Extension == a.Type {
							return sv.Value
						}
					}
				}
			}
		}
	}
	return ""
}

func fileName(a Dependency, versionTimestamped string) string {
	ext := "jar"
	if a.Type != "" {
		ext = a.Type
	}
	if a.Classifier != "" {
		if versionTimestamped != "" {
			return fmt.Sprintf("%s-%s-%s.%s", a.ArtifactID, versionTimestamped, a.Classifier, ext)
		}
		return fmt.Sprintf("%s-%s-%s.%s", a.ArtifactID, a.Version, a.Classifier, ext)
	}
	if versionTimestamped != "" {
		return fmt.Sprintf("%s-%s.%s", a.ArtifactID, versionTimestamped, ext)
	}
	return fmt.Sprintf("%s-%s.%s", a.ArtifactID, a.Version, ext)
}

func downloadFile(filepath string, url string) (err error) {
	log := logs.GetLogger("maven")

	log.Info("downloading artifact from ", url, " writing to ", filepath)
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		log.Error("Failed download of url ", url)
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
