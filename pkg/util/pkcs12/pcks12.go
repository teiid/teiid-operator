package pkcs12

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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"strconv"

	gopkcs12 "github.com/hetesiistvan/go-pkcs12"
)

// CreatePkcs12Keystore creates a PKCS12 keystore from a certificate and the private key byte slices
func CreatePkcs12Keystore(cert []byte, key []byte, password string) ([]byte, error) {

	domainCert, _ := parseCrt(cert)
	privateKey, _ := parseKey(key)

	pfxData, err := gopkcs12.Encode(rand.Reader, privateKey, domainCert, nil, password)

	return pfxData, err
}

func parseCrt(cert []byte) (*x509.Certificate, error) {
	p := &pem.Block{}
	p, _ = pem.Decode(cert)
	return x509.ParseCertificate(p.Bytes)
}

func parseKey(key []byte) (*rsa.PrivateKey, error) {
	p, _ := pem.Decode(key)
	return x509.ParsePKCS1PrivateKey(p.Bytes)
}

// CreatePkcs12Truststore --
func CreatePkcs12Truststore(password string, certs ...[]byte) ([]byte, error) {
	certificates := make(map[string]*x509.Certificate)

	var i int
	for _, pemCert := range certs {
		for {
			blockCert, rest := pem.Decode(pemCert)
			if blockCert == nil {
				return nil, errors.New("The supplied Pem certificate for truststore could not be decoded")
			}

			trustedCert, err := x509.ParseCertificate(blockCert.Bytes)
			if err != nil {
				return nil, err
			}

			str := "cert-" + strconv.Itoa(i)
			i++
			certificates[str] = trustedCert

			// there typically more than one certificate
			if len(rest) > 0 {
				pemCert = rest
			} else {
				break
			}
		}
	}

	pfxData, err := gopkcs12.EncodeTrustStore(rand.Reader, certificates, password)
	return pfxData, err
}
