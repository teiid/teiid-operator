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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/teiid/teiid-operator/pkg/util/envvar"
	corev1 "k8s.io/api/core/v1"
)

func TestHandleClusterProxySettings(t *testing.T) {
	vars := []corev1.EnvVar{
		{
			Name:  "MyEnv",
			Value: "MyValue",
		},
	}

	vars = handleClusterProxySettings(vars)

	ev := envvar.Get(vars, "MyEnv")
	assert.NotNil(t, ev)
	assert.Equal(t, "MyValue", ev.Value)
	assert.Nil(t, ev.ValueFrom)
	assert.Equal(t, 1, len(vars))

	envvar.SetVal(&vars, "HTTPS_PROXY", "foobar")
	vars = handleClusterProxySettings(vars)

	assert.Equal(t, 2, len(vars))
	ev = envvar.Get(vars, "HTTPS_PROXY")
	assert.NotNil(t, ev)
	assert.Equal(t, "foobar", ev.Value)
}

func TestHandleClusterProxySettingsFromCluster(t *testing.T) {
	vars := []corev1.EnvVar{
		{
			Name:  "MyEnv",
			Value: "MyValue",
		},
	}

	err := os.Setenv("HTTP_PROXY", "foobar")
	assert.Nil(t, err)
	vars = handleClusterProxySettings(vars)
	assert.Equal(t, 2, len(vars))

	ev := envvar.Get(vars, "MyEnv")
	assert.NotNil(t, ev)
	assert.Equal(t, "MyValue", ev.Value)
	assert.Nil(t, ev.ValueFrom)

	ev = envvar.Get(vars, "HTTP_PROXY")
	assert.NotNil(t, ev)
	assert.Equal(t, "foobar", ev.Value)
}

func TestMatchLabels(t *testing.T) {
	labels := matchLabels("foo")
	assert.Equal(t, 3, len(labels))
	assert.Equal(t, "foo", labels["app"])
	assert.Equal(t, "foo", labels["teiid.io/VirtualDatabase"])
	assert.Equal(t, "VirtualDatabase", labels["teiid.io/type"])
}
