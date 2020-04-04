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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func TestEnv(t *testing.T) {
	assert.NotNil(t, envReady("foo.amazon-s3"))
	assert.Equal(t, "FOO_AMAZON_S3", envReady("foo.amazon-s3"))
}

func TestSpringProperties(t *testing.T) {
	source := corev1.EnvVarSource{
		ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
			Key: "foo",
			LocalObjectReference: corev1.LocalObjectReference{
				Name: "map-name",
			},
		},
	}

	datasources := []v1alpha1.DataSourceObject{
		{
			Name: "dg",
			Type: "infinispan-hotrod",
			Properties: []corev1.EnvVar{
				{
					Name:  "url",
					Value: "localhost:11222",
				},
				{
					Name:  "importer.ProtobufName",
					Value: "accounts.proto",
				},
			},
		},
		{
			Name: "sampledb",
			Type: "postgresql",
			Properties: []corev1.EnvVar{
				{
					Name:  "jdbc-url",
					Value: "jdbc:postgresql://localhost:5432/sampledb",
				},
				{
					Name:      "password",
					ValueFrom: &source,
				},
			},
		},
	}

	envs, err := convert2SpringProperties(datasources)
	assert.NotNil(t, envs)
	assert.Nil(t, err)

	expected := []corev1.EnvVar{
		{
			Name:  "SPRING_TEIID_DATA_INFINISPAN_DG_URL",
			Value: "localhost:11222",
		},
		{
			Name:  "SPRING_TEIID_DATA_INFINISPAN_DG_IMPORTER_PROTOBUF_NAME",
			Value: "accounts.proto",
		},
		{
			Name:  "SPRING_DATASOURCE_SAMPLEDB_JDBC_URL",
			Value: "jdbc:postgresql://localhost:5432/sampledb",
		},
		{
			Name:      "SPRING_DATASOURCE_SAMPLEDB_PASSWORD",
			ValueFrom: &source,
		},
	}
	assert.Equal(t, expected, envs)
}

func TestUpperCase(t *testing.T) {
	assert.Equal(t, []string{"foo", "Bar"}, tokenizeAtUpperCase("fooBar"))
	assert.Equal(t, []string{"foo", "B", "A", "R"}, tokenizeAtUpperCase("fooBAR"))
	assert.Equal(t, []string{"Foo", "B", "A", "R"}, tokenizeAtUpperCase("FooBAR"))
}

func TestSanitizeName(t *testing.T) {
	assert.Equal(t, "foo-bar", sanitizeName("fooBar"))
	assert.Equal(t, "foo-bar", sanitizeName("fooBAR"))
	assert.Equal(t, "foo-bar", sanitizeName("FooBAR"))
	assert.Equal(t, "foo-bar", sanitizeName("foo-bar"))
	assert.Equal(t, "foobar", sanitizeName("foobar"))
	assert.Equal(t, "foo.bar", sanitizeName("foo.bar"))
	assert.Equal(t, "foo.bar", sanitizeName("foo.Bar"))
	assert.Equal(t, "foo-bar", sanitizeName("foo-Bar"))
	assert.Equal(t, "foo-bar", sanitizeName("Foo-Bar"))
}

func TestSoapProperties(t *testing.T) {
	source := corev1.EnvVarSource{
		ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
			Key: "foo",
			LocalObjectReference: corev1.LocalObjectReference{
				Name: "map-name",
			},
		},
	}

	datasources := []v1alpha1.DataSourceObject{
		{
			Name: "soapCountry",
			Type: "soap",
			Properties: []corev1.EnvVar{
				{
					Name:  "wsdl",
					Value: "http://www.oorsprong.org/websamples.countryinfo/CountryInfoService.wso?WSDL",
				},
				{
					Name:  "namespaceUri",
					Value: "http://www.oorsprong.org/websamples.countryinfo",
				},
				{
					Name:  "serviceName",
					Value: "CountryInfoService",
				},
				{
					Name:  "endPointName",
					Value: "CountryInfoServiceSoap12",
				},
			},
		},
		{
			Name: "sampledb",
			Type: "postgresql",
			Properties: []corev1.EnvVar{
				{
					Name:  "jdbc-url",
					Value: "jdbc:postgresql://localhost:5432/sampledb",
				},
				{
					Name:      "password",
					ValueFrom: &source,
				},
			},
		},
	}

	envs, err := convert2SpringProperties(datasources)
	assert.NotNil(t, envs)
	assert.Nil(t, err)

	expected := []corev1.EnvVar{
		{
			Name:  "SPRING_TEIID_DATA_SOAP_SOAP_COUNTRY_WSDL",
			Value: "http://www.oorsprong.org/websamples.countryinfo/CountryInfoService.wso?WSDL",
		},
		{
			Name:  "SPRING_TEIID_DATA_SOAP_SOAP_COUNTRY_NAMESPACE_URI",
			Value: "http://www.oorsprong.org/websamples.countryinfo",
		},
		{
			Name:  "SPRING_TEIID_DATA_SOAP_SOAP_COUNTRY_SERVICE_NAME",
			Value: "CountryInfoService",
		},
		{
			Name:  "SPRING_TEIID_DATA_SOAP_SOAP_COUNTRY_END_POINT_NAME",
			Value: "CountryInfoServiceSoap12",
		},
		{
			Name:  "SPRING_DATASOURCE_SAMPLEDB_JDBC_URL",
			Value: "jdbc:postgresql://localhost:5432/sampledb",
		},
		{
			Name:      "SPRING_DATASOURCE_SAMPLEDB_PASSWORD",
			ValueFrom: &source,
		},
	}
	assert.Equal(t, expected, envs)
}
