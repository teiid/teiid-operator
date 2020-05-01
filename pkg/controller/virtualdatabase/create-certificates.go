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

	"github.com/teiid/teiid-operator/pkg/controller/virtualdatabase/constants"
	"github.com/teiid/teiid-operator/pkg/util/kubernetes"
	"github.com/teiid/teiid-operator/pkg/util/pkcs12"

	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
)

// NewCreateCertificateAction creates a new initialize action
func NewCreateCertificateAction() Action {
	return &createCertificateAction{}
}

type createCertificateAction struct {
	baseAction
}

// Name returns a common name of the action
func (action *createCertificateAction) Name() string {
	return "createCertificateAction"
}

// CanHandle tells whether this action can handle the virtualdatabase
func (action *createCertificateAction) CanHandle(vdb *v1alpha1.VirtualDatabase) bool {
	return vdb.Status.Phase == v1alpha1.ReconcilerPhaseServiceCreated
}

// Handle handles the virtualdatabase
func (action *createCertificateAction) Handle(ctx context.Context, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) error {
	// check to see if the secret is already there, if yes, then nothing needs to be done.
	_, err := kubernetes.GetSecret(ctx, r.client, getKeystoreSecretName(vdb), vdb.ObjectMeta.Namespace)
	if err == nil {
		vdb.Status.Phase = v1alpha1.ReconcilerPhaseKeystoreCreated
		return nil
	}

	// if key store is not found then look for the either provided ir generated certificates and create a keystore with it
	certs, err := kubernetes.GetSecret(ctx, r.client, getCertificateSecretName(vdb), vdb.ObjectMeta.Namespace)
	if err != nil {
		log.Error("Failed to read certificate/key for encryption")
		return err
	}

	// build the keystore from the pem cert and key
	keystorePkcs12, err := pkcs12.CreatePkcs12Keystore(certs.Data["tls.crt"], certs.Data["tls.key"], constants.KeystorePassword)
	if err != nil {
		log.Error("Failed to create the Keystore")
		return err
	}

	// read the default Kubernestes service cert and then create a trust store
	defaultTrustCert, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt")
	if err != nil {
		log.Error("Failed to read /var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt")
		return err
	}
	truststorePkcs12, err := pkcs12.CreatePkcs12Truststore(constants.KeystorePassword, defaultTrustCert)
	if err != nil {
		log.Error("Failed to create the Truststore")
		return err
	}

	// build the secret with keystore and truststore
	data := map[string][]byte{
		constants.KeystoreName:   keystorePkcs12,
		constants.TruststoreName: truststorePkcs12,
	}
	err = kubernetes.CreateSecret(r.client, getKeystoreSecretName(vdb), vdb.ObjectMeta.Namespace, vdb, data)
	if err != nil {
		log.Error("Failed to create the Keystore Secret", err)
		return err
	}

	vdb.Status.Phase = v1alpha1.ReconcilerPhaseKeystoreCreated
	return nil
}

func getKeystoreSecretName(vdb *v1alpha1.VirtualDatabase) string {
	return vdb.ObjectMeta.Name + "-" + "keystore"
}

func getCertificateSecretName(vdb *v1alpha1.VirtualDatabase) string {
	return vdb.ObjectMeta.Name + "-" + "certificates"
}
