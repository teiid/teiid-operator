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
	"context"
	"fmt"
	"strings"

	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
	teiidclient "github.com/teiid/teiid-operator/pkg/client"
	"github.com/teiid/teiid-operator/pkg/util/logs"
	yaml2 "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var log = logs.GetLogger("kubernetes-util")

// ToJSON --
func ToJSON(value runtime.Object) ([]byte, error) {
	return json.Marshal(value)
}

// ToYAML --
func ToYAML(value runtime.Object) ([]byte, error) {
	data, err := ToJSON(value)
	if err != nil {
		return nil, err
	}

	return JSONToYAML(data)
}

// JSONToYAML --
func JSONToYAML(src []byte) ([]byte, error) {
	jsondata := map[string]interface{}{}
	err := json.Unmarshal(src, &jsondata)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling json: %v", err)
	}
	yamldata, err := yaml2.Marshal(&jsondata)
	if err != nil {
		return nil, fmt.Errorf("error marshalling to yaml: %v", err)
	}

	return yamldata, nil
}

// GetConfigMap --
func GetConfigMap(context context.Context, client k8sclient.Reader, name string, namespace string) (*corev1.ConfigMap, error) {
	key := k8sclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}

	answer := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	if err := client.Get(context, key, &answer); err != nil {
		return nil, err
	}

	return &answer, nil
}

// HasConfigMap --
func HasConfigMap(context context.Context, client k8sclient.Reader, name string, namespace string) bool {
	key := k8sclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}

	answer := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	if err := client.Get(context, key, &answer); err != nil {
		return false
	}

	return true
}

// HasSecret --
func HasSecret(context context.Context, client k8sclient.Reader, name string, namespace string) bool {
	key := k8sclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}

	answer := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	if err := client.Get(context, key, &answer); err != nil {
		return false
	}
	return true
}

// GetSecret --
func GetSecret(context context.Context, client k8sclient.Reader, name string, namespace string) (*corev1.Secret, error) {
	key := k8sclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}

	answer := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	if err := client.Get(context, key, &answer); err != nil {
		return nil, err
	}

	return &answer, nil
}

// GetService --
func GetService(context context.Context, client k8sclient.Reader, name string, namespace string) (*corev1.Service, error) {
	key := k8sclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}

	answer := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	if err := client.Get(context, key, &answer); err != nil {
		return nil, err
	}

	return &answer, nil
}

// GetSecretRefValue returns the value of a secret in the supplied namespace --
func GetSecretRefValue(ctx context.Context, client k8sclient.Reader, namespace string, selector *corev1.SecretKeySelector) (string, error) {
	secret, err := GetSecret(ctx, client, selector.Name, namespace)
	if err != nil {
		return "", err
	}

	if data, ok := secret.Data[selector.Key]; ok {
		return string(data), nil
	}

	return "", fmt.Errorf("key %s not found in secret %s", selector.Key, selector.Name)
}

// GetConfigMapRefValue returns the value of a configmap in the supplied namespace
func GetConfigMapRefValue(ctx context.Context, client k8sclient.Reader, namespace string, selector *corev1.ConfigMapKeySelector) (string, error) {
	cm, err := GetConfigMap(ctx, client, selector.Name, namespace)
	if err != nil {
		return "", err
	}

	if data, ok := cm.Data[selector.Key]; ok {
		return data, nil
	}

	return "", fmt.Errorf("key %s not found in config map %s", selector.Key, selector.Name)
}

// ResolveValueSource --
func ResolveValueSource(ctx context.Context, client k8sclient.Reader, namespace string, valueSource *v1alpha1.ValueSource) (string, error) {
	if valueSource.ConfigMapKeyRef != nil && valueSource.SecretKeyRef != nil {
		return "", fmt.Errorf("value source has bot config map and secret configured")
	}
	if valueSource.ConfigMapKeyRef != nil {
		return GetConfigMapRefValue(ctx, client, namespace, valueSource.ConfigMapKeyRef)
	}
	if valueSource.SecretKeyRef != nil {
		return GetSecretRefValue(ctx, client, namespace, valueSource.SecretKeyRef)
	}

	return "", nil
}

