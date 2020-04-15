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
	"crypto/sha256"
	"encoding/base64"
	"math/rand"
	"strconv"

	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/controller/virtualdatabase/constants"
	"github.com/teiid/teiid-operator/pkg/util/kubernetes"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

//ComputeConfigDigest --875040
func ComputeConfigDigest(ctx context.Context, client k8sclient.Reader, vdb *v1alpha1.VirtualDatabase) (string, error) {
	// check to see if any of the secrets or configmaps changed
	hashVal := sha256.New()
	for _, env := range vdb.Spec.Env {
		str, err := kubernetes.RevisionOfConfigMapOrSecret(ctx, client, vdb.ObjectMeta.Namespace, env)
		if err != nil {
			return "", err
		}
		hashVal.Write([]byte(str))
	}
	for _, source := range vdb.Spec.DataSources {
		hashVal.Write([]byte(source.Name))
		for _, env := range source.Properties {
			str, err := kubernetes.RevisionOfConfigMapOrSecret(ctx, client, vdb.ObjectMeta.Namespace, env)
			if err != nil {
				return "", err
			}
			hashVal.Write([]byte(str))
		}
	}
	configdigest := "c" + base64.RawURLEncoding.EncodeToString(hashVal.Sum(nil))
	return configdigest, nil
}

// ComputeForVirtualDatabase a digest of the fields that are relevant for the build
// Produces a digest that can be used as docker image tag
func ComputeForVirtualDatabase(vdb *v1alpha1.VirtualDatabase) (string, error) {
	hash := sha256.New()
	// Operator version is relevant
	if _, err := hash.Write([]byte(constants.Version)); err != nil {
		return "", err
	}

	// VDB DDL code
	if vdb.Spec.Build.Source.DDL != "" {
		if _, err := hash.Write([]byte(vdb.Spec.Build.Source.DDL)); err != nil {
			return "", err
		}
	}

	// if this Maven based deploy
	if vdb.Spec.Build.Source.Maven != "" {
		if _, err := hash.Write([]byte(vdb.Spec.Build.Source.Maven)); err != nil {
			return "", err
		}
	}

	// if it has OpenAPI
	if vdb.Spec.Build.Source.OpenAPI != "" {
		if _, err := hash.Write([]byte(vdb.Spec.Build.Source.OpenAPI)); err != nil {
			return "", err
		}
	}

	// Dependencies resources
	for _, item := range vdb.Spec.Build.Source.Dependencies {
		if _, err := hash.Write([]byte(item)); err != nil {
			return "", err
		}
	}

	// Add a letter at the beginning and use URL safe encoding
	digest := "v" + base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
	return digest, nil
}

// Random --
func Random() string {
	return "v" + strconv.FormatInt(rand.Int63(), 10)
}
