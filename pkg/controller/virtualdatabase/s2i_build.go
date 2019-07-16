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

	"github.com/teiid/teiid-operator/pkg/apis/vdb/v1alpha1"
)

// NewS2IBuildAction creates a new initialize action
func NewS2IBuildAction() Action {
	return &s2iBuildAction{}
}

type s2iBuildAction struct {
	baseAction
}

// Name returns a common name of the action
func (action *s2iBuildAction) Name() string {
	return "S2iBuildAction"
}

// CanHandle tells whether this action can handle the virtualdatabase
func (action *s2iBuildAction) CanHandle(vdb *v1alpha1.VirtualDatabase) bool {
	return vdb.Status.Phase == v1alpha1.ReconcilerPhaseS2IReady
}

// Handle handles the virtualdatabase
func (action *s2iBuildAction) Handle(ctx context.Context, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) error {

	// better not changing the spec section of the target because it may be used for comparison by a
	// higher level controller

	target := vdb.DeepCopy()

	target.Status.Phase = v1alpha1.ReconcilerPhaseCodeGeneration

	log.Info("VDB state transition", "phase", target.Status.Phase)

	return r.client.Status().Update(ctx, target)
}
