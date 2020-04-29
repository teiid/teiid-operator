package virtualdatabase

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

import (
	"strings"

	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/util"
	"github.com/teiid/teiid-operator/pkg/util/cachestore"
	"github.com/teiid/teiid-operator/pkg/util/events"
	"github.com/teiid/teiid-operator/pkg/util/kubernetes"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

//createNewCacheStore -- create a new instance of Infinispan
func createNewCacheStore(vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) (bool, error) {
	exists, err := cachestore.IsInfinispanOperatorAvailable(r.client, vdb.ObjectMeta.Namespace)
	if err != nil {
		return false, err
	}
	cacheStoreName := vdb.ObjectMeta.Name + "-cache-store"
	if exists {
		password := util.RandomPassword()
		identitySecret := strings.Join([]string{
			"credentials:",
			"- username: developer",
			"  password: " + password,
			"- username: operator",
			"  password: " + util.RandomPassword(),
		}, "\n")

		data := map[string][]byte{
			"identities.yaml": []byte(identitySecret),
		}

		log.Debugf("Creating a Identity Secret for Infinispan access %s", cacheStoreName+"-identity")
		err = kubernetes.CreateSecret(r.client, cacheStoreName+"-identity", vdb.ObjectMeta.Namespace, vdb, data)
		if err != nil {
			log.Debugf("failed, to create Identity Secret for Infinispan access %s", cacheStoreName+"-identity")
			return false, err
		}
		log.Debugf("Successfully created Identity Secret for Infinispan access %s", cacheStoreName+"-identity")

		log.Debugf("Starting to create Infinispan Cluster with name %s", cacheStoreName)
		ispnInstance := cachestore.NewInfinispanResource(vdb.ObjectMeta.Namespace, cacheStoreName, cacheStoreName+"-identity", 3)
		err = controllerutil.SetControllerReference(vdb, &ispnInstance, r.client.GetScheme())
		if err != nil {
			log.Error(err)
		}

		_, err := r.client.IspnClient().Infinispans(vdb.ObjectMeta.Namespace).Create(&ispnInstance)
		if err != nil {
			log.Debugf("Failed to create Infinispan Cluster with name %s", cacheStoreName)
			return false, err
		}

		log.Debugf("Success, in creating Infinispan Cluster with name %s", cacheStoreName)

		data = map[string][]byte{
			"name":      []byte(cacheStoreName),
			"namespace": []byte(vdb.ObjectMeta.Namespace),
			"username":  []byte("developer"),
			"password":  []byte(password),
			"url":       []byte(cacheStoreName + ":11222"),
		}

		log.Debugf("Creating a Secret for Infinispan access %s", cacheStoreName)
		err = kubernetes.CreateSecret(r.client, cacheStoreName, vdb.ObjectMeta.Namespace, vdb, data)
		if err != nil {
			log.Debugf("failed, to create Secret for Infinispan access %s", cacheStoreName)
			return false, err
		}
		log.Debugf("Successfully created Secret for Infinispan access %s", cacheStoreName)
	} else {
		log.Info("Failed to create CacheStore as Infinispan Operator not found")
	}
	return true, nil
}

// CacheStoreListener --
type CacheStoreListener struct {
	events.EventListener
}

func (s *CacheStoreListener) onEvent(eventType events.EventType, resource types.NamespacedName, payload interface{}) {
	// nothing do here yet until we have shared cache store, then we need to delete the expired caches
	log.Info("Received Event ", eventType)
}
