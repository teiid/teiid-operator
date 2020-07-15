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
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/client"
	"github.com/teiid/teiid-operator/pkg/util"
	"github.com/teiid/teiid-operator/pkg/util/cachestore"
	"github.com/teiid/teiid-operator/pkg/util/envvar"
	"github.com/teiid/teiid-operator/pkg/util/events"
	"github.com/teiid/teiid-operator/pkg/util/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// NewCacheStoreAction creates a new cachestore action
func NewCacheStoreAction() Action {
	return &cacheStoreAction{}
}

type cacheStoreAction struct {
	baseAction
}

// Name returns a common name of the action
func (action *cacheStoreAction) Name() string {
	return "cachestore"
}

// CanHandle tells whether this action can handle the virtualdatabase
func (action *cacheStoreAction) CanHandle(vdb *v1alpha1.VirtualDatabase) bool {
	return vdb.Status.Phase == v1alpha1.ReconcilerPhaseCreateCacheStore
}

// IgnoreCacheStore Check to see if the cacheStore needs to be used or not
func IgnoreCacheStore(r *ReconcileVirtualDatabase, vdb *v1alpha1.VirtualDatabase) bool {
	disableIspn := true
	ispnAvailable, err := cachestore.IsInfinispanOperatorAvailable(r.client, vdb.ObjectMeta.Namespace)
	if err != nil {
		ispnAvailable = false
	}
	// is ispn is available by default allow, unless explicitly turned off
	if ispnAvailable {
		disableIspn = false
	}
	cachingFlag := envvar.Get(vdb.Spec.Env, "DISABLE_ISPN_CACHING")
	if cachingFlag != nil {
		disableIspn, err = strconv.ParseBool(cachingFlag.Value) // ignore error
	}
	return !ispnAvailable || disableIspn
}

// Handle handles the virtualdatabase
func (action *cacheStoreAction) Handle(ctx context.Context, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) error {

	// make sure the cache store is ignored.
	if IgnoreCacheStore(r, vdb) {
		vdb.Status.Phase = v1alpha1.ReconcilerPhaseS2IReady
		return nil
	}

	// check to see if the cache store exists if not create one to be used for Teiid Materialization and
	// result set caching purposes.
	var secrectFound bool = true
	config, _ := cachestore.Credentials(vdb.ObjectMeta.Name, vdb.ObjectMeta.Namespace, r.client)
	if config == nil {
		log.Debug("Configuration for the Cache Store Not found, using default settings")
		config = &cachestore.InfinispanDetails{}
		config.Name = vdb.ObjectMeta.Name + "-cache-store"
		config.NameSpace = vdb.ObjectMeta.Namespace
		config.CreateIfNotFound = true
		config.Replicas = 3
		config.User = "developer"
		config.Password = util.RandomPassword()
		secrectFound = false
	}

	if config.CreateIfNotFound {
		err := action.createNewCacheStore(config, r.client, vdb)
		if err != nil {
			log.Info("Failed to create Cache Store ", err)
		} else {
			vdb.Status.CacheStore = config.NameSpace + "/" + config.Name
			// create a secret if we are self-create mode
			if !secrectFound {
				action.createCacheStoreSecret(config, r.client, vdb)
			}
		}
	}

	vdb.Status.Phase = v1alpha1.ReconcilerPhaseS2IReady
	return nil
}

//createNewCacheStore -- create a new instance of Infinispan
func (action *cacheStoreAction) createNewCacheStore(ispn *cachestore.InfinispanDetails, client client.Client, owner metav1.Object) error {
	exists, err := cachestore.IsInfinispanOperatorAvailable(client, ispn.NameSpace)
	if err != nil {
		return err
	}

	cacheStoreName := ispn.Name
	if exists {
		identitySecret := strings.Join([]string{
			"credentials:",
			"- username: " + ispn.User,
			"  password: " + ispn.Password,
			"- username: operator",
			"  password: " + util.RandomPassword(),
		}, "\n")

		data := map[string][]byte{
			"identities.yaml": []byte(identitySecret),
		}

		log.Debugf("Creating a Identity Secret for Infinispan access %s", cacheStoreName+"-identity")
		err = kubernetes.CreateSecret(client, cacheStoreName+"-identity", ispn.NameSpace, owner, data)
		if err != nil {
			log.Debugf("failed, to create Identity Secret for Infinispan access %s", cacheStoreName+"-identity")
			return err
		}
		log.Debugf("Successfully created Identity Secret for Infinispan access %s", cacheStoreName+"-identity")

		log.Debugf("Starting to create Infinispan Cluster with name %s", cacheStoreName)
		ispnInstance := cachestore.NewInfinispanResource(ispn.NameSpace, cacheStoreName, cacheStoreName+"-identity", 3)
		err = controllerutil.SetControllerReference(owner, &ispnInstance, client.GetScheme())
		if err != nil {
			log.Error(err)
		}

		_, err := client.IspnClient().Infinispans(ispn.NameSpace).Create(&ispnInstance)
		if err != nil {
			log.Debugf("Failed to create Infinispan Cluster with name %s", cacheStoreName)
			return err
		}
		log.Debugf("Success, in creating Infinispan Cluster with name %s", cacheStoreName)
	} else {
		log.Info("Failed to create CacheStore as Infinispan Operator not found")
		return errors.New("Failed to create CacheStore as Infinispan Operator not found")
	}
	return nil
}

func (action *cacheStoreAction) createCacheStoreSecret(ispn *cachestore.InfinispanDetails, client client.Client, owner metav1.Object) error {
	create := "TRUE"
	if ispn.CreateIfNotFound {
		create = "FALSE"
	}
	replicas := strconv.Itoa(int(ispn.Replicas))
	data := map[string][]byte{
		"name":      []byte(ispn.Name),
		"namespace": []byte(ispn.NameSpace),
		"username":  []byte(ispn.User),
		"password":  []byte(ispn.Password),
		"url":       []byte(ispn.Name + ":11222"),
		"create":    []byte(create),
		"replicas":  []byte(replicas),
	}

	log.Debugf("Creating a Secret for Infinispan access %s", ispn.Name)
	err := kubernetes.CreateSecret(client, ispn.Name, ispn.NameSpace, owner, data)
	if err != nil {
		log.Debugf("failed, to create Secret for Infinispan access %s", ispn.Name)
		return err
	}
	log.Debugf("Successfully created Secret for Infinispan access %s", ispn.Name)
	return nil
}

// CacheStoreListener --
type CacheStoreListener struct {
	events.EventListener
}

func (s *CacheStoreListener) onEvent(eventType events.EventType, resource types.NamespacedName, payload interface{}) {
	// nothing do here yet until we have shared cache store, then we need to delete the expired caches
	log.Info("Received Event ", eventType)
}
