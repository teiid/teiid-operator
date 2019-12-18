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
	"reflect"
	"time"

	oappsv1 "github.com/openshift/api/apps/v1"
	obuildv1 "github.com/openshift/api/build/v1"
	oroutev1 "github.com/openshift/api/route/v1"
	"github.com/teiid/teiid-operator/pkg/apis/vdb/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// NewDeploymentAction creates a new initialize action
func NewDeploymentAction() Action {
	return &deploymentAction{}
}

type deploymentAction struct {
	baseAction
}

// Name returns a common name of the action
func (action *deploymentAction) Name() string {
	return "DeploymentAction"
}

// CanHandle tells whether this action can handle the virtualdatabase
func (action *deploymentAction) CanHandle(vdb *v1alpha1.VirtualDatabase) bool {
	return vdb.Status.Phase == v1alpha1.ReconcilerPhaseServiceImageFinished || vdb.Status.Phase == v1alpha1.ReconcilerPhaseDeploying ||
		vdb.Status.Phase == v1alpha1.ReconcilerPhaseRunning
}

// Handle handles the virtualdatabase
func (action *deploymentAction) Handle(ctx context.Context, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) error {

	if vdb.Status.Phase == v1alpha1.ReconcilerPhaseServiceImageFinished {
		vdb.Status.Phase = v1alpha1.ReconcilerPhaseDeploying
		log.Info("Running the deployment")

		bc, err := r.buildClient.BuildConfigs(vdb.ObjectMeta.Namespace).Get(vdb.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		// check to see if the user configured a secret with certificates
		// if not have generate self signed
		hasCertSecret := false
		_, err = findSecret(vdb, r)
		if err == nil {
			hasCertSecret = true
		}

		dc, err := action.deploymentConfig(vdb, *bc, r)
		if err != nil {
			return err
		}

		existing, err := action.findDC(vdb, r)
		if existing == nil {
			err = errors.NewNotFound(schema.GroupResource{Group: "dc", Resource: "dc"}, vdb.ObjectMeta.Name)
		}
		err = action.ensureObj(&dc, err, r)
		if err != nil {
			return err
		}

		// Create the service and route needed. We are creating service
		// before so that we can use the service annotation to create certificate
		// that can be loaded in a deployment
		if len(existing.Spec.Template.Spec.Containers[0].Ports) != 0 {
			service, err := action.createService(*existing, vdb, r, hasCertSecret)
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
	} else if vdb.Status.Phase == v1alpha1.ReconcilerPhaseDeploying {
		item, _ := action.findDC(vdb, r)
		if item != nil && action.isDeploymentInReadyState(*item) {
			log.Info("Deployment finished:" + vdb.ObjectMeta.Name)
			vdb.Status.Phase = v1alpha1.ReconcilerPhaseRunning
		} else if item != nil && !action.isDeploymentProgressing(*item) {
			log.Info("Deployment Failed:" + vdb.ObjectMeta.Name)
			vdb.Status.Phase = v1alpha1.ReconcilerPhaseError
		}
	} else if vdb.Status.Phase == v1alpha1.ReconcilerPhaseRunning {
		item, _ := action.findDC(vdb, r)
		if item != nil && action.isDeploymentInReadyState(*item) {
			if *vdb.Spec.Replicas != item.Spec.Replicas {
				item.Spec.Replicas = *vdb.Spec.Replicas
			}
			if !reflect.DeepEqual(vdb.Spec.Env, item.Spec.Template.Spec.Containers[0].Env) {
				item.Spec.Template.Spec.Containers[0].Env = vdb.Spec.Env
			}
			if err := r.client.Update(ctx, item); err != nil {
				return err
			}
		}
	}
	return nil
}

func (action *deploymentAction) findDC(vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) (*oappsv1.DeploymentConfig, error) {
	listOpts := []client.ListOption{
		client.InNamespace(vdb.ObjectMeta.Namespace),
	}

	list := &oappsv1.DeploymentConfigList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DeploymentConfig",
			APIVersion: "apps.openshift.io/v1",
		},
	}

	err := r.client.List(context.TODO(), list, listOpts...)
	if err == nil {
		for _, item := range list.Items {
			if item.Name == vdb.ObjectMeta.Name {
				return &item, nil
			}
		}
	}
	return nil, err
}

func findSecret(vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) (*corev1.Secret, error) {
	obj := corev1.Secret{}
	key := client.ObjectKey{Namespace: vdb.ObjectMeta.Namespace, Name: vdb.ObjectMeta.Name}
	err := r.client.Get(context.TODO(), key, &obj)
	return &obj, err
}

func getTargetPort(port corev1.ContainerPort) intstr.IntOrString {
	p := int(port.ContainerPort)
	if p == 35443 {
		p = 5433
	} else if p == 35432 {
		p = 5432
	}
	return intstr.FromInt(p)
}

func (action *deploymentAction) createService(dc oappsv1.DeploymentConfig, vdb *v1alpha1.VirtualDatabase,
	r *ReconcileVirtualDatabase, hasCertSecret bool) (corev1.Service, error) {

	servicePorts := []corev1.ServicePort{}
	for _, port := range dc.Spec.Template.Spec.Containers[0].Ports {
		servicePorts = append(servicePorts, corev1.ServicePort{
			Name:       port.Name,
			Protocol:   port.Protocol,
			Port:       port.ContainerPort,
			TargetPort: getTargetPort(port),
		},
		)
	}

	labels := map[string]string{
		"discovery.3scale.net": "true",
	}

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
			Selector:        dc.Spec.Selector,
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
	err = action.ensureObj(&service,
		r.client.Get(context.TODO(), types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, &corev1.Service{}), r)
	if err != nil {
		return corev1.Service{}, err
	}
	return service, nil
}

