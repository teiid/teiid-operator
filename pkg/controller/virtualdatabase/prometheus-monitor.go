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

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/teiid/teiid-operator/pkg/apis/vdb/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/controller/virtualdatabase/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// NewPrometheusMonitorAction creates a new initialize action
func NewPrometheusMonitorAction() Action {
	return &prometheusMonitorAction{}
}

type prometheusMonitorAction struct {
	baseAction
}

// Name returns a common name of the action
func (action *prometheusMonitorAction) Name() string {
	return "PrometheusMonitorAction"
}

// CanHandle tells whether this action can handle the virtualdatabase
func (action *prometheusMonitorAction) CanHandle(vdb *v1alpha1.VirtualDatabase) bool {
	return vdb.Status.Phase == v1alpha1.ReconcilerPhaseRunning
}

// Handle handles the virtualdatabase
func (action *prometheusMonitorAction) Handle(ctx context.Context, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) error {
	if action.hasPrometheus(ctx, vdb, r) {
		if !action.hasServiceMonitor(vdb, r) {
			log.Info("Found Prometheus instance, creating a Service Monitor")
			err := action.createServiceMonitor(ctx, vdb, r)
			if err != nil {
				return err
			}
		}

	} else {
		log.Debug("Prometheus instance not found, skipping the metrics push to Prometheus")
	}
	return nil
}

func (action *prometheusMonitorAction) hasPrometheus(ctx context.Context, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) bool {
	list, err := r.prometheusClient.Prometheuses(vdb.ObjectMeta.Namespace).List(metav1.ListOptions{})
	if err == nil {
		for _, item := range list.Items {
			labels := item.Spec.ServiceMonitorSelector
			if labels != nil {
				for k, v := range constants.Config.Prometheus.MatchLabels {
					if labels.MatchLabels[k] == v {
						return true
					}
				}
			}
		}
	}
	return false
}

func (action *prometheusMonitorAction) hasServiceMonitor(vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) bool {
	list, err := r.prometheusClient.ServiceMonitors(vdb.ObjectMeta.Namespace).List(metav1.ListOptions{})
	if err == nil {
		for _, item := range list.Items {
			if item.ObjectMeta.Name == vdb.ObjectMeta.Name {
				return true
			}
		}
	}
	return false
}

func (action *prometheusMonitorAction) createServiceMonitor(ctx context.Context, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) error {
	monitor := monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceMonitor",
			APIVersion: "monitoring.coreos.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      vdb.ObjectMeta.Name,
			Namespace: vdb.ObjectMeta.Namespace,
			Labels:    constants.Config.Prometheus.MatchLabels,
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"teiid.io/VirtualDatabase": vdb.ObjectMeta.Name,
					"app":                      vdb.ObjectMeta.Name,
				},
			},
			Endpoints: []monitoringv1.Endpoint{
				{
					Port: "prometheus",
				},
			},
		},
	}

	// set the object reference to vdb
	if err := controllerutil.SetControllerReference(vdb, &monitor, r.scheme); err != nil {
		log.Error(err)
	}

	if err := r.client.Create(context.TODO(), &monitor); err != nil {
		return err
	}
	return nil
}
