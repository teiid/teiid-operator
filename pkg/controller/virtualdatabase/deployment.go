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

	obuildv1 "github.com/openshift/api/build/v1"
	oroutev1 "github.com/openshift/api/route/v1"
	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/controller/virtualdatabase/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

		existing, err := findDC(vdb, r)
		if err != nil {
			err = action.ensureObj(&dc, err, r)
			if err != nil {
				return err
			}
		} else {
			// if a new image is created then update the deployment with it
			if existing.Spec.Template.Spec.Containers[0].Image != bc.Spec.Output.To.Name {
				dc.Spec.Template.Spec.Containers[0].Image = bc.Spec.Output.To.Name
			}
			err = r.client.Update(context.TODO(), &dc)
			if err != nil {
				log.Warn("Failed to update object. ", err)
				return err
			}
		}

		existing, err = findDC(vdb, r)
		if err != nil {
			// wait until the dc is found, no need report error
			log.Debug(err)
			return nil
		}
		// Create the service and route needed. We are creating service
		// before so that we can use the service annotation to create certificate
		// that can be loaded in a deployment
		_, err = findService(vdb, r)
		if err != nil {
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
		// change the status
		vdb.Status.Phase = v1alpha1.ReconcilerPhaseDeploying
	} else if vdb.Status.Phase == v1alpha1.ReconcilerPhaseDeploying {
		item, _ := findDC(vdb, r)
		if item != nil && action.isDeploymentInReadyState(*item) {
			log.Info("Deployment finished:" + vdb.ObjectMeta.Name)
			vdb.Status.Phase = v1alpha1.ReconcilerPhaseRunning
		} else if item != nil && !action.isDeploymentProgressing(*item) {
			log.Info("Deployment Failed:" + vdb.ObjectMeta.Name)
			vdb.Status.Phase = v1alpha1.ReconcilerPhaseError
		}
	} else if vdb.Status.Phase == v1alpha1.ReconcilerPhaseRunning {
		item, _ := findDC(vdb, r)
		if item != nil && action.isDeploymentInReadyState(*item) {
			err := action.ensureReplicas(ctx, vdb, item, r)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (action *deploymentAction) ensureReplicas(ctx context.Context, vdb *v1alpha1.VirtualDatabase,
	item *appsv1.Deployment, r *ReconcileVirtualDatabase) error {

	if vdb.Spec.Replicas != item.Spec.Replicas {
		item.Spec.Replicas = vdb.Spec.Replicas
	}
	if !reflect.DeepEqual(vdb.Spec.Env, item.Spec.Template.Spec.Containers[0].Env) {
		item.Spec.Template.Spec.Containers[0].Env = vdb.Spec.Env
	}
	if err := r.client.Update(ctx, item); err != nil {
		return err
	}
	return nil
}

func findDC(vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) (*appsv1.Deployment, error) {
	obj := appsv1.Deployment{}
	key := client.ObjectKey{Namespace: vdb.ObjectMeta.Namespace, Name: vdb.ObjectMeta.Name}
	err := r.client.Get(context.TODO(), key, &obj)
	return &obj, err
}

func findSecret(vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) (*corev1.Secret, error) {
	obj := corev1.Secret{}
	key := client.ObjectKey{Namespace: vdb.ObjectMeta.Namespace, Name: vdb.ObjectMeta.Name}
	err := r.client.Get(context.TODO(), key, &obj)
	return &obj, err
}

func findService(vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) (*corev1.Service, error) {
	obj := corev1.Service{}
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

func (action *deploymentAction) createService(dc appsv1.Deployment, vdb *v1alpha1.VirtualDatabase,
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
		"discovery.3scale.net":        "true",
		"teiid.io/VirtualDatabase":    vdb.ObjectMeta.Name,
		"app":                         vdb.ObjectMeta.Name,
		"teiid.io/deployment-version": vdb.Status.Version,
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
			Selector:        dc.Spec.Selector.MatchLabels,
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

func (action *deploymentAction) isDeploymentInReadyState(dc appsv1.Deployment) bool {
	if len(dc.Status.Conditions) > 0 {
		for _, condition := range dc.Status.Conditions {
			if condition.Type == appsv1.DeploymentAvailable && condition.Status == corev1.ConditionTrue {
				return true
			}
		}
	}
	return false
}

func (action *deploymentAction) isDeploymentProgressing(dc appsv1.Deployment) bool {
	if len(dc.Status.Conditions) > 0 {
		for _, condition := range dc.Status.Conditions {
			if condition.Type == appsv1.DeploymentProgressing && condition.Status == corev1.ConditionTrue {
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
	r *ReconcileVirtualDatabase) (appsv1.Deployment, error) {

	var probe *corev1.Probe
	labels := map[string]string{
		"app":                      vdb.Name,
		"teiid.io/VirtualDatabase": vdb.ObjectMeta.Name,
		"teiid.io/type":            "VirtualDatabase",
	}
	// Add any custom labels
	for k := range constants.Config.Labels {
		labels[k] = constants.Config.Labels[k]
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
		FailureThreshold:    int32(5),
		InitialDelaySeconds: int32(15),
	}
	probe.Handler.HTTPGet = &corev1.HTTPGetAction{
		Path: "/actuator/health",
		Port: intstr.FromInt(8080),
	}

	dc := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        vdb.ObjectMeta.Name,
			Namespace:   vdb.Namespace,
			Labels:      labels,
			Annotations: make(map[string]string),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: vdb.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
			},
			Template: corev1.PodTemplateSpec{
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
		},
	}

	// Inject Jaeger agent as side car into the deployment
	if vdb.Spec.Jaeger != "" && r.jaegerClient.Jaegers(vdb.ObjectMeta.Namespace).HasJaeger(vdb.Spec.Jaeger) {
		dc.ObjectMeta.Annotations["sidecar.jaegertracing.io/inject"] = vdb.Spec.Jaeger
	}

	dc.SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("Deployment"))
	var err = controllerutil.SetControllerReference(vdb, &dc, r.scheme)
	if err != nil {
		log.Error(err)
		return appsv1.Deployment{}, err
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
