package cachestore

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

import (
	"context"
	"errors"

	infinispan "github.com/infinispan/infinispan-operator/pkg/apis/infinispan/v1"
	ispnClient "github.com/infinispan/infinispan-operator/pkg/generated/clientset/versioned/typed/infinispan/v1"
	teiidclient "github.com/teiid/teiid-operator/pkg/client"
	"github.com/teiid/teiid-operator/pkg/util/envvar"
	"github.com/teiid/teiid-operator/pkg/util/kubernetes"
	"github.com/teiid/teiid-operator/pkg/util/logs"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var log = logs.GetLogger("cachestore")

const (
	// InfinispanOperatorName is the Infinispan Operator default name
	InfinispanOperatorName = "infinispan-operator"
	infinispanServerGroup  = "infinispan.org"
	defaultInfinispanPort  = 11222
)

// InfinispanDetails --
type InfinispanDetails struct {
	Name             string `yaml:"name,omitempty"`
	NameSpace        string `yaml:"namespace,omitempty"`
	User             string `yaml:"username,omitempty"`
	Password         string `yaml:"password,omitempty"`
	URL              string `yaml:"url,omitempty"`
	CreateIfNotFound bool   `yaml:"create,omitempty"`
	Replicas         int32  `yaml:"replicas,omitempty"`
}

// ConfigurationExists -- Check to see if the configuration is supplied for a cache store
func configurationExists(vdbName string, vdbNamespace string, client k8sclient.Reader) (*corev1.Secret, bool) {
	ctx := context.TODO()
	secret, err := kubernetes.GetSecret(ctx, client, vdbName+"-cache-store", vdbNamespace)
	if err != nil {
		secret, err = kubernetes.GetSecret(ctx, client, "teiid-cache-store", vdbNamespace)
		if err != nil {
			return nil, false
		}
	}
	return secret, true
}

// Exists -- check to so if the Infinispan CacheStore exists
func Exists(vdbName string, vdbNamespace string, client k8sclient.Reader, ispnClient *ispnClient.InfinispanV1Client) bool {
	ctx := context.TODO()
	ispnSecret, exists := configurationExists(vdbName, vdbNamespace, client)
	if !exists {
		return false
	}
	details := readInfinispanDetails(*ispnSecret)
	return hasInfinispan(ctx, ispnClient, details.Name, details.NameSpace)
}

// Credentials --
func Credentials(vdbName string, vdbNamespace string, client k8sclient.Reader) (*InfinispanDetails, error) {
	ispnSecret, exists := configurationExists(vdbName, vdbNamespace, client)
	if !exists {
		return nil, errors.New("Failed to find configuration for the Cache Store")
	}
	details := readInfinispanDetails(*ispnSecret)
	return &details, nil
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
func hasInfinispan(context context.Context, client *ispnClient.InfinispanV1Client, name string, namespace string) bool {
	_, err := client.Infinispans(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return false
	}
	log.Info("Found Infinispan store ", name, " in namespace ", namespace)
	return true
}

// IsInfinispanCRDAvailable checks whether Infinispan CRD is available or not
func IsInfinispanCRDAvailable(cli teiidclient.Client) bool {
	return kubernetes.HasServerGroup(cli, infinispanServerGroup)
}

// IsInfinispanOperatorAvailable verify if Infinispan Operator is running in the given namespace and the CRD is available
func IsInfinispanOperatorAvailable(cli teiidclient.Client, namespace string) (bool, error) {
	log.Debugf("Checking if Infinispan Operator is available in the namespace %s", namespace)
	// first check for CRD
	if IsInfinispanCRDAvailable(cli) {
		log.Debugf("Infinispan CRDs available. Checking if Infinispan Operator is deployed in the namespace %s", namespace)
		// then check if there's an Infinispan Operator deployed
		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: InfinispanOperatorName}}
		exists := false
		var err error
		if exists, err = kubernetes.Resource(cli).Fetch(deployment); err != nil {
			return false, nil
		}
		if exists {
			log.Debugf("Infinispan Operator is available in the namespace %s", namespace)
			return true, nil
		}
	} else {
		log.Debug("Couldn't find Infinispan CRDs")
	}
	log.Debugf("Looks like Infinispan Operator is not available in the namespace %s", namespace)
	return false, nil
}

// NewInfinispanResource --
func NewInfinispanResource(namespace string, name string, secretName string, relicas int32) infinispan.Infinispan {
	infinispanRes := infinispan.Infinispan{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: infinispan.InfinispanSpec{
			Replicas: relicas,
			// ignoring generating secrets for now: https://github.com/infinispan/infinispan-operator/issues/211
			Security: infinispan.InfinispanSecurity{
				EndpointSecretName: secretName,
			},
			Service: infinispan.InfinispanServiceSpec{
				Type: infinispan.ServiceTypeDataGrid,
			},
		},
	}
	return infinispanRes
}