func (action *deploymentAction) createRoute(service corev1.Service, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) (oroutev1.Route, error) {
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
		err = action.ensureObj(&route, err, r)
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

func (action *deploymentAction) isDeploymentInReadyState(dc oappsv1.DeploymentConfig) bool {
	if len(dc.Status.Conditions) > 0 {
		for _, condition := range dc.Status.Conditions {
			if condition.Type == oappsv1.DeploymentAvailable && condition.Status == corev1.ConditionTrue {
				return true
			}
		}
	}
	return false
}

func (action *deploymentAction) isDeploymentProgressing(dc oappsv1.DeploymentConfig) bool {
	if len(dc.Status.Conditions) > 0 {
		for _, condition := range dc.Status.Conditions {
			if condition.Type == oappsv1.DeploymentProgressing && condition.Status == corev1.ConditionTrue {
				return true
			}
		}
		// this is one way I found when progression will stop
		if dc.Status.ObservedGeneration < 4 {
			return true
		}
		return false
	}
	return true
}

// newDCForCR returns a BuildConfig with the same name/namespace as the cr
func (action *deploymentAction) deploymentConfig(vdb *v1alpha1.VirtualDatabase, serviceBC obuildv1.BuildConfig,
	r *ReconcileVirtualDatabase) (oappsv1.DeploymentConfig, error) {

	var probe *corev1.Probe
	labels := map[string]string{
		"app":              vdb.Name,
		"syndesis.io/type": "datavirtualization",
	}

	ports := []corev1.ContainerPort{}
	ports = append(ports, corev1.ContainerPort{Name: "http", ContainerPort: int32(8080), Protocol: corev1.ProtocolTCP})
	ports = append(ports, corev1.ContainerPort{Name: "jolokia", ContainerPort: int32(8778), Protocol: corev1.ProtocolTCP})
	ports = append(ports, corev1.ContainerPort{Name: "prometheus", ContainerPort: int32(9779), Protocol: corev1.ProtocolTCP})
	ports = append(ports, corev1.ContainerPort{Name: "teiid", ContainerPort: int32(31000), Protocol: corev1.ProtocolTCP})
	ports = append(ports, corev1.ContainerPort{Name: "pg", ContainerPort: int32(35432), Protocol: corev1.ProtocolTCP})
	ports = append(ports, corev1.ContainerPort{Name: "teiid-secure", ContainerPort: int32(31443), Protocol: corev1.ProtocolTCP})
	ports = append(ports, corev1.ContainerPort{Name: "pg-secure", ContainerPort: int32(35443), Protocol: corev1.ProtocolTCP})

	// liveness and readiness probes
	probe = &corev1.Probe{
		TimeoutSeconds:      int32(5),
		PeriodSeconds:       int32(20),
		SuccessThreshold:    int32(1),
		FailureThreshold:    int32(3),
		InitialDelaySeconds: int32(60),
	}
	probe.Handler.HTTPGet = &corev1.HTTPGetAction{
		Path: "/actuator/health",
		Port: intstr.FromInt(8080),
	}

	dc := oappsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vdb.ObjectMeta.Name,
			Namespace: vdb.Namespace,
			Labels:    labels,
		},
		Spec: oappsv1.DeploymentConfigSpec{
			Replicas: *vdb.Spec.Replicas,
			Selector: labels,
			Strategy: oappsv1.DeploymentStrategy{
				Type: oappsv1.DeploymentStrategyTypeRolling,
			},
			Template: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					Name:   vdb.ObjectMeta.Name,
					Annotations: map[string]string{
						"prometheus.io/scrape": "true",
						"prometheus.io/port":   "9779",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            vdb.ObjectMeta.Name,
							Env:             vdb.Spec.Env,
							Resources:       vdb.Spec.Resources,
							Image:           serviceBC.Spec.Output.To.Name,
							ImagePullPolicy: corev1.PullAlways,
							Ports:           ports,
							LivenessProbe:   probe,
							ReadinessProbe:  probe,
							WorkingDir:      "/deployments",
						},
					},
				},
			},
			Triggers: oappsv1.DeploymentTriggerPolicies{
				{Type: oappsv1.DeploymentTriggerOnConfigChange},
				{
					Type: oappsv1.DeploymentTriggerOnImageChange,
					ImageChangeParams: &oappsv1.DeploymentTriggerImageChangeParams{
						Automatic:      true,
						ContainerNames: []string{vdb.ObjectMeta.Name},
						From:           *serviceBC.Spec.Output.To,
					},
				},
			},
		},
	}

	dc.SetGroupVersionKind(oappsv1.SchemeGroupVersion.WithKind("DeploymentConfig"))
	var err = controllerutil.SetControllerReference(vdb, &dc, r.scheme)
	if err != nil {
		log.Error(err)
		return oappsv1.DeploymentConfig{}, err
	}

	return dc, nil
}

// ensureObj creates an object based on the error passed in from a `client.Get`
func (action *deploymentAction) ensureObj(obj v1alpha1.OpenShiftObject, err error, r *ReconcileVirtualDatabase) error {
	log := log.With("kind", obj.GetObjectKind().GroupVersionKind().Kind, "name", obj.GetName(), "namespace", obj.GetNamespace())

	if err != nil && errors.IsNotFound(err) {
		// Define a new Object
		log.Info("Creating")
		err = r.client.Create(context.TODO(), obj)
		if err != nil {
			log.Warn("Failed to create object. ", err)
			return err
		}
		// Object created successfully - return and requeue
		return nil
	} else if err != nil {
		log.Error("Failed to get object. ", err)
		return err
	}
	log.Debug("Skip reconcile - object already exists")
	return nil
}
