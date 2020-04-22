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
	"errors"
	"strconv"

	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/util/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CacheStoreExists -- check to so if the Infinispan CacheStore exists
func CacheStoreExists(ctx context.Context, vdbName string, vdbNamespace string, r *ReconcileVirtualDatabase) bool {
	ispnSecret, err := kubernetes.GetSecret(ctx, r.client, vdbName+"-cache-store", vdbNamespace)
	if err != nil {
		ispnSecret, err = kubernetes.GetSecret(ctx, r.client, "teiid-cache-store", vdbNamespace)
		if err != nil {
			return false
		}
	}
	name := string(ispnSecret.Data["name"])
	namespace := string(ispnSecret.Data["namespace"])
	return HasInfinispan(ctx, r, name, namespace)
}

// HasInfinispan --
func HasInfinispan(context context.Context, r *ReconcileVirtualDatabase, name string, namespace string) bool {
	_, err := r.ispnClient.Infinispans(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return false
	}
	return true
}

// CreateOrUseCacheStore - check to so if the Infinispan CacheStore exists
func CreateOrUseCacheStore(ctx context.Context, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) error {
	ispnSecret, err := kubernetes.GetSecret(ctx, r.client, vdb.ObjectMeta.Name+"-cache-store", vdb.ObjectMeta.Namespace)
	if err != nil {
		ispnSecret, err = kubernetes.GetSecret(ctx, r.client, "teiid-cache-store", vdb.ObjectMeta.Namespace)
		if err != nil {
			return err
		}
	}
	name := string(ispnSecret.Data["name"])
	namespace := string(ispnSecret.Data["namespace"])
	create, err := strconv.ParseBool(string(ispnSecret.Data["create"]))
	if err != nil {
		return err
	}
	if !HasInfinispan(ctx, r, name, namespace) && create {
		log.Info("Create Infinispan instance is not supported yet")
		return errors.New("Create Infinispan instance is not supported yet")
	}
	return nil
}
