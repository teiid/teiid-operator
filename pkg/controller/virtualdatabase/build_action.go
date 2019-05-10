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
	"encoding/json"
	"io/ioutil"

	buildv1 "github.com/openshift/api/build/v1"
	"github.com/pkg/errors"
	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/controller/s2i"
	"github.com/teiid/teiid-operator/pkg/util/kubernetes/customclient"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// NewBuildAction creates a new build action
func NewBuildAction() Action {
	return &buildAction{}
}

type buildAction struct {
	baseAction
}

// Name returns a common name of the action
func (action *buildAction) Name() string {
	return "build"
}

// CanHandle tells whether this action can handle the virtualdatabase
func (action *buildAction) CanHandle(vdb *v1alpha1.VirtualDatabase) bool {
	return vdb.Status.Phase == v1alpha1.PublishingPhaseCodeGenerationCompleted || vdb.Status.Phase == v1alpha1.PublishingPhaseBuildImageSubmitted || vdb.Status.Phase == v1alpha1.PublishingPhaseBuildImageRunning
}

// Handle handles the virtualdatabase
func (action *buildAction) Handle(ctx context.Context, vdb *v1alpha1.VirtualDatabase) error {
	// better not changing the spec section of the target because it may be used for comparison by a
	// higher level controller
	if vdb.Status.Phase == v1alpha1.PublishingPhaseCodeGenerationCompleted {
		return submitForBuild(ctx, action, vdb)
	} else if vdb.Status.Phase == v1alpha1.PublishingPhaseBuildImageSubmitted || vdb.Status.Phase == v1alpha1.PublishingPhaseBuildImageRunning {
		return checkBuildProgress(ctx, action, vdb)
	}
	return nil
}

func checkBuildProgress(ctx context.Context, action *buildAction, vdb *v1alpha1.VirtualDatabase) error {
	target := vdb.DeepCopy()
	target.Status.Phase = v1alpha1.PublishingPhaseBuildImageRunning
	target.Status.Image = ""

	restClient, err := customclient.GetClientFor(action.client, "build.openshift.io", "v1")
	if err != nil {
		return err
	}

	result := restClient.Get().
		Namespace(vdb.ObjectMeta.Namespace).
		Resource("builds").
		Name("teiid-" + vdb.ObjectMeta.Name).
		Do()

	if result.Error() != nil {
		return errors.Wrap(result.Error(), "Can not check status of build")
	}

	data, err := result.Raw()
	if err != nil {
		return errors.Wrap(err, "no raw data retrieved")
	}

	build := buildv1.Build{}
	err = json.Unmarshal(data, &build)
	if err != nil {
		return errors.Wrap(err, "cannot unmarshal build response")
	}

	if build.Status.Phase == buildv1.BuildPhaseComplete {
		target.Status.Phase = v1alpha1.PublishingPhaseBuildImageComplete
	} else if build.Status.Phase == buildv1.BuildPhaseCancelled ||
		build.Status.Phase == buildv1.BuildPhaseFailed ||
		build.Status.Phase == buildv1.BuildPhaseError {
		return errors.New("build failed")
	}

	action.Log.Info("build finished", "phase", target.Status.Phase)
	return action.client.Status().Update(ctx, target)
}

func submitForBuild(ctx context.Context, action *buildAction, vdb *v1alpha1.VirtualDatabase) error {
	target := vdb.DeepCopy()
	target.Status.Phase = v1alpha1.PublishingPhaseBuildImageSubmitted
	target.Status.Image = ""

	buildStatus := ctx.Value(v1alpha1.BuildStatusKey).(v1alpha1.BuildStatus)
	archiveFile := buildStatus.TarFile

	action.Log.Info("Starting Build")

	bc := s2i.CreateBuildConfiguration(vdb)
	err := action.client.Delete(ctx, &bc)
	if err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrap(err, "cannot delete build config")
	}

	err = action.client.Create(ctx, &bc)
	if err != nil {
		return errors.Wrap(err, "cannot create build config")
	}

	is := s2i.CreateImageStream(vdb)
	err = action.client.Delete(ctx, &is)
	if err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrap(err, "cannot delete image stream")
	}

	err = action.client.Create(ctx, &is)
	if err != nil {
		return errors.Wrap(err, "cannot create image stream")
	}

	resource, err := ioutil.ReadFile(archiveFile)
	if err != nil {
		return errors.Wrap(err, "cannot fully read tar file "+archiveFile)
	}

	restClient, err := customclient.GetClientFor(action.client, "build.openshift.io", "v1")
	if err != nil {
		return err
	}

	result := restClient.Post().
		Namespace(vdb.ObjectMeta.Namespace).
		Body(resource).
		Resource("buildconfigs").
		Name("teiid-" + vdb.ObjectMeta.Name).
		SubResource("instantiatebinary").
		Do()

	if result.Error() != nil {
		return errors.Wrap(result.Error(), "cannot instantiate binary")
	}

	data, err := result.Raw()
	if err != nil {
		return errors.Wrap(err, "no raw data retrieved")
	}

	buildResult := buildv1.Build{}
	err = json.Unmarshal(data, &buildResult)
	if err != nil {
		return errors.Wrap(err, "cannot unmarshal instantiated binary response")
	}
	action.Log.Info("Submitted  Build for VDB " + vdb.ObjectMeta.Name)

	action.Log.Info("VDB state transition", "phase", target.Status.Phase)
	return action.client.Status().Update(ctx, target)
}
