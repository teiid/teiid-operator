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
	"io/ioutil"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	oappsv1 "github.com/openshift/api/apps/v1"
	obuildv1 "github.com/openshift/api/build/v1"
	oimagev1 "github.com/openshift/api/image/v1"
	oroutev1 "github.com/openshift/api/route/v1"
	scheme "github.com/openshift/client-go/build/clientset/versioned/scheme"
	buildv1client "github.com/openshift/client-go/build/clientset/versioned/typed/build/v1"
	imagev1 "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	"github.com/teiid/teiid-operator/pkg/apis/vdb/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/controller/virtualdatabase/constants"
	"github.com/teiid/teiid-operator/pkg/controller/virtualdatabase/logs"
	"github.com/teiid/teiid-operator/pkg/controller/virtualdatabase/shared"
	"github.com/teiid/teiid-operator/pkg/controller/virtualdatabase/status"
	"github.com/teiid/teiid-operator/pkg/util/envvar"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	cachev1 "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var log = logs.GetLogger("virtualdatabase")
var _ reconcile.Reconciler = &ReconcileVirtualDatabase{}

// ReconcileVirtualDatabase reconciles a VirtualDatabase object
type ReconcileVirtualDatabase struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client      client.Client
	scheme      *runtime.Scheme
	cache       cachev1.Cache
	imageClient *imagev1.ImageV1Client
	buildClient *buildv1client.BuildV1Client
}

