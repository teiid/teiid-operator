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

package s2i

import (
	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"

	buildv1 "github.com/openshift/api/build/v1"
	imagev1 "github.com/openshift/api/image/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateBuildConfiguration(vdb *v1alpha1.VirtualDatabase) buildv1.BuildConfig {
	bc := buildv1.BuildConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: buildv1.SchemeGroupVersion.String(),
			Kind:       "BuildConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "teiid-build-" + vdb.ObjectMeta.Name,
			Namespace: vdb.ObjectMeta.Namespace,
			Labels: map[string]string{
				"application": vdb.ObjectMeta.Name,
				"managedby":   "syndesis",
			},
		},
		Spec: buildv1.BuildConfigSpec{
			RunPolicy: buildv1.BuildRunPolicySerialLatestOnly,
			CommonSpec: buildv1.CommonSpec{
				Source: buildv1.BuildSource{
					Type: buildv1.BuildSourceBinary,
				},
				Strategy: buildv1.BuildStrategy{
					SourceStrategy: &buildv1.SourceBuildStrategy{
						From: corev1.ObjectReference{
							Kind: "ImageStreamTag",
							Name: vdb.Spec.ImageSpec.BaseImage,
						},
					},
				},
				Output: buildv1.BuildOutput{
					To: &corev1.ObjectReference{
						Kind: "ImageStreamTag",
						Name: "teiid-" + vdb.ObjectMeta.Name + ":latest",
					},
				},
			},
		},
	}
	return bc
}

func CreateImageStream(vdb *v1alpha1.VirtualDatabase) imagev1.ImageStream {
	is := imagev1.ImageStream{
		TypeMeta: metav1.TypeMeta{
			APIVersion: imagev1.SchemeGroupVersion.String(),
			Kind:       "ImageStream",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "teiid-" + vdb.ObjectMeta.Name,
			Namespace: vdb.ObjectMeta.Namespace,
		},
		Spec: imagev1.ImageStreamSpec{
			LookupPolicy: imagev1.ImageLookupPolicy{
				Local: true,
			},
		},
	}
	return is
}

/*
// Publisher --
func Publisher(vdb *v1alpha1.VirtualDatabase) error {


	resource, err := ioutil.ReadFile(ctx.Archive)
	if err != nil {
		return errors.Wrap(err, "cannot fully read tar file "+ctx.Archive)
	}

	restClient, err := customclient.GetClientFor(ctx.Client, "build.openshift.io", "v1")
	if err != nil {
		return err
	}

	result := restClient.Post().
		Namespace(ctx.Namespace).
		Body(resource).
		Resource("buildconfigs").
		Name("camel-k-" + ctx.Build.Meta.Name).
		SubResource("instantiatebinary").
		Do()

	if result.Error() != nil {
		return errors.Wrap(result.Error(), "cannot instantiate binary")
	}

	data, err := result.Raw()
	if err != nil {
		return errors.Wrap(err, "no raw data retrieved")
	}

	ocbuild := buildv1.Build{}
	err = json.Unmarshal(data, &ocbuild)
	if err != nil {
		return errors.Wrap(err, "cannot unmarshal instantiated binary response")
	}

	err = kubernetes.WaitCondition(ctx.C, ctx.Client, &ocbuild, func(obj interface{}) (bool, error) {
		if val, ok := obj.(*buildv1.Build); ok {
			if val.Status.Phase == buildv1.BuildPhaseComplete {
				return true, nil
			} else if val.Status.Phase == buildv1.BuildPhaseCancelled ||
				val.Status.Phase == buildv1.BuildPhaseFailed ||
				val.Status.Phase == buildv1.BuildPhaseError {
				return false, errors.New("build failed")
			}
		}
		return false, nil
	}, ctx.Build.Platform.Build.Timeout.Duration)

	if err != nil {
		return err
	}

	key, err := k8sclient.ObjectKeyFromObject(&is)
	if err != nil {
		return err
	}
	err = ctx.Client.Get(ctx.C, key, &is)
	if err != nil {
		return err
	}

	if is.Status.DockerImageRepository == "" {
		return errors.New("dockerImageRepository not available in ImageStream")
	}

	ctx.Image = is.Status.DockerImageRepository + ":" + ctx.Build.Meta.ResourceVersion

	return nil
}

func replaceHost(ctx *builder.Context) error {
	ctx.PublicImage = getImageWithOpenShiftHost(ctx.Image)
	return nil
}

func getImageWithOpenShiftHost(image string) string {
	pattern := regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+([:/].*)`)
	return pattern.ReplaceAllString(image, openShiftDockerRegistryHost+"$1")
}
*/
