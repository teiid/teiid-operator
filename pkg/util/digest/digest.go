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

package digest

import (
	"crypto/sha256"
	"encoding/base64"
	"math/rand"
	"strconv"

	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/util/defaults"
)

// ComputeForIntegration a digest of the fields that are relevant for the deployment
// Produces a digest that can be used as docker image tag
func ComputeForVirtualDatabase(vdb *v1alpha1.VirtualDatabase) (string, error) {
	hash := sha256.New()
	// Operator version is relevant
	if _, err := hash.Write([]byte(defaults.Version)); err != nil {
		return "", err
	}

	// Integration code
	if vdb.Spec.Content != "" {
		if _, err := hash.Write([]byte(vdb.Spec.Content)); err != nil {
			return "", err
		}
	}

	// Integration dependencies
	for _, item := range vdb.Spec.Dependencies {
		if _, err := hash.Write([]byte(item)); err != nil {
			return "", err
		}
	}

	// Integration configuration
	for _, item := range vdb.Spec.Configuration {
		if _, err := hash.Write([]byte(item.String())); err != nil {
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