// Reconcile reads that state of the cluster for a VirtualDatabase object and makes changes based on the state read
// and what is in the VirtualDatabase.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileVirtualDatabase) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// log.Info("Reconciling VirtualDatabase")

	// Fetch the VirtualDatabase instance
	instance := &v1alpha1.VirtualDatabase{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Set some CR defaults
	if instance.Spec.Runtime == "" || instance.Spec.Runtime != v1alpha1.SpringbootRuntimeType {
		instance.Spec.Runtime = v1alpha1.SpringbootRuntimeType
	}
	if instance.Spec.Build.Incremental == nil {
		inc := true
		instance.Spec.Build.Incremental = &inc
	}

	// Define new BuildConfig objects
	buildConfigs := newBCsForCR(instance)
	for imageType, buildConfig := range buildConfigs {
		var setOwner bool
		// set ownerreference for service BC only
		if imageType == "service" {
			setOwner = true
			err := controllerutil.SetControllerReference(instance, &buildConfig, r.scheme)
			if err != nil {
				log.Error(err)
			}
		}
		if _, err := r.ensureImageStream(buildConfig.Name, instance, setOwner); err != nil {
			return reconcile.Result{}, err
		}

		// Check if this BC already exists
		bc, err := r.buildClient.BuildConfigs(buildConfig.Namespace).Get(buildConfig.Name, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			log.Info("Creating a new BuildConfig ", buildConfig.Name, " in namespace ", buildConfig.Namespace)
			bc, err = r.buildClient.BuildConfigs(buildConfig.Namespace).Create(&buildConfig)
			if err != nil {
				return reconcile.Result{}, err
			}
		} else if err != nil {
			return reconcile.Result{}, err
		}

		// Trigger first build of "builder" and binary BCs
		if (imageType == "builder" || bc.Spec.Source.Type == obuildv1.BuildSourceBinary) && bc.Status.LastVersion == 0 {
			if err = r.triggerBuild(*bc, instance); err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	// Create new DeploymentConfig object
	depConfig, err := r.newDCForCR(instance, buildConfigs["service"])
	if err != nil {
		return reconcile.Result{}, err
	}
	rResult, err := r.createObj(
		&depConfig,
		r.client.Get(context.TODO(), types.NamespacedName{Name: depConfig.Name, Namespace: depConfig.Namespace}, &oappsv1.DeploymentConfig{}),
	)
	if err != nil {
		return rResult, err
	}

	dcUpdated, err := r.updateDeploymentConfigs(instance, depConfig)
	if err != nil {
		return reconcile.Result{}, err
	}
	if dcUpdated && status.SetProvisioning(instance) {
		return r.UpdateObj(instance)
	}

	// Expose DC with service and route
	serviceRoute := ""
	if len(depConfig.Spec.Template.Spec.Containers[0].Ports) != 0 {
		servicePorts := []corev1.ServicePort{}
		for _, port := range depConfig.Spec.Template.Spec.Containers[0].Ports {
			servicePorts = append(servicePorts, corev1.ServicePort{
				Name:       port.Name,
				Protocol:   port.Protocol,
				Port:       port.ContainerPort,
				TargetPort: intstr.FromInt(int(port.ContainerPort)),
			},
			)
		}
		service := corev1.Service{
			ObjectMeta: depConfig.ObjectMeta,
			Spec: corev1.ServiceSpec{
				Selector: depConfig.Spec.Selector,
				Type:     corev1.ServiceTypeClusterIP,
				Ports:    servicePorts,
			},
		}
		service.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Service"))
		err = controllerutil.SetControllerReference(instance, &service, r.scheme)
		if err != nil {
			log.Error(err)
		}

		service.ResourceVersion = ""
		rResult, err := r.createObj(
			&service,
			r.client.Get(context.TODO(), types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, &corev1.Service{}),
		)
		if err != nil {
			return rResult, err
		}

		// Create route
		rt := oroutev1.Route{
			ObjectMeta: service.ObjectMeta,
			Spec: oroutev1.RouteSpec{
				Port: &oroutev1.RoutePort{
					TargetPort: intstr.FromInt(8080),
				},
				To: oroutev1.RouteTargetReference{
					Kind: "Service",
					Name: service.Name,
				},
			},
		}
		if serviceRoute = r.GetRouteHost(rt, instance); serviceRoute != "" {
			instance.Status.Route = fmt.Sprintf("http://%s", serviceRoute)
		}
	}

	/*

		bcUpdated, err := r.updateBuildConfigs(instance, buildConfig)
		if err != nil {
			return reconcile.Result{}, err
		}
		if bcUpdated && status.SetProvisioning(instance) {
			return r.UpdateObj(instance)
		}
	*/

	// Fetch the cached VirtualDatabase instance
	cachedInstance := &v1alpha1.VirtualDatabase{}
	err = r.cache.Get(context.TODO(), request.NamespacedName, cachedInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		r.setFailedStatus(instance, v1alpha1.UnknownReason, err)
		return reconcile.Result{}, err
	}

	// Update CR if needed
	if r.hasSpecChanges(instance, cachedInstance) {
		if status.SetProvisioning(instance) && instance.ResourceVersion == cachedInstance.ResourceVersion {
			return r.UpdateObj(instance)
		}
		return reconcile.Result{Requeue: true}, nil
	}
	if r.hasStatusChanges(instance, cachedInstance) {
		if instance.ResourceVersion == cachedInstance.ResourceVersion {
			return r.UpdateObj(instance)
		}
		return reconcile.Result{Requeue: true}, nil
	}
	if status.SetDeployed(instance) {
		if instance.ResourceVersion == cachedInstance.ResourceVersion {
			return r.UpdateObj(instance)
		}
		return reconcile.Result{Requeue: true}, nil
	}

	return reconcile.Result{}, nil
}

// newBCForCR returns a BuildConfig with the same name/namespace as the cr
func newBCsForCR(cr *v1alpha1.VirtualDatabase) map[string]obuildv1.BuildConfig {
	buildConfigs := map[string]obuildv1.BuildConfig{}
	serviceBC := obuildv1.BuildConfig{}
	images := constants.RuntimeImageDefaults[cr.Spec.Runtime]

	for _, imageDefaults := range images {
		if imageDefaults.BuilderImage {
			builderName := strings.Join([]string{cr.ObjectMeta.Name, "builder"}, "-")
			builderBC := obuildv1.BuildConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      builderName,
					Namespace: cr.Namespace,
					Labels: map[string]string{
						"app": cr.Name,
					},
				},
			}
			builderBC.SetGroupVersionKind(obuildv1.SchemeGroupVersion.WithKind("BuildConfig"))
			builderBC.Spec.Source.Git = &obuildv1.GitBuildSource{
				URI: cr.Spec.Build.GitSource.URI,
				Ref: cr.Spec.Build.GitSource.Reference,
			}
			builderBC.Spec.Source.ContextDir = cr.Spec.Build.GitSource.ContextDir
			builderBC.Spec.Output.To = &corev1.ObjectReference{Name: strings.Join([]string{builderName, "latest"}, ":"), Kind: "ImageStreamTag"}
			builderBC.Spec.Strategy.Type = obuildv1.SourceBuildStrategyType
			builderBC.Spec.Strategy.SourceStrategy = &obuildv1.SourceBuildStrategy{
				Incremental: cr.Spec.Build.Incremental,
				Env:         cr.Spec.Build.Env,
				From: corev1.ObjectReference{
					Name:      fmt.Sprintf("%s:%s", imageDefaults.ImageStreamName, imageDefaults.ImageStreamTag),
					Namespace: imageDefaults.ImageStreamNamespace,
					Kind:      "ImageStreamTag",
				},
			}

			buildConfigs["builder"] = builderBC
		} else {
			serviceBC = obuildv1.BuildConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cr.ObjectMeta.Name,
					Namespace: cr.Namespace,
					Labels: map[string]string{
						"app": cr.Name,
					},
				},
			}
			serviceBC.SetGroupVersionKind(obuildv1.SchemeGroupVersion.WithKind("BuildConfig"))
			serviceBC.Spec.Output.To = &corev1.ObjectReference{Name: strings.Join([]string{cr.ObjectMeta.Name, "latest"}, ":"), Kind: "ImageStreamTag"}
			serviceBC.Spec.Strategy.Type = obuildv1.SourceBuildStrategyType
		}
	}

	serviceBC.Spec.Strategy.SourceStrategy = &obuildv1.SourceBuildStrategy{
		From:      *buildConfigs["builder"].Spec.Output.To,
		ForcePull: false,
	}
	if len(cr.Spec.Build.SourceFileChanges) > 0 {
		serviceBC.Spec.Source.Type = obuildv1.BuildSourceBinary
		//serviceBC.Spec.Source.Binary = &obuildv1.BinaryBuildSource{}
	} else {
		serviceBC.Spec.Source.Type = obuildv1.BuildSourceImage
		serviceBC.Spec.Triggers = []obuildv1.BuildTriggerPolicy{
			{
				Type:        obuildv1.ImageChangeBuildTriggerType,
				ImageChange: &obuildv1.ImageChangeTrigger{From: buildConfigs["builder"].Spec.Output.To},
			},
		}
	}
	buildConfigs["service"] = serviceBC

	return buildConfigs
}

