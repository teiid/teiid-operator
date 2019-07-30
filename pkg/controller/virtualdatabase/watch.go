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
	oappsv1 "github.com/openshift/api/apps/v1"
	obuildv1 "github.com/openshift/api/build/v1"
	oimagev1 "github.com/openshift/api/image/v1"
	routev1 "github.com/openshift/api/route/v1"
	buildv1client "github.com/openshift/client-go/build/clientset/versioned/typed/build/v1"
	imagev1 "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	"github.com/teiid/teiid-operator/pkg/apis/vdb/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new VirtualDatabase Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	imageClient, err := imagev1.NewForConfig(mgr.GetConfig())
	if err != nil {
		log.Errorf("Error getting image client: %v", err)
		return &ReconcileVirtualDatabase{}
	}
	buildClient, err := buildv1client.NewForConfig(mgr.GetConfig())
	if err != nil {
		log.Errorf("Error getting build client: %v", err)
		return &ReconcileVirtualDatabase{}
	}
	return &ReconcileVirtualDatabase{
		client:      mgr.GetClient(),
		scheme:      mgr.GetScheme(),
		cache:       mgr.GetCache(),
		imageClient: imageClient,
		buildClient: buildClient,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("virtualdatabase-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource VirtualDatabase
	watchObjects := []runtime.Object{
		&v1alpha1.VirtualDatabase{},
		&obuildv1.BuildConfig{},
		&obuildv1.Build{},
		&oimagev1.ImageStream{},
		&appsv1.Deployment{},
	}
	objectHandler := &handler.EnqueueRequestForObject{}
	for _, watchObject := range watchObjects {
		err = c.Watch(&source.Kind{Type: watchObject}, objectHandler)
		if err != nil {
			return err
		}
	}

	// Watch for changes to secondary resource Pods and requeue the owner VirtualDatabase
	err = c.Watch(&source.Kind{Type: &v1alpha1.VirtualDatabase{}}, &handler.EnqueueRequestForObject{}, predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldVirtualDatabase := e.ObjectOld.(*v1alpha1.VirtualDatabase)
			newVirtualDatabase := e.ObjectNew.(*v1alpha1.VirtualDatabase)
			// Ignore updates to the integration status in which case metadata.Generation does not change,
			// or except when the integration phase changes as it's used to transition from one phase
			// to another
			return oldVirtualDatabase.Generation != newVirtualDatabase.Generation ||
				oldVirtualDatabase.Status.Phase != newVirtualDatabase.Status.Phase
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Evaluates to false if the object has been confirmed deleted
			return !e.DeleteStateUnknown
		},
	})
	if err != nil {
		return err
	}

	watchOwnedObjects := []runtime.Object{
		&oappsv1.DeploymentConfig{},
		&corev1.PersistentVolumeClaim{},
		&corev1.Service{},
		&routev1.Route{},
		&obuildv1.BuildConfig{},
		&obuildv1.Build{},
		&oimagev1.ImageStream{},
	}
	ownerHandler := &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &v1alpha1.VirtualDatabase{},
	}
	for _, watchObject := range watchOwnedObjects {
		err = c.Watch(&source.Kind{Type: watchObject}, ownerHandler)
		if err != nil {
			return err
		}
	}

	return nil
}
