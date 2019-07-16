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
	"context"
	"fmt"
	"strings"

	obuildv1 "github.com/openshift/api/build/v1"
	scheme "github.com/openshift/client-go/build/clientset/versioned/scheme"
	"github.com/teiid/teiid-operator/pkg/apis/vdb/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/controller/virtualdatabase/constants"
	"github.com/teiid/teiid-operator/pkg/controller/virtualdatabase/shared"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewBuilderImageAction creates a new initialize action
func NewBuilderImageAction() Action {
	return &builderImageAction{}
}

type builderImageAction struct {
	baseAction
}

// Name returns a common name of the action
func (action *builderImageAction) Name() string {
	return "BuilderImageAction"
}

// CanHandle tells whether this action can handle the virtualdatabase
func (action *builderImageAction) CanHandle(vdb *v1alpha1.VirtualDatabase) bool {
	return vdb.Status.Phase == v1alpha1.ReconcilerPhaseS2IReady || vdb.Status.Phase == v1alpha1.ReconcilerPhaseBuilderImage
}

// Handle handles the virtualdatabase
func (action *builderImageAction) Handle(ctx context.Context, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) error {

	if vdb.Status.Phase == v1alpha1.ReconcilerPhaseS2IReady {
		vdb.Status.Phase = v1alpha1.ReconcilerPhaseBuilderImage

		log.Info("Building Base builder Image")
		// Define new BuildConfig objects
		buildConfig := action.buildBC(vdb)
		// set ownerreference for service BC only
		if _, err := r.ensureImageStream(buildConfig.Name, vdb, false); err != nil {
			return err
		}

		// Check if this BC already exists
		bc, err := r.buildClient.BuildConfigs(buildConfig.Namespace).Get(buildConfig.Name, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			log.Info("Creating a new BuildConfig ", buildConfig.Name, " in namespace ", buildConfig.Namespace)
			bc, err = r.buildClient.BuildConfigs(buildConfig.Namespace).Create(&buildConfig)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		log.Info("Created BuildConfig")

		// Trigger first build of "builder" and binary BCs
		if bc.Status.LastVersion == 0 {
			log.Info("triggering the base builder image build")
			if err = action.triggerBuild(*bc, vdb, r); err != nil {
				return err
			}
		}
	} else if vdb.Status.Phase == v1alpha1.ReconcilerPhaseBuilderImage {
		builds := &obuildv1.BuildList{}
		options := metav1.ListOptions{
			FieldSelector: "metadata.namespace=" + vdb.ObjectMeta.Namespace,
			LabelSelector: "buildconfig=" + constants.BuilderImageTargetName,
		}

		builds, err := r.buildClient.Builds(vdb.ObjectMeta.Namespace).List(options)
		if err != nil {
			return err
		}

		for _, build := range builds.Items {
			// set status of the build
			if build.Status.Phase == obuildv1.BuildPhaseComplete && vdb.Status.Phase != v1alpha1.ReconcilerPhaseBuilderImageFinished {
				vdb.Status.Phase = v1alpha1.ReconcilerPhaseBuilderImageFinished
			} else if (build.Status.Phase == obuildv1.BuildPhaseError ||
				build.Status.Phase == obuildv1.BuildPhaseFailed ||
				build.Status.Phase == obuildv1.BuildPhaseCancelled) && vdb.Status.Phase != v1alpha1.ReconcilerPhaseBuilderImageFailed {
				vdb.Status.Phase = v1alpha1.ReconcilerPhaseBuilderImageFailed
			} else if build.Status.Phase == obuildv1.BuildPhaseRunning && vdb.Status.Phase != v1alpha1.ReconcilerPhaseBuilderImage {
				vdb.Status.Phase = v1alpha1.ReconcilerPhaseBuilderImage
			}
		}
	}
	return nil
}

// newBCForCR returns a BuildConfig with the same name/namespace as the cr
func (action *builderImageAction) buildBC(vdb *v1alpha1.VirtualDatabase) obuildv1.BuildConfig {
	bc := obuildv1.BuildConfig{}
	images := constants.RuntimeImageDefaults[vdb.Spec.Runtime]

	for _, imageDefaults := range images {
		if imageDefaults.BuilderImage {
			builderName := constants.BuilderImageTargetName
			bc = obuildv1.BuildConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      builderName,
					Namespace: vdb.ObjectMeta.Namespace,
					Labels: map[string]string{
						"app": vdb.ObjectMeta.Name,
					},
				},
			}
			bc.SetGroupVersionKind(obuildv1.SchemeGroupVersion.WithKind("BuildConfig"))
			bc.Spec.Source.Git = &obuildv1.GitBuildSource{
				URI: vdb.Spec.Build.GitSource.URI,
				Ref: vdb.Spec.Build.GitSource.Reference,
			}
			bc.Spec.Source.ContextDir = vdb.Spec.Build.GitSource.ContextDir
			bc.Spec.Output.To = &corev1.ObjectReference{Name: strings.Join([]string{builderName, "latest"}, ":"), Kind: "ImageStreamTag"}
			bc.Spec.Strategy.Type = obuildv1.SourceBuildStrategyType
			bc.Spec.Strategy.SourceStrategy = &obuildv1.SourceBuildStrategy{
				Incremental: vdb.Spec.Build.Incremental,
				Env:         vdb.Spec.Build.Env,
				From: corev1.ObjectReference{
					Name:      fmt.Sprintf("%s:%s", imageDefaults.ImageStreamName, imageDefaults.ImageStreamTag),
					Namespace: imageDefaults.ImageStreamNamespace,
					Kind:      "ImageStreamTag",
				},
			}
		}
	}
	return bc
}