// newDCForCR returns a BuildConfig with the same name/namespace as the cr
func (r *ReconcileVirtualDatabase) newDCForCR(cr *v1alpha1.VirtualDatabase, serviceBC obuildv1.BuildConfig) (oappsv1.DeploymentConfig, error) {
	var probe *corev1.Probe
	replicas := int32(1)
	if cr.Spec.Replicas != nil {
		replicas = *cr.Spec.Replicas
	}
	labels := map[string]string{
		"app":              cr.Name,
		"syndesis.io/type": "datavirtualization",
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
		if envvar.Get(cr.Spec.Env, v.Name) == nil {
			envvar.SetVar(&cr.Spec.Env, v)
		}
	}

	ports := []corev1.ContainerPort{}
	ports = append(ports, corev1.ContainerPort{Name: "http", ContainerPort: int32(8080), Protocol: corev1.ProtocolTCP})
	ports = append(ports, corev1.ContainerPort{Name: "jolokia", ContainerPort: int32(8778), Protocol: corev1.ProtocolTCP})
	ports = append(ports, corev1.ContainerPort{Name: "promenthus", ContainerPort: int32(9779), Protocol: corev1.ProtocolTCP})
	ports = append(ports, corev1.ContainerPort{Name: "teiid", ContainerPort: int32(31000), Protocol: corev1.ProtocolTCP})
	ports = append(ports, corev1.ContainerPort{Name: "pg", ContainerPort: int32(35432), Protocol: corev1.ProtocolTCP})

	// resources for the container
	if &cr.Spec.Resources == nil {
		cr.Spec.Resources = corev1.ResourceRequirements{
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

	depConfig := oappsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.ObjectMeta.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: oappsv1.DeploymentConfigSpec{
			Replicas: replicas,
			Selector: labels,
			Strategy: oappsv1.DeploymentStrategy{
				Type: oappsv1.DeploymentStrategyTypeRolling,
			},
			Template: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					Name:   cr.ObjectMeta.Name,
					Annotations: map[string]string{
						"prometheus.io/scrape": "true",
						"prometheus.io/port":   "9779",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            cr.ObjectMeta.Name,
							Env:             cr.Spec.Env,
							Resources:       cr.Spec.Resources,
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
						ContainerNames: []string{cr.ObjectMeta.Name},
						From:           *serviceBC.Spec.Output.To,
					},
				},
			},
		},
	}

	depConfig.SetGroupVersionKind(oappsv1.SchemeGroupVersion.WithKind("DeploymentConfig"))
	var err = controllerutil.SetControllerReference(cr, &depConfig, r.scheme)
	if err != nil {
		log.Error(err)
		return oappsv1.DeploymentConfig{}, err
	}

	return depConfig, nil
}

