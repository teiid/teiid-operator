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
	"os"
	"strings"

	obuildv1 "github.com/openshift/api/build/v1"
	scheme "github.com/openshift/client-go/build/clientset/versioned/scheme"
	"github.com/teiid/teiid-operator/pkg/apis/vdb/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/controller/virtualdatabase/constants"
	"github.com/teiid/teiid-operator/pkg/util"
	"github.com/teiid/teiid-operator/pkg/util/envvar"
	"github.com/teiid/teiid-operator/pkg/util/image"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// News2IBuilderImageAction creates a new initialize action
func News2IBuilderImageAction() Action {
	return &s2iBuilderImageAction{}
}

type s2iBuilderImageAction struct {
	baseAction
}

// Name returns a common name of the action
func (action *s2iBuilderImageAction) Name() string {
	return "S2IBuilderImageAction"
}

// CanHandle tells whether this action can handle the virtualdatabase
func (action *s2iBuilderImageAction) CanHandle(vdb *v1alpha1.VirtualDatabase) bool {
	return vdb.Status.Phase == v1alpha1.ReconcilerPhaseS2IReady || vdb.Status.Phase == v1alpha1.ReconcilerPhaseBuilderImage
}

// Handle handles the virtualdatabase
func (action *s2iBuilderImageAction) Handle(ctx context.Context, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) error {

	if vdb.Status.Phase == v1alpha1.ReconcilerPhaseS2IReady {
		vdb.Status.Phase = v1alpha1.ReconcilerPhaseBuilderImage

		opDeployment := &appsv1.Deployment{}
		opDeploymentNS := os.Getenv("WATCH_NAMESPACE")
		opDeploymentName := os.Getenv("OPERATOR_NAME")
		r.client.Get(ctx, types.NamespacedName{Namespace: opDeploymentNS, Name: opDeploymentName}, opDeployment)

		log.Info("Building Base builder Image")
		// Define new BuildConfig objects
		buildConfig := action.buildBC(vdb, r)
		// set ownerreference for service BC only
		if _, err := image.EnsureImageStream(buildConfig.Name, vdb.ObjectMeta.Namespace, true, opDeployment, r.imageClient, r.scheme); err != nil {
			return err
		}

		// check to make sure the base s2i image for the build is available
		isName := buildConfig.Spec.Strategy.SourceStrategy.From.Name
		isNameSpace := buildConfig.Spec.Strategy.SourceStrategy.From.Namespace
		_, err := r.imageClient.ImageStreamTags(isNameSpace).Get(isName, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			log.Warn(isNameSpace, "/", isName, " ImageStreamTag does not exist and is required for this build.")
			return err
		} else if err != nil {
			return err
		}

		// Check if this BC already exists
		bc, err := r.buildClient.BuildConfigs(buildConfig.Namespace).Get(buildConfig.Name, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			log.Info("Creating a new BuildConfig ", buildConfig.Name, " in namespace ", buildConfig.Namespace)

			// make the Operator as the owner
			err := controllerutil.SetControllerReference(opDeployment, &buildConfig, r.scheme)
			if err != nil {
				log.Error(err)
			}

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
			if err = action.triggerBuild(*bc, r); err != nil {
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
func (action *s2iBuilderImageAction) buildBC(vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) obuildv1.BuildConfig {
	bc := obuildv1.BuildConfig{}
	images := constants.RuntimeImageDefaults[vdb.Spec.Runtime.Type]
	env := []corev1.EnvVar{}
	envvar.SetVal(&env, "DEPLOYMENTS_DIR", "/opt/jboss") // this is avoid copying the jar file
	envvar.SetVal(&env, "MAVEN_ARGS_APPEND", "-Dmaven.compiler.source=1.8 -Dmaven.compiler.target=1.8")
	envvar.SetVal(&env, "ARTIFACT_DIR", "target/")

	incremental := true
	for _, imageDefaults := range images {
		if imageDefaults.BuilderImage {

			imageName := fmt.Sprintf("%s:%s", imageDefaults.ImageStreamName, imageDefaults.ImageStreamTag)
			isNamespace := imageDefaults.ImageStreamNamespace
			// check if the base image is found otherwise use from dockerhub, add to local images
			if !image.CheckImageStream(imageName, isNamespace, r.imageClient) {
				dockerImage := fmt.Sprintf("%s/%s/%s", imageDefaults.ImageRegistry, imageDefaults.ImageRepository, imageDefaults.ImageStreamName)
				image.CreateImageStream(imageDefaults.ImageStreamName, vdb.ObjectMeta.Namespace, dockerImage, imageDefaults.ImageStreamTag, r.imageClient, r.scheme)
				isNamespace = vdb.ObjectMeta.Namespace
			}

			builderName := constants.BuilderImageTargetName
			bc = obuildv1.BuildConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      builderName,
					Namespace: vdb.ObjectMeta.Namespace,
				},
			}
			bc.SetGroupVersionKind(obuildv1.SchemeGroupVersion.WithKind("BuildConfig"))
			bc.Spec.Source.Binary = &obuildv1.BinaryBuildSource{}
			bc.Spec.Output.To = &corev1.ObjectReference{Name: strings.Join([]string{builderName, "latest"}, ":"), Kind: "ImageStreamTag"}
			bc.Spec.Strategy.Type = obuildv1.SourceBuildStrategyType
			bc.Spec.Strategy.SourceStrategy = &obuildv1.SourceBuildStrategy{
				Incremental: &incremental,
				Env:         env,
				From: corev1.ObjectReference{
					Name:      imageName,
					Namespace: isNamespace,
					Kind:      "ImageStreamTag",
				},
			}
		}
	}
	return bc
}

// triggerBuild triggers a BuildConfig to start a new build
func (action *s2iBuilderImageAction) triggerBuild(bc obuildv1.BuildConfig, r *ReconcileVirtualDatabase) error {
	log := log.With("kind", "BuildConfig", "name", bc.GetName(), "namespace", bc.GetNamespace())
	log.Info("starting the build for base image")
	buildConfig, err := r.buildClient.BuildConfigs(bc.Namespace).Get(bc.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	vdb := &v1alpha1.VirtualDatabase{}
	vdb.ObjectMeta.Name = "virtualdatabase-image"
	vdb.ObjectMeta.Namespace = bc.GetNamespace()
	vdb.Spec.Build.Source.DDL = action.ddlFile()

	files := map[string]string{}
	pom, err := GeneratePom(vdb, true, true)
	if err != nil {
		return err
	}

	files["/pom.xml"] = pom
	files["/src/main/resources/teiid.ddl"] = action.ddlFile()
	log.Debug(pom)

	tarReader, err := util.Tar(files)
	if err != nil {
		return err
	}

	// do the binary build
	binaryBuildRequest := obuildv1.BinaryBuildRequestOptions{ObjectMeta: metav1.ObjectMeta{Name: buildConfig.Name}}
	binaryBuildRequest.SetGroupVersionKind(obuildv1.SchemeGroupVersion.WithKind("BinaryBuildRequestOptions"))
	log.Info("Triggering binary build ", buildConfig.Name)
	err = r.buildClient.RESTClient().Post().
		Namespace(bc.GetNamespace()).
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
	return nil
}

func (action *s2iBuilderImageAction) ddlFile() string {
	return `CREATE DATABASE customer OPTIONS (ANNOTATION 'Customer VDB');	USE DATABASE customer;
	SET NAMESPACE 'http://teiid.org/rest' AS REST;
	CREATE FOREIGN DATA WRAPPER h2;
	CREATE SERVER mydb TYPE 'NONE' FOREIGN DATA WRAPPER h2;`
}
