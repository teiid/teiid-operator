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
	"reflect"

	"github.com/teiid/teiid-operator/pkg/apis/vdb/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/controller/virtualdatabase/constants"
	"github.com/teiid/teiid-operator/pkg/util/envvar"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
		// initialize with defaults
		vdb.Status.Phase = v1alpha1.ReconcilerPhaseS2IReady
		if err := action.init(ctx, vdb, r); err != nil {
			return err
		}
		vdb.Status.Digest = digest
	}
	return nil
}

func (action *initializeAction) init(ctx context.Context, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) error {

	if &vdb.Spec.Runtime == nil || !reflect.DeepEqual(vdb.Spec.Runtime, constants.SpringBootRuntime) {
		vdb.Spec.Runtime = constants.SpringBootRuntime
	}

	if vdb.Spec.Build.Incremental == nil {
		inc := true
		vdb.Spec.Build.Incremental = &inc
	}

	replicas := int32(1)
	if vdb.Spec.Replicas == nil {
		vdb.Spec.Replicas = &replicas
	}

	// environment variables
	defaultEnv := []corev1.EnvVar{
		{
			Name:  "JAVA_APP_DIR",
			Value: "/deployments",
		},
		{
			Name:  "JAVA_OPTIONS",
			Value: "-Djava.net.preferIPv4Stack=true -Duser.home=/tmp -Djava.net.preferIPv4Addresses=true",
		},
		{
			Name:  "JAVA_DEBUG",
			Value: "false",
		},
		{
			Name:  "AB_JMX_EXPORTER_CONFIG",
			Value: "/tmp/src/src/main/resources/prometheus-config.yml",
		},
		{
			Name: "NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
	}

	// merge/update env with user defined
	for _, v := range defaultEnv {
		if envvar.Get(vdb.Spec.Env, v.Name) == nil {
			envvar.SetVar(&vdb.Spec.Env, v)
		}
	}

	// resources for the container
	if &vdb.Spec.Resources == nil {
		vdb.Spec.Resources = corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				"memory": resource.MustParse("512Mi"),
				"cpu":    resource.MustParse("1.0"),
			},
			Requests: corev1.ResourceList{
				"memory": resource.MustParse("256Mi"),
				"cpu":    resource.MustParse("0.2"),
			},
		}
	}
	return nil
}