// updateBuildConfigs ...
func (r *ReconcileVirtualDatabase) updateBuildConfigs(instance *v1alpha1.VirtualDatabase, bc *obuildv1.BuildConfig) (bool, error) {
	log := log.With("kind", instance.Kind, "name", instance.Name, "namespace", instance.Namespace)
	listOps := &client.ListOptions{Namespace: instance.Namespace}
	bcList := &obuildv1.BuildConfigList{}
	err := r.client.List(context.TODO(), listOps, bcList)
	if err != nil {
		log.Warn("Failed to list bc's. ", err)
		r.setFailedStatus(instance, v1alpha1.UnknownReason, err)
		return false, err
	}

	var bcUpdates []obuildv1.BuildConfig
	for _, lbc := range bcList.Items {
		if bc.Name == lbc.Name {
			bcUpdates = r.bcUpdateCheck(*bc, lbc, bcUpdates, instance)
		}
	}
	if len(bcUpdates) > 0 {
		for _, uBc := range bcUpdates {
			fmt.Println(uBc)
			_, err := r.UpdateObj(&uBc)
			if err != nil {
				r.setFailedStatus(instance, v1alpha1.DeploymentFailedReason, err)
				return false, err
			}
		}
		return true, nil
	}
	return false, nil
}

// UpdateObj reconciles the given object
func (r *ReconcileVirtualDatabase) UpdateObj(obj v1alpha1.OpenShiftObject) (reconcile.Result, error) {
	log := log.With("kind", obj.GetObjectKind().GroupVersionKind().Kind, "name", obj.GetName(), "namespace", obj.GetNamespace())
	log.Info("Updating")
	err := r.client.Update(context.TODO(), obj)
	if err != nil {
		log.Warn("Failed to update object. ", err)
		return reconcile.Result{}, err
	}
	// Object updated - return and requeue
	return reconcile.Result{Requeue: true}, nil
}

func (r *ReconcileVirtualDatabase) setFailedStatus(instance *v1alpha1.VirtualDatabase, reason v1alpha1.ReasonType, err error) {
	status.SetFailed(instance, reason, err)
	_, updateError := r.UpdateObj(instance)
	if updateError != nil {
		log.Warn("Unable to update object after receiving failed status. ", err)
	}
}