// triggerBuild triggers a BuildConfig to start a new build
func (action *builderImageAction) triggerBuild(bc obuildv1.BuildConfig, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) error {
	log := log.With("kind", "BuildConfig", "name", bc.GetName(), "namespace", bc.GetNamespace())
	buildConfig, err := r.buildClient.BuildConfigs(bc.Namespace).Get(bc.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if buildConfig.Spec.Source.Type == obuildv1.BuildSourceBinary {
		files := map[string]string{}
		// Create list of files to archive
		for _, file := range vdb.Spec.Build.SourceFileChanges {
			files[file.RelativePath] = file.Contents
		}
		tarReader, err := shared.Tar(files)
		if err != nil {
			return err
		}
		isName := buildConfig.Spec.Strategy.SourceStrategy.From.Name
		_, err = r.imageClient.ImageStreamTags(buildConfig.Namespace).Get(isName, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			log.Warn(isName, " ImageStreamTag does not exist yet and is required for this build.")
		} else if err != nil {
			return err
		} else {
			binaryBuildRequest := obuildv1.BinaryBuildRequestOptions{ObjectMeta: metav1.ObjectMeta{Name: buildConfig.Name}}
			binaryBuildRequest.SetGroupVersionKind(obuildv1.SchemeGroupVersion.WithKind("BinaryBuildRequestOptions"))
			log.Info("Triggering binary build ", buildConfig.Name)
			err = r.buildClient.RESTClient().Post().
				Namespace(vdb.ObjectMeta.Namespace).
				Resource("buildconfigs").
				Name(buildConfig.Name).
				SubResource("instantiatebinary").
				Body(tarReader).
				VersionedParams(&binaryBuildRequest, scheme.ParameterCodec).
				Do().
				Into(&obuildv1.Build{})
			if err != nil {
				return err
			}
		}
	} else {
		buildRequest := obuildv1.BuildRequest{ObjectMeta: metav1.ObjectMeta{Name: buildConfig.Name}}
		buildRequest.SetGroupVersionKind(obuildv1.SchemeGroupVersion.WithKind("BuildRequest"))
		buildRequest.TriggeredBy = []obuildv1.BuildTriggerCause{{Message: fmt.Sprintf("Triggered by %s operator", vdb.Kind)}}
		log.Info("Triggering build ", buildConfig.Name)
		_, err := r.buildClient.BuildConfigs(buildConfig.Namespace).Instantiate(buildConfig.Name, &buildRequest)
		if err != nil {
			return err
		}
	}

	return nil
}
