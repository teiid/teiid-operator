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
	"crypto/sha256"
	"encoding/base64"
	"math/rand"
	"strconv"

	"github.com/teiid/teiid-operator/pkg/apis/vdb/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/controller/virtualdatabase/constants"
)

// ComputeForVirtualDatabase a digest of the fields that are relevant for the deployment
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

	// Dependencies resources
	for _, item := range vdb.Spec.Build.Source.Dependencies {
		if _, err := hash.Write([]byte(item)); err != nil {
			return "", err
		}
	}

	// Git code, may be webhook is more appropriate here
	if vdb.Spec.Build.Git.URI != "" {
		if _, err := hash.Write([]byte(vdb.Spec.Build.Git.URI)); err != nil {
			return "", err
		}
		if vdb.Spec.Build.Git.Reference != "" {
			if _, err := hash.Write([]byte(vdb.Spec.Build.Git.Reference)); err != nil {
				return "", err
			}
		}
		if vdb.Spec.Build.Git.ContextDir != "" {
			if _, err := hash.Write([]byte(vdb.Spec.Build.Git.ContextDir)); err != nil {
				return "", err
			}
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
