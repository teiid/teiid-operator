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
	"strings"
	"time"

	oimagev1 "github.com/openshift/api/image/v1"
	buildv1client "github.com/openshift/client-go/build/clientset/versioned/typed/build/v1"
	imagev1 "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	"github.com/teiid/teiid-operator/pkg/apis/vdb/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/controller/virtualdatabase/logs"
	"k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileVirtualDatabase) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Debug("Reconciling VirtualDatabase")

	ctx := context.TODO()

	// Fetch the VirtualDatabase instance
	instance := &v1alpha1.VirtualDatabase{}
	err := r.client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	buildSteps := []Action{
		NewInitializeAction(),
		News2IBuilderImageAction(),
		NewServiceImageAction(),
		NewDeploymentAction(),
	}

	// make deep copy and do not directly update the stock copy as other might
	// have access to this
	target := instance.DeepCopy()

	for _, a := range buildSteps {
		if a.CanHandle(target) {
			phaseFrom := target.Status.Phase

			log.Debugf("Invoking action %s", a.Name())
			if err := a.Handle(ctx, target, r); err != nil {
				log.Error("Failed during action ", a.Name(), err)
				return reconcile.Result{}, err
			}

			// only if the object changed update it
			if r.hasChanges(instance, target) {
				// update runtime object
				if err := r.client.Update(ctx, target); err != nil {
					if k8serrors.IsConflict(err) {
						log.Error(err, "conflict")
						return reconcile.Result{
							Requeue:      true,
							RequeueAfter: 5 * time.Second,
						}, nil
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
			}
		} else {
			continue
		}
		break
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
