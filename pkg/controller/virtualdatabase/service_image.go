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
	"errors"
	"fmt"
	"strings"

	obuildv1 "github.com/openshift/api/build/v1"
	scheme "github.com/openshift/client-go/build/clientset/versioned/scheme"
	"github.com/teiid/teiid-operator/pkg/apis/vdb/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/controller/virtualdatabase/constants"
	"github.com/teiid/teiid-operator/pkg/controller/virtualdatabase/pom"
	"github.com/teiid/teiid-operator/pkg/controller/virtualdatabase/shared"
	"github.com/teiid/teiid-operator/pkg/util/envvar"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
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

		// check for the VDB source type
		if vdb.Spec.Build.GitSource.URI == "" && vdb.Spec.Build.DDLSource.Contents == "" {
			return errors.New("Only Git or Content based VDBs are allowed, neither are defined")
		}

		// Define new BuildConfig objects
		buildConfig := action.serviceBC(vdb)
		if _, err := r.ensureImageStream(buildConfig.Name, vdb.Namespace, true, vdb); err != nil {
			return err
		}

		// Check if this BC already exists
		bc, err := r.buildClient.BuildConfigs(buildConfig.Namespace).Get(buildConfig.Name, metav1.GetOptions{})
		if err != nil && apierr.IsNotFound(err) {
			log.Info("Creating a new BuildConfig ", buildConfig.Name, " in namespace ", buildConfig.Namespace)
			// set ownerreference for service BC only
			err := controllerutil.SetControllerReference(vdb, &buildConfig, r.scheme)
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
	baseImage := corev1.ObjectReference{Name: strings.Join([]string{constants.BuilderImageTargetName, "latest"}, ":"), Kind: "ImageStreamTag"}

	// set it back original default
	envvar.SetVal(&vdb.Spec.Build.Env, "DEPLOYMENTS_DIR", "/deployments")
	// this below is add clean, to remove the previous jar file in target from builder image
	envvar.SetVal(&vdb.Spec.Build.Env, "MAVEN_ARGS", "clean package -DskipTests -Dmaven.javadoc.skip=true -Dmaven.site.skip=true -Dmaven.source.skip=true -Djacoco.skip=true -Dcheckstyle.skip=true -Dfindbugs.skip=true -Dpmd.skip=true -Dfabric8.skip=true -e -B")

	bc := obuildv1.BuildConfig{}
	bc = obuildv1.BuildConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vdb.ObjectMeta.Name,
			Namespace: vdb.ObjectMeta.Namespace,
			Labels: map[string]string{
				"app": vdb.ObjectMeta.Name,
			},
		},
	}
	bc.SetGroupVersionKind(obuildv1.SchemeGroupVersion.WithKind("BuildConfig"))
	bc.Spec.Output.To = &corev1.ObjectReference{Name: strings.Join([]string{vdb.ObjectMeta.Name, "latest"}, ":"), Kind: "ImageStreamTag"}

	// for some reason "vdb.Spec.Build.GitSource" comes in as empty object rather than nil
	if vdb.Spec.Build.GitSource.URI != "" {
		log.Info("Git based build is chosen..")
		bc.Spec.Source.Git = &obuildv1.GitBuildSource{
			URI: vdb.Spec.Build.GitSource.URI,
			Ref: vdb.Spec.Build.GitSource.Reference,
		}
		bc.Spec.Source.ContextDir = vdb.Spec.Build.GitSource.ContextDir
	} else if vdb.Spec.Build.DDLSource.Contents != "" {
		log.Info("DDL based build is chosen..")
		bc.Spec.Source.Binary = &obuildv1.BinaryBuildSource{}
	}

	bc.Spec.Strategy.Type = obuildv1.SourceBuildStrategyType
	bc.Spec.Strategy.SourceStrategy = &obuildv1.SourceBuildStrategy{
		From:        baseImage,
		ForcePull:   false,
		Incremental: vdb.Spec.Build.Incremental,
		Env:         vdb.Spec.Build.Env,
	}
	bc.Spec.Source.Type = obuildv1.BuildSourceImage
	bc.Spec.Triggers = []obuildv1.BuildTriggerPolicy{
		{
			Type:        obuildv1.ImageChangeBuildTriggerType,
			ImageChange: &obuildv1.ImageChangeTrigger{From: &baseImage},
		},
	}
	return bc
}

// triggerBuild triggers a BuildConfig to start a new build
func (action *serviceImageAction) triggerBuild(bc obuildv1.BuildConfig, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) error {
	log := log.With("kind", "BuildConfig", "name", bc.GetName(), "namespace", bc.GetNamespace())
	buildConfig, err := r.buildClient.BuildConfigs(bc.Namespace).Get(bc.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if buildConfig.Spec.Source.Type == obuildv1.BuildSourceBinary {
		log.Info("starting the binary build for service image ")
		files := map[string]string{}

		//Binary build, generate the pom file
		pom, err := pom.GeneratePom(vdb, false)
		if err != nil {
			return nil
		}
		files["/pom.xml"] = pom
		files["/src/main/resources/teiid.ddl"] = vdb.Spec.Build.DDLSource.Contents

		tarReader, err := shared.Tar(files)
		if err != nil {
			return err
		}
		isName := buildConfig.Spec.Strategy.SourceStrategy.From.Name
		_, err = r.imageClient.ImageStreamTags(buildConfig.Namespace).Get(isName, metav1.GetOptions{})
		if err != nil && apierr.IsNotFound(err) {
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