func (r *ReconcileVirtualDatabase) bcUpdateCheck(current, new obuildv1.BuildConfig, bcUpdates []obuildv1.BuildConfig, cr *v1alpha1.VirtualDatabase) []obuildv1.BuildConfig {
	log := log.With("kind", current.GetObjectKind().GroupVersionKind().Kind, "name", current.Name, "namespace", current.Namespace)
	update := false

	if !reflect.DeepEqual(current.Spec.Source, new.Spec.Source) {
		log.Debug("Changes detected in 'Source' config.", " OLD - ", current.Spec.Source, " NEW - ", new.Spec.Source)
		update = true
	}
	if !shared.EnvVarCheck(current.Spec.Strategy.SourceStrategy.Env, new.Spec.Strategy.SourceStrategy.Env) {
		log.Debug("Changes detected in 'Env' config.", " OLD - ", current.Spec.Strategy.SourceStrategy.Env, " NEW - ", new.Spec.Strategy.SourceStrategy.Env)
		update = true
	}
	if !reflect.DeepEqual(current.Spec.Resources, new.Spec.Resources) {
		log.Debug("Changes detected in 'Resource' config.", " OLD - ", current.Spec.Resources, " NEW - ", new.Spec.Resources)
		update = true
	}

	if update {
		bcnew := new
		err := controllerutil.SetControllerReference(cr, &bcnew, r.scheme)
		if err != nil {
			log.Error("Error setting controller reference for bc. ", err)
		}
		bcnew.SetNamespace(current.Namespace)
		bcnew.SetResourceVersion(current.ResourceVersion)
		bcnew.SetGroupVersionKind(obuildv1.SchemeGroupVersion.WithKind("BuildConfig"))

		bcUpdates = append(bcUpdates, bcnew)
	}
	return bcUpdates
}

func (r *ReconcileVirtualDatabase) hasSpecChanges(instance, cached *v1alpha1.VirtualDatabase) bool {
	if !reflect.DeepEqual(instance.Spec, cached.Spec) {
		return true
	}
	return false
}

func (r *ReconcileVirtualDatabase) hasStatusChanges(instance, cached *v1alpha1.VirtualDatabase) bool {
	if !reflect.DeepEqual(instance.Status, cached.Status) {
		return true
	}
	return false
}

// checkImageStream checks for ImageStream
func (r *ReconcileVirtualDatabase) checkImageStream(name, namespace string) bool {
	log := log.With("kind", "ImageStream", "name", name, "namespace", namespace)
	result := strings.Split(name, ":")
	_, err := r.imageClient.ImageStreams(namespace).Get(result[0], metav1.GetOptions{})
	if err != nil {
		log.Debug("Object does not exist")
		return false
	}
	return true
}

// ensureImageStream ...
func (r *ReconcileVirtualDatabase) ensureImageStream(name string, cr *v1alpha1.VirtualDatabase, setOwner bool) (string, error) {
	if r.checkImageStream(name, cr.Namespace) {
		return cr.Namespace, nil
	}
	err := r.createLocalImageStream(name, cr, setOwner)
	if err != nil {
		return cr.Namespace, err
	}
	return cr.Namespace, nil
}

// createLocalImageStream creates local ImageStream
func (r *ReconcileVirtualDatabase) createLocalImageStream(tagRefName string, cr *v1alpha1.VirtualDatabase, setOwner bool) error {
	result := strings.Split(tagRefName, ":")
	if len(result) == 1 {
		result = append(result, "latest")
	}

	isnew := &oimagev1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name:      result[0],
			Namespace: cr.Namespace,
		},
		Spec: oimagev1.ImageStreamSpec{
			LookupPolicy: oimagev1.ImageLookupPolicy{
				Local: true,
			},
		},
	}
	isnew.SetGroupVersionKind(oimagev1.SchemeGroupVersion.WithKind("ImageStream"))
	if setOwner {
		err := controllerutil.SetControllerReference(cr, isnew, r.scheme)
		if err != nil {
			log.Error("Error setting controller reference for ImageStream. ", err)
			return err
		}
	}

	log := log.With("kind", isnew.GetObjectKind().GroupVersionKind().Kind, "name", isnew.Name, "namespace", isnew.Namespace)
	log.Info("Creating")

	_, err := r.imageClient.ImageStreams(isnew.Namespace).Create(isnew)
	if err != nil && !errors.IsAlreadyExists(err) {
		log.Info("Already exists.")
	}
	return nil
}

