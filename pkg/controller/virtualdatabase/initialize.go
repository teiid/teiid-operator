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
	"github.com/teiid/teiid-operator/pkg/util/kubernetes"
	"github.com/teiid/teiid-operator/pkg/util/maven"
)

// NewInitializeAction creates a new initialize action
func NewInitializeAction() Action {
	return &initializeAction{}
}

type initializeAction struct {
	baseAction
}

// Name returns a common name of the action
func (action *initializeAction) Name() string {
	return "initialize"
}

// CanHandle tells whether this action can handle the virtualdatabase
func (action *initializeAction) CanHandle(vdb *v1alpha1.VirtualDatabase) bool {
	return vdb.Status.Phase == v1alpha1.ReconcilerPhaseInitial
}

// Handle handles the virtualdatabase
func (action *initializeAction) Handle(ctx context.Context, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) error {
	// build digest the vdb/config contents
	digest, err := ComputeForVirtualDatabase(vdb)
	if err != nil {
		return err
	}

	if &vdb.Status.Phase == nil || vdb.Status.Phase == v1alpha1.ReconcilerPhaseInitial {

		// make sure all env properties exist before proceeding
		if !kubernetes.EnvironmentPropertiesExists(ctx, r.client, vdb.ObjectMeta.Namespace, vdb.Spec.Env) {
			vdb.Status.Failure = "Configuration missing, make sure to supply all the ConfigMaps and Secrets required"
			return nil
		}

		// make sure all the data source properties exist
		for _, ds := range vdb.Spec.DataSources {
			if !kubernetes.EnvironmentPropertiesExists(ctx, r.client, vdb.ObjectMeta.Namespace, ds.Properties) {
				vdb.Status.Failure = "Configuration missing, make sure to supply all the ConfigMaps and Secrets required"
				return nil
			}
		}

		// initialize with defaults
		vdb.Status.Failure = ""
		vdb.Status.Phase = v1alpha1.ReconcilerPhaseCreateCacheStore
		if err := action.init(ctx, vdb, r); err != nil {
			return err
		}
		vdb.Status.Digest = digest
		vdb.Status.ConfigDigest, err = ComputeConfigDigest(ctx, r.client, vdb)
		if err != nil {
			return err
		}
	}
	return nil
}

func (action *initializeAction) init(ctx context.Context, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) error {

	replicas := int32(1)
	if vdb.Spec.Replicas == nil {
		vdb.Spec.Replicas = &replicas
	}

	// set the VDB version for the deployment
	if vdb.Spec.Build.Source.Version == "" && vdb.Spec.Build.Source.DDL != "" && vdb.Status.Version == "" {
		vdb.Status.Version = "1"
	}

	if vdb.Spec.Build.Source.Maven != "" {
		dep, err := maven.ParseGAV(vdb.Spec.Build.Source.Maven)
		if err == nil {
			vdb.Status.Version = dep.Version
		}
	}
	return nil
}
