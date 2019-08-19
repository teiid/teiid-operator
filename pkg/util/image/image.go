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

package image

import (
	"strings"

	oimagev1 "github.com/openshift/api/image/v1"

	imagev1 "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	"github.com/teiid/teiid-operator/pkg/util/logs"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var log = logs.GetLogger("virtualdatabase")

// CheckImageStream checks for ImageStream
func CheckImageStream(name, namespace string, client *imagev1.ImageV1Client) bool {
	log := log.With("kind", "ImageStream", "name", name, "namespace", namespace)
	result := strings.Split(name, ":")
	_, err := client.ImageStreams(namespace).Get(result[0], metav1.GetOptions{})
	if err != nil {
		log.Debug("Object does not exist")
		return false
	}
	return true
}

// EnsureImageStream ...
func EnsureImageStream(name string, namespace string, setOwner bool, owner v1.Object,
	client *imagev1.ImageV1Client, scheme *runtime.Scheme) (string, error) {

	if CheckImageStream(name, namespace, client) {
		return namespace, nil
	}
	err := createLocalImageStream(name, namespace, setOwner, owner, client, scheme)
	if err != nil {
		return namespace, err
	}
	return namespace, nil
}

// CreateImageStream --
func CreateImageStream(name string, namespace string, dockerImage string, tag string, client *imagev1.ImageV1Client, scheme *runtime.Scheme) error {

	isnew := &oimagev1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: oimagev1.ImageStreamSpec{
			Tags: []oimagev1.TagReference{
				{
					Name: tag,
					From: &corev1.ObjectReference{
						Kind: "DockerImage",
						Name: dockerImage + ":" + tag,
					},
				},
			},
		},
	}
	isnew.SetGroupVersionKind(oimagev1.SchemeGroupVersion.WithKind("ImageStream"))
	log := log.With("kind", isnew.GetObjectKind().GroupVersionKind().Kind, "name", isnew.Name, "namespace", isnew.Namespace)
	log.Info("Creating")

	_, err := client.ImageStreams(isnew.Namespace).Create(isnew)
	if err != nil && !errors.IsAlreadyExists(err) {
		log.Info("Already exists.", err)
	}
	return nil
}

// createLocalImageStream creates local ImageStream
func createLocalImageStream(tagRefName string, namespace string, setOwner bool, owner v1.Object,
	client *imagev1.ImageV1Client, scheme *runtime.Scheme) error {

	result := strings.Split(tagRefName, ":")
	if len(result) == 1 {
		result = append(result, "latest")
	}

	isnew := &oimagev1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name:      result[0],
			Namespace: namespace,
		},
		Spec: oimagev1.ImageStreamSpec{
			LookupPolicy: oimagev1.ImageLookupPolicy{
				Local: true,
			},
		},
	}
	isnew.SetGroupVersionKind(oimagev1.SchemeGroupVersion.WithKind("ImageStream"))
	if setOwner {
		err := controllerutil.SetControllerReference(owner, isnew, scheme)
		if err != nil {
			log.Error("Error setting controller reference for ImageStream. ", err)
			return err
		}
	}

	log := log.With("kind", isnew.GetObjectKind().GroupVersionKind().Kind, "name", isnew.Name, "namespace", isnew.Namespace)
	log.Info("Creating")

	_, err := client.ImageStreams(isnew.Namespace).Create(isnew)
	if err != nil && !errors.IsAlreadyExists(err) {
		log.Info("Already exists.")
	}
	return nil
}