// triggerBuild triggers a BuildConfig to start a new build
func (r *ReconcileVirtualDatabase) triggerBuild(bc obuildv1.BuildConfig, cr *v1alpha1.VirtualDatabase) error {
	log := log.With("kind", "BuildConfig", "name", bc.GetName(), "namespace", bc.GetNamespace())
	buildConfig, err := r.buildClient.BuildConfigs(bc.Namespace).Get(bc.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if buildConfig.Spec.Source.Type == obuildv1.BuildSourceBinary {
		files := map[string]string{}
		// Create list of files to archive
		for _, file := range cr.Spec.Build.SourceFileChanges {
			files[file.RelativePath] = file.Contents
		}
		tarReader, err := shared.Tar(files)
		if err != nil {
			return err
		}
		isName := buildConfig.Spec.Strategy.SourceStrategy.From.Name
		_, err = r.imageClient.ImageStreamTags(buildConfig.Namespace).Get(isName, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			log.Warn(isName, " ImageStreamTag does not exist yet and is required for this build.")
		} else if err != nil {
			return err
		} else {
			binaryBuildRequest := obuildv1.BinaryBuildRequestOptions{ObjectMeta: metav1.ObjectMeta{Name: buildConfig.Name}}
			binaryBuildRequest.SetGroupVersionKind(obuildv1.SchemeGroupVersion.WithKind("BinaryBuildRequestOptions"))
			log.Info("Triggering binary build ", buildConfig.Name)
			err = r.buildClient.RESTClient().Post().
				Namespace(cr.Namespace).
				Resource("buildconfigs").
				Name(buildConfig.Name).
				SubResource("instantiatebinary").
				Body(tarReader).
				VersionedParams(&binaryBuildRequest, scheme.ParameterCodec).
				Do().
				Into(&obuildv1.Build{})
			if err != nil {
				return err
			}
		}
	} else {
		buildRequest := obuildv1.BuildRequest{ObjectMeta: metav1.ObjectMeta{Name: buildConfig.Name}}
		buildRequest.SetGroupVersionKind(obuildv1.SchemeGroupVersion.WithKind("BuildRequest"))
		buildRequest.TriggeredBy = []obuildv1.BuildTriggerCause{{Message: fmt.Sprintf("Triggered by %s operator", cr.Kind)}}
		log.Info("Triggering build ", buildConfig.Name)
		_, err := r.buildClient.BuildConfigs(buildConfig.Namespace).Instantiate(buildConfig.Name, &buildRequest)
		if err != nil {
			return err
		}
	}

	return nil
}

// createObj creates an object based on the error passed in from a `client.Get`
func (r *ReconcileVirtualDatabase) createObj(obj v1alpha1.OpenShiftObject, err error) (reconcile.Result, error) {
	log := log.With("kind", obj.GetObjectKind().GroupVersionKind().Kind, "name", obj.GetName(), "namespace", obj.GetNamespace())

	if err != nil && errors.IsNotFound(err) {
		// Define a new Object
		log.Info("Creating")
		err = r.client.Create(context.TODO(), obj)
		if err != nil {
			log.Warn("Failed to create object. ", err)
			return reconcile.Result{}, err
		}
		// Object created successfully - return and requeue
		return reconcile.Result{RequeueAfter: time.Duration(200) * time.Millisecond}, nil
	} else if err != nil {
		log.Error("Failed to get object. ", err)
		return reconcile.Result{}, err
	}
	log.Debug("Skip reconcile - object already exists")
	return reconcile.Result{}, nil
}

func (r *ReconcileVirtualDatabase) updateDeploymentConfigs(instance *v1alpha1.VirtualDatabase, depConfig oappsv1.DeploymentConfig) (bool, error) {
	log := log.With("kind", instance.Kind, "name", instance.Name, "namespace", instance.Namespace)
	listOps := &client.ListOptions{Namespace: instance.Namespace}
	dcList := &oappsv1.DeploymentConfigList{}
	err := r.client.List(context.TODO(), listOps, dcList)
	if err != nil {
		log.Warn("Failed to list dc's. ", err)
		r.setFailedStatus(instance, v1alpha1.UnknownReason, err)
		return false, err
	}
	instance.Status.Deployments = getDeploymentsStatuses(dcList.Items, instance)

	var dcUpdates []oappsv1.DeploymentConfig
	for _, dc := range dcList.Items {
		if dc.Name == depConfig.Name {
			dcUpdates = r.dcUpdateCheck(dc, depConfig, dcUpdates, instance)
		}
	}
	log.Debugf("There are %d updated DCs", len(dcUpdates))
	if len(dcUpdates) > 0 {
		for _, uDc := range dcUpdates {
			_, err := r.UpdateObj(&uDc)
			if err != nil {
				r.setFailedStatus(instance, v1alpha1.DeploymentFailedReason, err)
				return false, err
			}
		}
		return true, nil
	}
	return false, nil
}

func sliceExists(slice interface{}, item interface{}) bool {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		panic("SliceExists() given a non-slice type")
	}
	for i := 0; i < s.Len(); i++ {
		if s.Index(i).Interface() == item {
			return true
		}
	}

	return false
}

