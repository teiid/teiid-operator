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
	"time"

	oroutev1 "github.com/openshift/api/route/v1"
	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/util/kubernetes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// NewCreateServiceAction creates a new initialize action
func NewCreateServiceAction() Action {
	return &createServiceAction{}
}

type createServiceAction struct {
	baseAction
}

// Name returns a common name of the action
func (action *createServiceAction) Name() string {
	return "createServiceAction"
}

// CanHandle tells whether this action can handle the virtualdatabase
func (action *createServiceAction) CanHandle(vdb *v1alpha1.VirtualDatabase) bool {
	return vdb.Status.Phase == v1alpha1.ReconcilerPhaseServiceImageFinished
}

// Handle handles the virtualdatabase
func (action *createServiceAction) Handle(ctx context.Context, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) error {
	// check to see if the user configured a secret with certificates
	// if not have generate self signed
	hasCertSecret := false
	_, err := findSecret(vdb, r)
	if err == nil {
		hasCertSecret = true
	}

	// Create the service and route needed. We are creating service
	// before so that we can use the service annotation to create certificate
	// that can be loaded in a deployment
	_, err = kubernetes.GetService(ctx, r.client, vdb.ObjectMeta.Name, vdb.ObjectMeta.Namespace)
	if err != nil {
		service, err := action.createService(vdb, r, hasCertSecret)
		if err != nil {
			vdb.Status.Phase = v1alpha1.ReconcilerPhaseError
			vdb.Status.Failure = "Failed to create Service"
		} else {
			log.Info("Services created:" + vdb.ObjectMeta.Name)
			if vdb.Spec.ExposeVia3Scale {
				log.Info("creation of Route skipped as it is configured to be exposed through 3scale")
			} else {
				route, err := action.createRoute(service, vdb, r)
				if err != nil {
					vdb.Status.Phase = v1alpha1.ReconcilerPhaseError
					vdb.Status.Failure = "Failed to create route"
				} else {
					log.Info("Route created:" + vdb.ObjectMeta.Name)
					vdb.Status.Route = fmt.Sprintf("https://%s/odata", route.Spec.Host)
				}
			}
		}
	}
	vdb.Status.Phase = v1alpha1.ReconcilerPhaseServiceCreated
	return nil
}

func (action *createServiceAction) createService(vdb *v1alpha1.VirtualDatabase,
	r *ReconcileVirtualDatabase, hasCertSecret bool) (corev1.Service, error) {

	servicePorts := []corev1.ServicePort{}
	for _, port := range containerPorts() {
		servicePorts = append(servicePorts, corev1.ServicePort{
			Name:       port.Name,
			Protocol:   port.Protocol,
			Port:       port.ContainerPort,
			TargetPort: getTargetPort(port),
		},
		)
	}

	labels := map[string]string{
		"app":                         vdb.ObjectMeta.Name,
		"discovery.3scale.net":        "true",
		"teiid.io/VirtualDatabase":    vdb.ObjectMeta.Name,
		"teiid.io/type":               "VirtualDatabase",
		"teiid.io/deployment-version": vdb.Status.Version,
	}

	matchLables := matchLabels(vdb)

	// if openapi is in use then use the openapi for it
	apiLink := "/odata/openapi.json"
	if len(vdb.Spec.Build.Source.OpenAPI) > 0 {
		apiLink = "/openapi.json"
	}

	annotations := map[string]string{
		"discovery.3scale.net/scheme":           "http",
		"discovery.3scale.net/port":             "8080",
		"discovery.3scale.net/description-path": apiLink,
	}

	// if there is no secret certificate then annotate to create one
	if !hasCertSecret {
		annotations["service.alpha.openshift.io/serving-cert-secret-name"] = vdb.ObjectMeta.Name
	}

	meta := metav1.ObjectMeta{
		Name:        vdb.ObjectMeta.Name,
		Namespace:   vdb.Namespace,
		Labels:      labels,
		Annotations: annotations,
	}
	timeout := int32(86400)
	service := corev1.Service{
		ObjectMeta: meta,
		Spec: corev1.ServiceSpec{
			Selector:        matchLables,
			Type:            corev1.ServiceTypeClusterIP,
			Ports:           servicePorts,
			SessionAffinity: corev1.ServiceAffinityClientIP,
			SessionAffinityConfig: &corev1.SessionAffinityConfig{
				ClientIP: &corev1.ClientIPConfig{
					TimeoutSeconds: &timeout,
				},
			},
		},
	}
	service.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Service"))
	err := controllerutil.SetControllerReference(vdb, &service, r.scheme)
	if err != nil {
		log.Error(err)
	}

	service.ResourceVersion = ""
	err = kubernetes.EnsureObject(&service,
		r.client.Get(context.TODO(), types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, &corev1.Service{}), r.client)
	if err != nil {
		return corev1.Service{}, err
	}
	return service, nil
}

func (action *createServiceAction) createRoute(service corev1.Service, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) (oroutev1.Route, error) {
	metadata := service.ObjectMeta.DeepCopy()
	metadata.Labels["teiid.io/api"] = "odata"
	route := oroutev1.Route{
		ObjectMeta: *metadata,
		Spec: oroutev1.RouteSpec{
			Port: &oroutev1.RoutePort{
				TargetPort: intstr.FromInt(8080),
			},
			To: oroutev1.RouteTargetReference{
				Kind: "Service",
				Name: service.Name,
			},
			TLS: &oroutev1.TLSConfig{
				Termination: oroutev1.TLSTerminationEdge,
			},
		},
	}
	route.SetGroupVersionKind(oroutev1.SchemeGroupVersion.WithKind("Route"))
	err := controllerutil.SetControllerReference(vdb, &route, r.scheme)
	if err != nil {
		log.Error("Error setting controller reference. ", err)
	}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: route.Name, Namespace: route.Namespace}, &oroutev1.Route{})
	if err != nil && errors.IsNotFound(err) {
		route.ResourceVersion = ""
		err = kubernetes.EnsureObject(&route, err, r.client)
		if err != nil {
			log.Error("Error creating Route. ", err)
		}
	}

	// wait until route is created
	found := &oroutev1.Route{}
	for i := 1; i < 60; i++ {
		time.Sleep(time.Duration(100) * time.Millisecond)
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: route.Name, Namespace: route.Namespace}, found)
		if err == nil {
			break
		}
	}
	return *found, err
}
