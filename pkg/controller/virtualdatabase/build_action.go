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
	"github.com/teiid/teiid-operator/pkg/apis/vdb/v1alpha1"
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
	return "BuildAction"
}

// CanHandle tells whether this action can handle the virtualdatabase
func (action *buildAction) CanHandle(vdb *v1alpha1.VirtualDatabase) bool {
	return vdb.Status.Phase == v1alpha1.ReconcilerPhaseCodeGenerationCompleted || vdb.Status.Phase == v1alpha1.ReconcilerPhaseBuildImageSubmitted || vdb.Status.Phase == v1alpha1.ReconcilerPhaseBuildImageRunning
}

// Handle handles the virtualdatabase
func (action *buildAction) Handle(ctx context.Context, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) error {
	// better not changing the spec section of the target because it may be used for comparison by a
	// higher level controller
	if vdb.Status.Phase == v1alpha1.ReconcilerPhaseCodeGenerationCompleted {
		return submitForBuild(ctx, action, vdb, r)
	} else if vdb.Status.Phase == v1alpha1.ReconcilerPhaseBuildImageSubmitted || vdb.Status.Phase == v1alpha1.ReconcilerPhaseBuildImageRunning {
		return checkBuildProgress(ctx, action, vdb, r)
	}
	return nil
}

func checkBuildProgress(ctx context.Context, action *buildAction, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) error {
	target := vdb.DeepCopy()
	target.Status.Phase = v1alpha1.ReconcilerPhaseBuildImageRunning

	restClient, err := customclient.GetClientFor("build.openshift.io", "v1")
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
		target.Status.Phase = v1alpha1.ReconcilerPhaseBuildImageComplete
	} else if build.Status.Phase == buildv1.BuildPhaseCancelled ||
		build.Status.Phase == buildv1.BuildPhaseFailed ||
		build.Status.Phase == buildv1.BuildPhaseError {
		return errors.New("build failed")
	}

	log.Info("build finished", "phase", target.Status.Phase)
	return r.client.Status().Update(ctx, target)
}

func submitForBuild(ctx context.Context, action *buildAction, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) error {
	target := vdb.DeepCopy()
	target.Status.Phase = v1alpha1.ReconcilerPhaseBuildImageSubmitted

	buildStatus := ctx.Value(v1alpha1.BuildStatusKey).(v1alpha1.BuildStatus)
	archiveFile := buildStatus.TarFile

	log.Info("Starting Build")

	bc := s2i.CreateBuildConfiguration(vdb)
	err := r.client.Delete(ctx, &bc)
	if err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrap(err, "cannot delete build config")
	}

	err = r.client.Create(ctx, &bc)
	if err != nil {
		return errors.Wrap(err, "cannot create build config")
	}

	is := s2i.CreateImageStream(vdb)
	err = r.client.Delete(ctx, &is)
	if err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrap(err, "cannot delete image stream")
	}

	err = r.client.Create(ctx, &is)
	if err != nil {
		return errors.Wrap(err, "cannot create image stream")
	}

	resource, err := ioutil.ReadFile(archiveFile)
	if err != nil {
		return errors.Wrap(err, "cannot fully read tar file "+archiveFile)
	}

	restClient, err := customclient.GetClientFor("build.openshift.io", "v1")
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
	log.Info("Submitted  Build for VDB " + vdb.ObjectMeta.Name)

	log.Info("VDB state transition", "phase", target.Status.Phase)
	return r.client.Status().Update(ctx, target)
}

/*
buildStatus := &v1alpha1.BuildStatus{}
tempDir, err := ioutil.TempDir(os.TempDir(), "builder-")
if err != nil {
	log.Error(err, "Unexpected error while creating a temporary dir")
	return reconcile.Result{}, err
}
buildStatus.TarFile = tempDir + "vdb.tar"
defer os.RemoveAll(tempDir)
*/