func (r *ReconcileVirtualDatabase) dcUpdateCheck(current, new oappsv1.DeploymentConfig, dcUpdates []oappsv1.DeploymentConfig, cr *v1alpha1.VirtualDatabase) []oappsv1.DeploymentConfig {
	log := log.With("kind", new.GetObjectKind().GroupVersionKind().Kind, "name", current.Name, "namespace", current.Namespace)
	update := false
	if !reflect.DeepEqual(current.Spec.Template.Labels, new.Spec.Template.Labels) {
		log.Debug("Changes detected in labels.", " OLD - ", current.Spec.Template.Labels, " NEW - ", new.Spec.Template.Labels)
		update = true
	}
	if current.Spec.Replicas != new.Spec.Replicas {
		log.Debug("Changes detected in replicas.", " OLD - ", current.Spec.Replicas, " NEW - ", new.Spec.Replicas)
		update = true
	}

	cContainer := current.Spec.Template.Spec.Containers[0]
	nContainer := new.Spec.Template.Spec.Containers[0]
	if !shared.EnvVarCheck(cContainer.Env, nContainer.Env) {
		log.Debug("Changes detected in 'Env' config.", " OLD - ", cContainer.Env, " NEW - ", nContainer.Env)
		update = true
	}
	if !reflect.DeepEqual(cContainer.Resources, nContainer.Resources) {
		log.Debug("Changes detected in 'Resource' config.", " OLD - ", cContainer.Resources, " NEW - ", nContainer.Resources)
		update = true
	}
	sort.Slice(cContainer.Ports, func(i, j int) bool {
		return cContainer.Ports[i].Name < cContainer.Ports[j].Name
	})
	sort.Slice(nContainer.Ports, func(i, j int) bool {
		return nContainer.Ports[i].Name < nContainer.Ports[j].Name
	})
	if !reflect.DeepEqual(cContainer.Ports, nContainer.Ports) {
		log.Debug("Changes detected in 'Ports' config.", " OLD - ", cContainer.Ports, " NEW - ", nContainer.Ports)
		update = true
	}
	if update {
		dcnew := new
		err := controllerutil.SetControllerReference(cr, &dcnew, r.scheme)
		if err != nil {
			log.Error("Error setting controller reference for dc. ", err)
		}
		dcnew.SetNamespace(current.Namespace)
		dcnew.SetResourceVersion(current.ResourceVersion)
		dcnew.SetGroupVersionKind(oappsv1.SchemeGroupVersion.WithKind("DeploymentConfig"))

		dcUpdates = append(dcUpdates, dcnew)
	}
	return dcUpdates
}

func getDeploymentsStatuses(dcs []oappsv1.DeploymentConfig, cr *v1alpha1.VirtualDatabase) v1alpha1.Deployments {
	var ready, starting, stopped []string
	for _, dc := range dcs {
		for _, ownerRef := range dc.GetOwnerReferences() {
			if ownerRef.UID == cr.UID {
				if dc.Spec.Replicas == 0 {
					stopped = append(stopped, dc.Name)
				} else if dc.Status.Replicas == 0 {
					stopped = append(stopped, dc.Name)
				} else if dc.Status.ReadyReplicas < dc.Status.Replicas {
					starting = append(starting, dc.Name)
				} else {
					ready = append(ready, dc.Name)
				}
			}
		}
	}
	log.Debugf("Found DCs with status stopped [%s], starting [%s], and ready [%s]", stopped, starting, ready)
	return v1alpha1.Deployments{
		Stopped:  stopped,
		Starting: starting,
		Ready:    ready,
	}
}

