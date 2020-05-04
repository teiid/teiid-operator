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

	"github.com/teiid/teiid-operator/pkg/controller/virtualdatabase/constants"
	"github.com/teiid/teiid-operator/pkg/util/envvar"
	"github.com/teiid/teiid-operator/pkg/util/vdbutil"
	corev1 "k8s.io/api/core/v1"

	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
)

func convert2SpringProperties(sourcesConfigured []v1alpha1.DataSourceObject, sourcesFromDdl []vdbutil.DatasourceInfo) ([]corev1.EnvVar, error) {
	envs := make([]corev1.EnvVar, 0)

	for _, source := range sourcesFromDdl {
		prefix := "SPRING_DATASOURCE"

		datasourceName := sanitizeName(removeDash(strings.ToLower(source.Name)))
		configuredSource, err := findConfiguredProperties(source.Name, sourcesConfigured)
		if err != nil {
			log.Info(err)
			continue
		}

		// Make sure we do not have incompatible names
		if strings.Contains(configuredSource.Name, " ") {
			return nil, errors.New("Configured Datasource " + configuredSource.Name + " has spaces, which is not allowed")
		}

		if c, ok := constants.ConnectionFactories[strings.ToLower(source.Type)]; ok {
			prefix = removeDash(strings.ToLower(c.SpringBootPropertyPrefix))
		} else {
			// Custom translators must map to this property prefix
			prefix = "spring.teiid.data." + removeDash(strings.ToLower(source.Type))
		}

		// covert properties
		for _, p := range configuredSource.Properties {
			if strings.Contains(p.Name, " ") {
				return nil, errors.New("Datasource " + configuredSource.Name + " has a Property " + p.Name + " which has spaces in its name, which is not allowed")
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

func findConfiguredProperties(name string, configured []v1alpha1.DataSourceObject) (v1alpha1.DataSourceObject, error) {
	for _, ds := range configured {
		if ds.Name == name {
			return ds, nil
		}
	}
	return v1alpha1.DataSourceObject{}, errors.New("Configuration for the Data Source " + name + " not found in DataSources, one can define the configuration also using the ENV properties otherwise the deployment will fail")
}

func envReady(v string) string {
	str := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(v, ".", "_"), "-", "_"))
	return str
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

func removeDash(str string) string {
	return strings.ReplaceAll(str, "-", "")
}
