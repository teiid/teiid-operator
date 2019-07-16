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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// NewServiceImageAction creates a new initialize action
func NewServiceImageAction() Action {
	return &serviceImageAction{}
}

type serviceImageAction struct {
	baseAction
}

// Name returns a common name of the action
func (action *serviceImageAction) Name() string {
	return "ServiceImageAction"
}

// CanHandle tells whether this action can handle the virtualdatabase
func (action *serviceImageAction) CanHandle(vdb *v1alpha1.VirtualDatabase) bool {
	return vdb.Status.Phase == v1alpha1.ReconcilerPhaseBuilderImageFinished || vdb.Status.Phase == v1alpha1.ReconcilerPhaseServiceImage
}

// Handle handles the virtualdatabase
func (action *serviceImageAction) Handle(ctx context.Context, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) error {

	if vdb.Status.Phase == v1alpha1.ReconcilerPhaseBuilderImageFinished {
		vdb.Status.Phase = v1alpha1.ReconcilerPhaseServiceImage

		// Define new BuildConfig objects
		buildConfig := action.serviceBC(vdb)
		// set ownerreference for service BC only
		err := controllerutil.SetControllerReference(vdb, &buildConfig, r.scheme)
		if err != nil {
			log.Error(err)
		}
		if _, err := r.ensureImageStream(buildConfig.Name, vdb, true); err != nil {
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

		// Trigger first build of "builder" and binary BCs
		if bc.Spec.Source.Type == obuildv1.BuildSourceBinary && bc.Status.LastVersion == 0 {
			if err = action.triggerBuild(*bc, vdb, r); err != nil {
				return err
			}
		}
	} else if vdb.Status.Phase == v1alpha1.ReconcilerPhaseServiceImage {

		builds := &obuildv1.BuildList{}
		options := metav1.ListOptions{
			FieldSelector: "metadata.namespace=" + vdb.ObjectMeta.Namespace,
			LabelSelector: "buildconfig=" + vdb.ObjectMeta.Name,
		}

		builds, err := r.buildClient.Builds(vdb.ObjectMeta.Namespace).List(options)
		if err != nil {
			return err
		}

		for _, build := range builds.Items {
			// set status of the build
			if build.Status.Phase == obuildv1.BuildPhaseComplete {
				vdb.Status.Phase = v1alpha1.ReconcilerPhaseServiceImageFinished
			} else if build.Status.Phase == obuildv1.BuildPhaseError ||
				build.Status.Phase == obuildv1.BuildPhaseFailed ||
				build.Status.Phase == obuildv1.BuildPhaseCancelled {
				vdb.Status.Phase = v1alpha1.ReconcilerPhaseServiceImageFailed
			} else if build.Status.Phase == obuildv1.BuildPhaseRunning {
				vdb.Status.Phase = v1alpha1.ReconcilerPhaseServiceImage
			}
		}
	}
	return nil
}

func (action *serviceImageAction) serviceBC(vdb *v1alpha1.VirtualDatabase) obuildv1.BuildConfig {
	serviceBC := obuildv1.BuildConfig{}

	serviceBC = obuildv1.BuildConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vdb.ObjectMeta.Name,
			Namespace: vdb.ObjectMeta.Namespace,
			Labels: map[string]string{
				"app": vdb.ObjectMeta.Name,
			},
		},
	}
	serviceBC.SetGroupVersionKind(obuildv1.SchemeGroupVersion.WithKind("BuildConfig"))
	serviceBC.Spec.Output.To = &corev1.ObjectReference{Name: strings.Join([]string{vdb.ObjectMeta.Name, "latest"}, ":"), Kind: "ImageStreamTag"}
	serviceBC.Spec.Strategy.Type = obuildv1.SourceBuildStrategyType

	baseImage := corev1.ObjectReference{Name: strings.Join([]string{constants.BuilderImageTargetName, "latest"}, ":"), Kind: "ImageStreamTag"}
	serviceBC.Spec.Strategy.SourceStrategy = &obuildv1.SourceBuildStrategy{
		From:      baseImage,
		ForcePull: false,
	}
	if len(vdb.Spec.Build.SourceFileChanges) > 0 {
		serviceBC.Spec.Source.Type = obuildv1.BuildSourceBinary
		//serviceBC.Spec.Source.Binary = &obuildv1.BinaryBuildSource{}
	} else {
		serviceBC.Spec.Source.Type = obuildv1.BuildSourceImage
		serviceBC.Spec.Triggers = []obuildv1.BuildTriggerPolicy{
			{
				Type:        obuildv1.ImageChangeBuildTriggerType,
				ImageChange: &obuildv1.ImageChangeTrigger{From: &baseImage},
			},
		}
	}
	return serviceBC
}

// triggerBuild triggers a BuildConfig to start a new build
func (action *serviceImageAction) triggerBuild(bc obuildv1.BuildConfig, cr *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) error {
	log := log.With("kind", "BuildConfig", "name", bc.GetName(), "namespace", bc.GetNamespace())
	buildConfig, err := r.buildClient.BuildConfigs(bc.Namespace).Get(bc.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if buildConfig.Spec.Source.Type == obuildv1.BuildSourceBinary {
		files := map[string]string{}
		// Create list of files to archive
		for _, file := range cr.Spec.Build.SourceFileChanges {
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
				Namespace(cr.ObjectMeta.Namespace).
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
		buildRequest.TriggeredBy = []obuildv1.BuildTriggerCause{{Message: fmt.Sprintf("Triggered by %s operator", cr.Kind)}}
		log.Info("Triggering build ", buildConfig.Name)
		_, err := r.buildClient.BuildConfigs(buildConfig.Namespace).Instantiate(buildConfig.Name, &buildRequest)
		if err != nil {
			return err
		}
	}

	return nil
}
