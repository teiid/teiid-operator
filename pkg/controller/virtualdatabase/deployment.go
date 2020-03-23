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

	obuildv1 "github.com/openshift/api/build/v1"
	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/controller/virtualdatabase/constants"
	"github.com/teiid/teiid-operator/pkg/util/proxy"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	return vdb.Status.Phase == v1alpha1.ReconcilerPhaseKeystoreCreated || vdb.Status.Phase == v1alpha1.ReconcilerPhaseDeploying ||
		vdb.Status.Phase == v1alpha1.ReconcilerPhaseRunning
}

// Handle handles the virtualdatabase
func (action *deploymentAction) Handle(ctx context.Context, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) error {

	if vdb.Status.Phase == v1alpha1.ReconcilerPhaseKeystoreCreated {
		log.Info("Running the deployment")
		bc, err := r.buildClient.BuildConfigs(vdb.ObjectMeta.Namespace).Get(vdb.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		existing, err := findDC(vdb, r)
		if err != nil {
			dc, err2 := action.buildDeployment(vdb, *bc, r)
			if err2 != nil {
				return err2
			}

			_, err = r.kubeClient.AppsV1().Deployments(vdb.ObjectMeta.Namespace).Create(&dc)
			//err = kubernetes.EnsureObject(&dc, err, r.client)
			if err != nil {
				return err
			}
		} else {
			// if a new image is created then update the deployment with it
			if existing.Spec.Template.Spec.Containers[0].Image != bc.Spec.Output.To.Name {
				existing.Spec.Template.Spec.Containers[0].Image = bc.Spec.Output.To.Name
				_, err = r.kubeClient.AppsV1().Deployments(vdb.ObjectMeta.Namespace).Update(existing)
				//err = r.client.Update(context.TODO(), existing)
				if err != nil {
					log.Warn("Failed to update object. ", err)
					return err
				}
			}
		}

		// change the status, needs to be done before next method.
		vdb.Status.Phase = v1alpha1.ReconcilerPhaseDeploying
		return nil
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

	deploymentEnvs := DeploymentEnvironments(vdb, r)

	update := false
	if *vdb.Spec.Replicas != *item.Spec.Replicas {
		item.Spec.Replicas = vdb.Spec.Replicas
		update = true
	}
	if !reflect.DeepEqual(deploymentEnvs, item.Spec.Template.Spec.Containers[0].Env) {
		item.Spec.Template.Spec.Containers[0].Env = deploymentEnvs
		update = true
	}

	if !update {
		// check to see if any of the secrets or configmaps changed
		configdigest, err := ComputeConfigDigest(ctx, r.client, vdb, deploymentEnvs)
		if err != nil {
			return err
		}

		if configdigest != vdb.Status.ConfigDigest {
			log.Info("ConfigMap or Secret has changed redeploying")
			update = true
			vdb.Status.ConfigDigest = configdigest
			item.Spec.Template.ObjectMeta.Annotations["configHash"] = configdigest
		}
	}

	if update {
		if err := r.client.Update(ctx, item); err != nil {
			return err
		}
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

func getTargetPort(port corev1.ContainerPort) intstr.IntOrString {
	p := int(port.ContainerPort)
	if p == 35443 {
		p = 5433
	} else if p == 35432 {
		p = 5432
	}
	return intstr.FromInt(p)
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

func containerPorts() []corev1.ContainerPort {
	ports := []corev1.ContainerPort{}
	ports = append(ports, corev1.ContainerPort{Name: "http", ContainerPort: int32(8080), Protocol: corev1.ProtocolTCP})
	ports = append(ports, corev1.ContainerPort{Name: "jolokia", ContainerPort: int32(8778), Protocol: corev1.ProtocolTCP})
	ports = append(ports, corev1.ContainerPort{Name: "prometheus", ContainerPort: int32(9779), Protocol: corev1.ProtocolTCP})
	ports = append(ports, corev1.ContainerPort{Name: "teiid", ContainerPort: int32(31000), Protocol: corev1.ProtocolTCP})
	ports = append(ports, corev1.ContainerPort{Name: "pg", ContainerPort: int32(35432), Protocol: corev1.ProtocolTCP})
	ports = append(ports, corev1.ContainerPort{Name: "teiid-secure", ContainerPort: int32(31443), Protocol: corev1.ProtocolTCP})
	ports = append(ports, corev1.ContainerPort{Name: "pg-secure", ContainerPort: int32(35443), Protocol: corev1.ProtocolTCP})
	return ports
}

func matchLabels(vdbName string) map[string]string {
	labels := map[string]string{
		"app":                      vdbName,
		"teiid.io/VirtualDatabase": vdbName,
		"teiid.io/type":            "VirtualDatabase",
	}
	return labels
}

// newDCForCR returns a BuildConfig with the same name/namespace as the cr
func (action *deploymentAction) buildDeployment(vdb *v1alpha1.VirtualDatabase, serviceBC obuildv1.BuildConfig,
	r *ReconcileVirtualDatabase) (appsv1.Deployment, error) {

	var probe *corev1.Probe
	matchLabels := matchLabels(vdb.ObjectMeta.Name)
	computingResources := constants.GetComputingResources(vdb)

	labels := map[string]string{
		"app":                      vdb.Name,
		"teiid.io/VirtualDatabase": vdb.ObjectMeta.Name,
		"teiid.io/type":            "VirtualDatabase",
	}

	// Add any custom labels
	for k := range constants.Config.Labels {
		labels[k] = constants.Config.Labels[k]
	}

	annotations := map[string]string{
		"configHash": vdb.Status.ConfigDigest,
	}

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

	// convert data source properties into ENV properties
	deploymentEnvs := DeploymentEnvironments(vdb, r)

	// Passing down cluster proxy config to Operands
	deploymentEnvs, _ = proxy.HTTPSettings(deploymentEnvs)

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
				MatchLabels: matchLabels,
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Name:        vdb.ObjectMeta.Name,
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            vdb.ObjectMeta.Name,
							Env:             deploymentEnvs,
							Resources:       computingResources,
							Image:           serviceBC.Spec.Output.To.Name,
							ImagePullPolicy: corev1.PullAlways,
							Ports:           containerPorts(),
							LivenessProbe:   probe,
							ReadinessProbe:  probe,
							WorkingDir:      "/deployments",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "keystore",
									ReadOnly:  true,
									MountPath: constants.KeystoreLocation,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "keystore",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: getKeystoreSecretName(vdb),
								},
							},
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
