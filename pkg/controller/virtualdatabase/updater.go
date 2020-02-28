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
	"strconv"

	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
)

// IsVdbUpdated --
func IsVdbUpdated(vdb *v1alpha1.VirtualDatabase) bool {
	digest, err := ComputeForVirtualDatabase(vdb)
	if err == nil {
		return digest != vdb.Status.Digest
	}
	return false
}

// RedeployVdb Handle handles the virtualdatabase
func RedeployVdb(vdb *v1alpha1.VirtualDatabase) error {
	digest, _ := ComputeForVirtualDatabase(vdb)
	vdb.Status.Phase = v1alpha1.ReconcilerPhaseInitial
	vdb.Status.Digest = digest

	// we only want to update the version implicitly when the DDL based model is used
	// for maven based it is expected of the user to change the version of maven to be reflected here
	if vdb.Spec.Build.Source.DDL != "" && vdb.Spec.Build.Source.Version == "" {
		ver, err := strconv.Atoi(vdb.Status.Version)
		if err == nil {
			vdb.Status.Version = strconv.Itoa(ver + 1)
		}
		log.Debugf("new version is %s", vdb.Status.Version)
	}
	log.Infof("Changes detected in VDB %s:%s, redeploying", vdb.ObjectMeta.Name, vdb.Status.Version)
	return nil
}
