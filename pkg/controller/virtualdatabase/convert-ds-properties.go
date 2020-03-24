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
	"strings"

	"github.com/teiid/teiid-operator/pkg/util/envvar"
	corev1 "k8s.io/api/core/v1"

	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
)

// DeploymentEnvironments --
func DeploymentEnvironments(vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) []corev1.EnvVar {
	dataSourceConfig := convert2SpringProperties(vdb.Spec.DataSources)
	return envvar.Combine(r.vdbContext.Env, dataSourceConfig)
}

func convert2SpringProperties(datasources []v1alpha1.DataSourceObject) []corev1.EnvVar {
	envs := make([]corev1.EnvVar, 0)

	dsConfig := make(map[string]string)
	dsConfig["salesforce"] = "spring.teiid.data.salesforce"
	dsConfig["google-spreadsheet"] = "spring.teiid.data.google.sheets"
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

		if c, ok := dsConfig[strings.ToLower(v.Type)]; ok {
			prefix = c
		}

		// covert properties
		for _, p := range v.Properties {
			if p.Value != "" {
				envvar.SetVal(&envs, envReady(prefix+"_"+v.Name+"_"+p.Name), p.Value)
			}
			if p.ValueFrom != nil {
				envvar.SetValueFrom(&envs, envReady(prefix+"_"+v.Name+"_"+p.Name), p.ValueFrom)
			}
		}
	}
	return envs
}

func envReady(v string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(v, ".", "_"), "-", "_"))
}
