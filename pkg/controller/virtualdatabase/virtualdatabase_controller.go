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
	"time"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/client/versioned/typed/monitoring/v1"
	buildv1client "github.com/openshift/client-go/build/clientset/versioned/typed/build/v1"
	imagev1 "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
	teiidclient "github.com/teiid/teiid-operator/pkg/client"
	"github.com/teiid/teiid-operator/pkg/util/logs"
	"github.com/teiid/teiid-operator/pkg/util/openshift"
	otclient "github.com/teiid/teiid-operator/pkg/util/opentracing/client"
	"k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var log = logs.GetLogger("virtualdatabase")
var _ reconcile.Reconciler = &ReconcileVirtualDatabase{}

//var eventSubscribers = &events.EventSubscribers{}

// ReconcileVirtualDatabase reconciles a VirtualDatabase object
type ReconcileVirtualDatabase struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client           teiidclient.Client
	imageClient      *imagev1.ImageV1Client
	buildClient      *buildv1client.BuildV1Client
	prometheusClient monitoringv1.MonitoringV1Interface
	jaegerClient     *otclient.JaegertracingV1Client
}

// Reconcile reads that state of the cluster for a VirtualDatabase object and makes changes based on the state read
// and what is in the VirtualDatabase.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileVirtualDatabase) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := context.TODO()

	// event listeners
	//eventSubscribers.Register(CacheStoreListener{})

	// Fetch the VirtualDatabase instance
	instance := &v1alpha1.VirtualDatabase{}
	err := r.client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// eventSubscribers.Trigger(events.VdbDeleted, request.NamespacedName, r)
			if err := openshift.ConsoleLinkExists(); err == nil {
				instance.ObjectMeta = metav1.ObjectMeta{
					Name:      request.Name,
					Namespace: request.Namespace,
				}
				openshift.RemoveConsoleLink(ctx, r.client, instance)
			}
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	log.Debugf("Reconciling VirtualDatabase: %s", instance.ObjectMeta.Name)

	buildSteps := []Action{
		NewInitializeAction(),
		NewCacheStoreAction(),
		News2IBuilderImageAction(),
		NewServiceImageAction(),
		NewCreateServiceAction(),
		NewCreateCertificateAction(),
		NewDeploymentAction(),
		NewPrometheusMonitorAction(),
	}

	// make deep copy and do not directly update the stock copy as other might
	// have access to this
	target := instance.DeepCopy()

	// check if the VDB has been updated, then redo everything
	if IsVdbUpdated(target) {
		RedeployVdb(target)
		if err := r.client.Update(ctx, target); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	// run through the different actions now
	for _, a := range buildSteps {
		if a.CanHandle(target) {
			phaseFrom := target.Status.Phase

			log.Debugf("Invoking action %s", a.Name())
			var processError error
			if processError = a.Handle(ctx, target, r); processError != nil {
				log.Error("Failed during action ", a.Name(), " ", processError)
				//return reconcile.Result{}, err
			}

			// only if the object changed update it
			if r.hasChanges(instance, target) {
				// update runtime object
				if err := r.client.Update(ctx, target); err != nil {
					if k8serrors.IsConflict(err) {
						log.Error(err, "conflict")
						//log.Debug(err, " conflict ", instance, " target ", target)
					}
					return reconcile.Result{}, err
				}

				targetPhase := target.Status.Phase

				if targetPhase != phaseFrom {
					log.Info(
						"state transition",
						" phase-from:", phaseFrom,
						" phase-to:", targetPhase,
					)
				}
				// this will auto-queue since the update is successful
				return reconcile.Result{}, processError
			}
		} else {
			continue
		}
	}

	// Requeue
	return reconcile.Result{
		RequeueAfter: 5 * time.Second,
	}, nil
}

func (r *ReconcileVirtualDatabase) hasChanges(instance, cached *v1alpha1.VirtualDatabase) bool {
	if !reflect.DeepEqual(instance.Spec, cached.Spec) {
		return true
	}
	if !reflect.DeepEqual(instance.Status, cached.Status) {
		return true
	}
	return false
}
