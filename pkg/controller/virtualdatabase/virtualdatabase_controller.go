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
	"io/ioutil"
	"os"
	"time"

	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
	teiidv1alpha1 "github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/client"
	teiidclient "github.com/teiid/teiid-operator/pkg/client"
	"github.com/teiid/teiid-operator/pkg/util/log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	c, err := client.FromManager(mgr)
	if err != nil {
		return err
	}
	return add(mgr, newReconciler(mgr, c))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, c client.Client) reconcile.Reconciler {
	return &ReconcileVirtualDatabase{client: c, scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("virtualdatabase-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource VirtualDatabase
	err = c.Watch(&source.Kind{Type: &teiidv1alpha1.VirtualDatabase{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Pods and requeue the owner VirtualDatabase
	err = c.Watch(&source.Kind{Type: &teiidv1alpha1.VirtualDatabase{}}, &handler.EnqueueRequestForObject{}, predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldVirtualDatabase := e.ObjectOld.(*teiidv1alpha1.VirtualDatabase)
			newVirtualDatabase := e.ObjectNew.(*teiidv1alpha1.VirtualDatabase)
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

	return nil
}

var _ reconcile.Reconciler = &ReconcileVirtualDatabase{}

// ReconcileVirtualDatabase reconciles a VirtualDatabase object
type ReconcileVirtualDatabase struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client teiidclient.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a VirtualDatabase object and makes changes based on the state read
// and what is in the VirtualDatabase.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileVirtualDatabase) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	logger := log.Log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	logger.Info("Reconciling VirtualDatabase")

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
	instance := &teiidv1alpha1.VirtualDatabase{}
	err = r.client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// // Define a new Pod object
	// pod := newPodForCR(instance)

	// // Set VirtualDatabase instance as the owner and controller
	// if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
	// 	return reconcile.Result{}, err
	// }

	buildSteps := []Action{
		NewInitializeAction(),
		NewCodeGenerationAction(),
	}

	// Delete phase
	if instance.GetDeletionTimestamp() != nil {
		instance.Status.Phase = teiidv1alpha1.PublishingPhaseDeleting
	}

	ilog := logger.ForVirtualDatabase(instance)
	for _, a := range buildSteps {
		a.InjectClient(r.client)
		a.InjectLogger(ilog)
		if a.CanHandle(instance) {
			ilog.Infof("Invoking action %s", a.Name())
			if err := a.Handle(ctx, instance); err != nil {
				if k8serrors.IsConflict(err) {
					ilog.Error(err, "conflict")
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
		if k8serrors.IsNotFound(err) && instance.Status.Phase == teiidv1alpha1.PublishingPhaseDeleting {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, err
	}

	if instance.Status.Phase == teiidv1alpha1.PublishingPhaseRunning {
		return reconcile.Result{}, nil
	}

	// Requeue
	return reconcile.Result{
		RequeueAfter: 5 * time.Second,
	}, nil
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newPodForCR(cr *teiidv1alpha1.VirtualDatabase) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-pod",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: []string{"sleep", "3600"},
				},
			},
		},
	}
}