// EnsureObject creates an object based on the error passed in from a `client.Get`
func EnsureObject(obj v1alpha1.OpenShiftObject, err error, client k8sclient.Writer) error {
	log := log.With("kind", obj.GetObjectKind().GroupVersionKind().Kind, "name", obj.GetName(), "namespace", obj.GetNamespace())

	if err != nil && errors.IsNotFound(err) {
		// Define a new Object
		log.Info("Creating")
		err = client.Create(context.TODO(), obj)
		if err != nil {
			log.Warn("Failed to create object. ", err)
			return err
		}
		// Object created successfully - return and requeue
		return nil
	} else if err != nil {
		log.Error("Failed to get object. ", err)
		return err
	}
	log.Debug("Skip reconcile - object already exists")
	return nil
}

// EnvironmentPropertiesExists --
func EnvironmentPropertiesExists(ctx context.Context, client k8sclient.Reader, namespace string, envs []corev1.EnvVar) bool {
	for _, env := range envs {
		if env.ValueFrom != nil {
			// check if this ConfigMap
			if env.ValueFrom.ConfigMapKeyRef != nil {
				_, err := GetConfigMapRefValue(ctx, client, namespace, env.ValueFrom.ConfigMapKeyRef)
				if err != nil {
					log.Infof("Error reading ConfigMap %s, for property: %s", env.ValueFrom.ConfigMapKeyRef.Name, env.Name)
					return false
				}
			} else if env.ValueFrom.SecretKeyRef != nil {
				_, err := GetSecretRefValue(ctx, client, namespace, env.ValueFrom.SecretKeyRef)
				if err != nil {
					log.Infof("Error reading Secret %s, for property: %s", env.ValueFrom.SecretKeyRef.Name, env.Name)
					return false
				}
			} else {
				log.Infof("Unknown type of ValueFrom configured for environment property: %s", env.Name)
				return false
			}
		}
	}
	return true
}

// RevisionOfConfigMapOrSecret --
func RevisionOfConfigMapOrSecret(ctx context.Context, client k8sclient.Reader, namespace string, env corev1.EnvVar) (string, error) {
	if env.Value != "" {
		return env.Value, nil
	}
	if env.ValueFrom != nil {
		// check if this ConfigMap
		if env.ValueFrom.ConfigMapKeyRef != nil {
			cm, err := GetConfigMap(ctx, client, env.ValueFrom.ConfigMapKeyRef.Name, namespace)
			if err != nil {
				log.Infof("Error reading ConfigMap %s, for property: %s", env.ValueFrom.ConfigMapKeyRef.Name, env.Name)
				return "", err
			}
			return cm.ObjectMeta.ResourceVersion, nil
		} else if env.ValueFrom.SecretKeyRef != nil {
			s, err := GetSecret(ctx, client, env.ValueFrom.SecretKeyRef.Name, namespace)
			if err != nil {
				log.Infof("Error reading Secret %s, for property: %s", env.ValueFrom.SecretKeyRef.Name, env.Name)
				return "", err
			}
			return s.ObjectMeta.ResourceVersion, nil
		} else {
			// ignore these as these field refs are not supplied directly by user
			return "", nil
		}
	}
	return env.Value, nil
}

// IsOpenshift detects if the application is running on OpenShift or not
func IsOpenshift(client kubernetes.Interface) bool {
	return HasServerGroup(client, "openshift.io")
}

// HasServerGroup detects if the given api group is supported by the server
func HasServerGroup(client kubernetes.Interface, groupName string) bool {
	if client.Discovery() != nil {
		groups, err := client.Discovery().ServerGroups()
		if err != nil {
			log.Warnf("Impossible to get server groups using discovery API: %s", err)
			return false
		}
		for _, group := range groups.Groups {
			if strings.Contains(group.Name, groupName) {
				return true
			}
		}
		return false
	}
	log.Warnf("Tried to discover the platform, but no discovery API is available")
	return false
}

// CreateSecret --
func CreateSecret(client teiidclient.Client, name, namespace string, owner metav1.Object, data map[string][]byte) error {
	// build the secret with keystore and truststore
	secret := corev1.Secret{
		Type: corev1.SecretTypeOpaque,
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}

	// set owner reference
	if owner != nil {
		err := controllerutil.SetControllerReference(owner, &secret, client.GetScheme())
		if err != nil {
			return err
		}
	}

	// create the secret
	_, err := client.CoreV1().Secrets(namespace).Create(&secret)
	if err != nil {
		log.Error("Failed to create the Keystore Secret")
		return err
	}
	return nil
}
