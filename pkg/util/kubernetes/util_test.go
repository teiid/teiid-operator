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
package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestValidateDataSourceNames(t *testing.T) {
	env := []corev1.EnvVar{
		{
			Name:  "FOO_BAR",
			Value: "bar",
		},
	}
	assert.Nil(t, ValidateEnvironmentPropertyNames(env))

	env = []corev1.EnvVar{
		{
			Name:  "FOO.BAR",
			Value: "bar",
		},
	}
	assert.NotNil(t, ValidateEnvironmentPropertyNames(env))

	env = []corev1.EnvVar{
		{
			Name:  "FOObAR",
			Value: "bar",
		},
	}
	assert.NotNil(t, ValidateEnvironmentPropertyNames(env))

	env = []corev1.EnvVar{
		{
			Name:  "FOOêBAR",
			Value: "bar",
		},
	}
	assert.NotNil(t, ValidateEnvironmentPropertyNames(env))
}
