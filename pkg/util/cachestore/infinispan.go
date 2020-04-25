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
package cachestore

import (
	"context"
	"strconv"

	ispn "github.com/infinispan/infinispan-operator/pkg/generated/clientset/versioned/typed/infinispan/v1"
	"github.com/teiid/teiid-operator/pkg/util/envvar"
	"github.com/teiid/teiid-operator/pkg/util/kubernetes"
	"github.com/teiid/teiid-operator/pkg/util/logs"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var log = logs.GetLogger("cachestore")

// InfinispanDetails --
type InfinispanDetails struct {
	Name              string `yaml:"name,omitempty"`
	NameSpace         string `yaml:"namespace,omitempty"`
	CreateIfNotExists bool   `yaml:"create,omitempty"`
	User              string `yaml:"user,omitempty"`
	Password          string `yaml:"password,omitempty"`
	URL               string `yaml:"url,omitempty"`
}

// Exists -- check to so if the Infinispan CacheStore exists
func Exists(vdbName string, vdbNamespace string, client k8sclient.Reader, ispnClient *ispn.InfinispanV1Client) bool {
	ctx := context.TODO()
	ispnSecret, err := kubernetes.GetSecret(ctx, client, vdbName+"-cache-store", vdbNamespace)
	if err != nil {
		ispnSecret, err = kubernetes.GetSecret(ctx, client, "teiid-cache-store", vdbNamespace)
		if err != nil {
			return false
		}
	}
	details := readInfinispanDetails(*ispnSecret)
	return hasInfinispan(ctx, ispnClient, details.Name, details.NameSpace)
}

// Credentials --
func Credentials(vdbName string, vdbNamespace string, client k8sclient.Reader) (InfinispanDetails, error) {
	ctx := context.TODO()
	ispnSecret, err := kubernetes.GetSecret(ctx, client, vdbName+"-cache-store", vdbNamespace)
	if err != nil {
		ispnSecret, err = kubernetes.GetSecret(ctx, client, "teiid-cache-store", vdbNamespace)
		if err != nil {
			return InfinispanDetails{}, err
		}
	}
	details := readInfinispanDetails(*ispnSecret)
	return details, nil
}

func readInfinispanDetails(secret v1.Secret) InfinispanDetails {
	details := InfinispanDetails{}

	if secret.Data["name"] != nil {
		details.Name = string(secret.Data["name"])
	} else {
		details.Name = secret.StringData["name"]
	}

	if secret.Data["namespace"] != nil {
		details.NameSpace = string(secret.Data["namespace"])
	} else {
		details.NameSpace = secret.StringData["namespace"]
	}

	if secret.Data["username"] != nil {
		details.User = string(secret.Data["username"])
	} else {
		details.User = secret.StringData["username"]
	}

	if secret.Data["password"] != nil {
		details.Password = string(secret.Data["password"])
	} else {
		details.Password = secret.StringData["password"]
	}

	if secret.Data["url"] != nil {
		details.URL = string(secret.Data["url"])
	} else {
		details.URL = secret.StringData["url"]
	}

	var err error
	if secret.Data["create"] != nil {
		details.CreateIfNotExists, err = strconv.ParseBool(string(secret.Data["create"]))
	} else {
		details.CreateIfNotExists, err = strconv.ParseBool(secret.StringData["create"])
	}

	if err != nil {
		details.CreateIfNotExists = false
	}

	return details
}

// CredentialsAsEnv --
func CredentialsAsEnv(vdbName string, vdbNamespace string, client k8sclient.Reader) []corev1.EnvVar {
	ctx := context.TODO()
	envs := make([]corev1.EnvVar, 0)

	secretName := vdbName + "-cache-store"
	if !kubernetes.HasSecret(ctx, client, vdbName+"-cache-store", vdbNamespace) {
		secretName = "teiid-cache-store"
	}

	envvar.SetVar(&envs, corev1.EnvVar{
		Name: "SPRING_TEIID_DATA_INFINISPANHOTROD_CACHESTORE_USERNAME",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secretName,
				},
				Key: "username",
			},
		},
	})

	envvar.SetVar(&envs, corev1.EnvVar{
		Name: "SPRING_TEIID_DATA_INFINISPANHOTROD_CACHESTORE_PASSWORD",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secretName,
				},
				Key: "password",
			},
		},
	})

	envvar.SetVar(&envs, corev1.EnvVar{
		Name: "SPRING_TEIID_DATA_INFINISPANHOTROD_CACHESTORE_URL",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secretName,
				},
				Key: "url",
			},
		},
	})

	envvar.SetVar(&envs, corev1.EnvVar{
		Name:  "SPRING_TEIID_DATA_INFINISPANHOTROD_CACHESTORE_TRANSACTIONMODE",
		Value: "NON_XA",
	})

	envvar.SetVar(&envs, corev1.EnvVar{
		Name:  "SPRING_TEIID_DATA_INFINISPANHOTROD_CACHESTORE_CACHENAME",
		Value: vdbName,
	})

	return envs
}

// HasInfinispan --
func hasInfinispan(context context.Context, client *ispn.InfinispanV1Client, name string, namespace string) bool {
	_, err := client.Infinispans(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return false
	}
	log.Info("Found Infinispan store ", name, " in namespace ", namespace)
	return true
}
