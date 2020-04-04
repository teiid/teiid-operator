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
	"errors"
	"strings"
	"unicode"

	"github.com/teiid/teiid-operator/pkg/util/envvar"
	corev1 "k8s.io/api/core/v1"

	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
)

// DeploymentEnvironments --
func DeploymentEnvironments(vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) ([]corev1.EnvVar, error) {
	dataSourceConfig, err := convert2SpringProperties(vdb.Spec.DataSources)
	if err != nil {
		return nil, err
	}
	return envvar.Combine(r.vdbContext.Env, dataSourceConfig), nil
}

func convert2SpringProperties(datasources []v1alpha1.DataSourceObject) ([]corev1.EnvVar, error) {
	envs := make([]corev1.EnvVar, 0)

	dsConfig := make(map[string]string)
	dsConfig["salesforce"] = "spring.teiid.data.salesforce"
	dsConfig["google-spreadsheet"] = "spring.teiid.data.google.sheets"
	dsConfig["amazon-s3"] = "spring.teiid.data.amazon-s3"
	dsConfig["infinispan-hotrod"] = "spring.teiid.data.infinispan"
	dsConfig["mongodb"] = "spring.teiid.data.mongodb"
	dsConfig["soap"] = "spring.teiid.data.soap"
	dsConfig["rest"] = "spring.teiid.rest"
	dsConfig["odata"] = "spring.teiid.rest"
	dsConfig["odata4"] = "spring.teiid.rest"
	dsConfig["openapi"] = "spring.teiid.rest"
	dsConfig["sap-gateway"] = "spring.teiid.rest"
	dsConfig["excel"] = "spring.teiid.file"
	dsConfig["file"] = "spring.teiid.file"

	for _, v := range datasources {
		prefix := "SPRING_DATASOURCE"

		if strings.Contains(v.Name, " ") || strings.Contains(v.Type, " ") {
			return nil, errors.New("Datasource " + v.Name + " or its Type " + v.Type + " has spaces, which is not allowed")
		}

		if c, ok := dsConfig[strings.ToLower(v.Type)]; ok {
			prefix = c
		}
		datasourceName := sanitizeName(v.Name)
		// covert properties
		for _, p := range v.Properties {
			if strings.Contains(p.Name, " ") {
				return nil, errors.New("Datasource " + v.Name + " has a Property " + p.Name + " which has spaces in its name, which is not allowed")
			}
			propertyName := sanitizeName(p.Name)
			if p.Value != "" {
				envvar.SetVal(&envs, envReady(prefix+"_"+datasourceName+"_"+propertyName), p.Value)
			}
			if p.ValueFrom != nil {
				envvar.SetValueFrom(&envs, envReady(prefix+"_"+datasourceName+"_"+propertyName), p.ValueFrom)
			}
		}
	}
	return envs, nil
}

func envReady(v string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(v, ".", "_"), "-", "_"))
}

func sanitizeName(v string) string {
	// splits at every captital letter
	tokens := tokenizeAtUpperCase(v)

	var modified string
	for i, str := range tokens {
		if i == 0 {
			modified = strings.ToLower(str)
		} else {
			// if previous token with single char, this like name "fooDB", where above will split at
			// D and B. we only want capture D
			prev := tokens[i-1]
			if len(prev) == 1 || (len(prev) > 1 && prev[len(prev)-1:] == ".") || (len(prev) > 1 && prev[len(prev)-1:] == "-") {
				modified = modified + strings.ToLower(str)
			} else {
				modified = modified + "-" + strings.ToLower(str)
			}
		}
	}
	return modified
}

func tokenizeAtUpperCase(str string) []string {
	var words []string
	l := 0
	for s := str; s != ""; s = s[l:] {
		l = strings.IndexFunc(s[1:], unicode.IsUpper) + 1
		if l <= 0 {
			l = len(s)
		}
		words = append(words, s[:l])
	}
	return words
}
