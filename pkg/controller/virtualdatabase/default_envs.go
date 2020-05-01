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
	"github.com/teiid/teiid-operator/pkg/util/proxy"
	corev1 "k8s.io/api/core/v1"
)

// GetDefaultEnvs --
func getDefaultEnvs(userDefined []corev1.EnvVar) []corev1.EnvVar {
	// Passing down cluster proxy config to Operands
	envs := envvar.Clone(userDefined)
	allEnvs, properties := proxy.HTTPSettings(envs)
	var javaProperties string
	for k, v := range properties {
		javaProperties = javaProperties + "-D" + k + "=" + v + " "
	}

	str := strings.Join([]string{
		" ",
		"-Djava.net.preferIPv4Stack=true",
		"-Duser.home=/tmp",
		"-Djava.net.preferIPv4Addresses=true",
		"-Djava.net.useSystemProxies=true",
	}, " ")

	// environment variables
	defaultEnv := []corev1.EnvVar{
		{
			Name:  "JAVA_APP_DIR",
			Value: "/deployments",
		},
		{
			Name:  "JAVA_OPTIONS",
			Value: javaProperties + str,
		},
		{
			Name:  "JAVA_DEBUG",
			Value: "false",
		},
		{
			Name:  "AB_JMX_EXPORTER_CONFIG",
			Value: "/tmp/src/src/main/resources/prometheus-config.yml",
		},
		{
			Name: "NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
	}

	// merge/update env with user defined
	for _, v := range defaultEnv {
		if envvar.Get(allEnvs, v.Name) == nil {
			envvar.SetVar(&allEnvs, v)
		}
	}

	return allEnvs
}

func getDefaultJaegerEnvs(serviceName string) []corev1.EnvVar {
	envs := make([]corev1.EnvVar, 0)
	envvar.SetVar(&envs, corev1.EnvVar{
		Name:  "JAEGER_AGENT_HOST",
		Value: "localhost",
	})
	envvar.SetVar(&envs, corev1.EnvVar{
		Name:  "JAEGER_AGENT_PORT",
		Value: "6831",
	})
	envvar.SetVar(&envs, corev1.EnvVar{
		Name:  "JAEGER_SERVICE_NAME",
		Value: serviceName,
	})
	return envs
}
