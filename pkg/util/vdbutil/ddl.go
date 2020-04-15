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

package vdbutil

import (
	"io/ioutil"

	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/controller/virtualdatabase/constants"
	"github.com/teiid/teiid-operator/pkg/util/logs"
	"github.com/teiid/teiid-operator/pkg/util/maven"
	"github.com/teiid/teiid-operator/pkg/util/zip"
)

var log = logs.GetLogger("vdbutil")

// FetchDdl Get DDL from Custom Resource file, if maven based from Maven file
func FetchDdl(vdb *v1alpha1.VirtualDatabase) (string, error) {
	ddlStr := vdb.Spec.Build.Source.DDL
	if vdb.Spec.Build.Source.Maven != "" {
		str, err := readDdlFromMavenRepo(vdb, "/tmp/teiid.vdb")
		if err != nil {
			log.Error("failed to read VDB from maven ", err)
			return "", err
		}
		ddlStr = str
	}
	return ddlStr, nil
}

func readDdlFromMavenRepo(vdb *v1alpha1.VirtualDatabase, targetName string) (string, error) {
	dep, err := maven.ParseGAV(vdb.Spec.Build.Source.Maven)
	if err != nil {
		return "", err
	}
	mavenRepos := constants.GetMavenRepositories(vdb)
	vdbFile, err := maven.DownloadDependency(dep, targetName, mavenRepos)
	if err != nil {
		return "", err
	}
	files, err := zip.Unzip(vdbFile, "/tmp/"+vdb.ObjectMeta.Name)
	if err != nil {
		return "", err
	}
	log.Info("Maven based VDB file contains files: ", files)
	b, err := ioutil.ReadFile("/tmp/" + vdb.ObjectMeta.Name + "/META-INF/vdb.ddl")
	if err != nil {
		return "", err
	}
	ddl := string(b)
	log.Debug("Read VDB File: " + ddl)
	return ddl, nil
}
