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

	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
)

// NewUpdateAction creates a new initialize action
func NewUpdateAction() Action {
	return &updateAction{}
}

type updateAction struct {
	baseAction
}

// Name returns a common name of the action
func (action *updateAction) Name() string {
	return "UpdateAction"
}

// CanHandle tells whether this action can handle the virtualdatabase
func (action *updateAction) CanHandle(vdb *v1alpha1.VirtualDatabase) bool {
	return vdb.Status.Phase == v1alpha1.ReconcilerPhaseRunning
}

// Handle handles the virtualdatabase
func (action *updateAction) Handle(ctx context.Context, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) error {
	digest, err := ComputeForVirtualDatabase(vdb)
	if err != nil {
		return err
	}
	// when digest do not match restart the whole process, which will update the build
	if digest != vdb.Status.Digest {
		log.Infof("Changes detected in VDB %s, redeploying", vdb.ObjectMeta.Name)
		vdb.Status.Phase = v1alpha1.ReconcilerPhaseInitial
	}
	return nil
}