// GetRouteHost returns the Hostname of the route provided
func (r *ReconcileVirtualDatabase) GetRouteHost(route oroutev1.Route, cr *v1alpha1.VirtualDatabase) string {
	route.SetGroupVersionKind(oroutev1.SchemeGroupVersion.WithKind("Route"))
	log := log.With("kind", route.GetObjectKind().GroupVersionKind().Kind, "name", route.Name, "namespace", route.Namespace)
	err := controllerutil.SetControllerReference(cr, &route, r.scheme)
	if err != nil {
		log.Error("Error setting controller reference. ", err)
	}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: route.Name, Namespace: route.Namespace}, &oroutev1.Route{})
	if err != nil && errors.IsNotFound(err) {
		route.ResourceVersion = ""
		_, err = r.createObj(
			&route,
			err,
		)
		if err != nil {
			log.Error("Error creating Route. ", err)
		}
	}

	found := &oroutev1.Route{}
	for i := 1; i < 60; i++ {
		time.Sleep(time.Duration(100) * time.Millisecond)
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: route.Name, Namespace: route.Namespace}, found)
		if err == nil {
			break
		}
	}
	if err != nil {
		log.Error("Error getting Route. ", err)
	}

	return found.Spec.Host
}

func addDefaultsToCR(cr *v1alpha1.VirtualDatabase) {
	// Set some CR defaults
	if cr.Spec.Runtime == "" || cr.Spec.Runtime != v1alpha1.SpringbootRuntimeType {
		cr.Spec.Runtime = v1alpha1.SpringbootRuntimeType
	}
	if cr.Spec.Build.Incremental == nil {
		inc := true
		cr.Spec.Build.Incremental = &inc
	}
}

// Reconcile reads that state of the cluster for a VirtualDatabase object and makes changes based on the state read
// and what is in the VirtualDatabase.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileVirtualDatabase) Reconcile2(request reconcile.Request) (reconcile.Result, error) {
	log.Info("Reconciling VirtualDatabase")

	buildStatus := &v1alpha1.BuildStatus{}
	tempDir, err := ioutil.TempDir(os.TempDir(), "builder-")
	if err != nil {
		log.Error(err, "Unexpected error while creating a temporary dir")
		return reconcile.Result{}, err
	}
	buildStatus.TarFile = tempDir + "vdb.tar"
	defer os.RemoveAll(tempDir)

	ctx := context.WithValue(context.TODO(), v1alpha1.BuildStatusKey, buildStatus)

	// Fetch the VirtualDatabase instance
	instance := &v1alpha1.VirtualDatabase{}
	err = r.client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	addDefaultsToCR(instance)

	// // Define a new Pod object
	// pod := newPodForCR(instance)

	// Set VirtualDatabase instance as the owner and controller
	// if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
	// 	return reconcile.Result{}, err
	// }

	buildSteps := []Action{
		NewInitializeAction(),
		NewCodeGenerationAction(),
	}

	// Delete phase
	if instance.GetDeletionTimestamp() != nil {
		instance.Status.Phase = v1alpha1.PublishingPhaseDeleting
	}

	for _, a := range buildSteps {
		a.InjectClient(r.client)
		a.InjectLogger(log)
		if a.CanHandle(instance) {
			log.Infof("Invoking action %s", a.Name())
			if err := a.Handle(ctx, instance); err != nil {
				if k8serrors.IsConflict(err) {
					log.Error(err, "conflict")
					return reconcile.Result{
						Requeue: true,
					}, nil
				}
				return reconcile.Result{}, err
			}
		}
	}

	// Fetch the VirtualDatabase again and check the state
	if err = r.client.Get(ctx, request.NamespacedName, instance); err != nil {
		if k8serrors.IsNotFound(err) && instance.Status.Phase == v1alpha1.PublishingPhaseDeleting {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, err
	}

	if instance.Status.Phase == v1alpha1.PublishingPhaseRunning {
		return reconcile.Result{}, nil
	}

	// Requeue
	return reconcile.Result{
		RequeueAfter: 5 * time.Second,
	}, nil
}
