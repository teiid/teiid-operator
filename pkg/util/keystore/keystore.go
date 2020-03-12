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
package keystore

import (
	"bytes"
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"math/rand"
	"strconv"
	"time"

	"github.com/pavel-v-chernykh/keystore-go"
	"github.com/teiid/teiid-operator/pkg/util/logs"
)

var log = logs.GetLogger("keystore-util")

//https://github.com/pavel-v-chernykh/keystore-go/issues/9

// GenerateKeyStoreFromPem returns a Java Keystore with a self-signed certificate
func GenerateKeyStoreFromPem(alias string, password string, pemCert []byte, pemKey []byte) ([]byte, error) {
	var chain []keystore.Certificate

	// decode the certificates
	for {
		blockCert, rest := pem.Decode(pemCert)
		if blockCert == nil {
			return nil, errors.New("The supplied Pem certificate could not be decoded")
		}

		chain = append(chain, keystore.Certificate{
			Type:    "X509",
			Content: blockCert.Bytes,
		})

		// there typically more than one certificate
		if len(rest) > 0 {
			pemCert = rest
		} else {
			break
		}
	}

	// now go after the key, single one
	blockKey, _ := pem.Decode(pemKey)
	if blockKey == nil {
		return nil, errors.New("The supplied Pem certificate key could not be decoded")
	}

	// write the keystore now
	keyStore := keystore.KeyStore{
		alias: &keystore.PrivateKeyEntry{
			Entry: keystore.Entry{
				CreationDate: time.Now(),
			},
			PrivKey:   blockKey.Bytes,
			CertChain: chain,
		},
	}

	var b bytes.Buffer
	err := keystore.Encode(&b, keyStore, []byte(password))
	if err != nil {
		log.Error("Error encrypting and signing keystore. ", err)
		return nil, err
	}
	return b.Bytes(), nil
}

// GenerateTrustStoreFromPem returns a Java Keystore with a self-signed certificate
func GenerateTrustStoreFromPem(alias string, password string, certs ...[]byte) ([]byte, error) {
	keyStore := keystore.KeyStore{}

	var i int
	for _, pemCert := range certs {
		for {
			blockCert, rest := pem.Decode(pemCert)
			if blockCert == nil  {
				return nil, errors.New("The supplied Pem certificate for truststore could not be decoded")
			}

			tse := keystore.TrustedCertificateEntry{
				Entry: keystore.Entry{
					CreationDate: time.Now(),
				},
				Certificate: keystore.Certificate{
					Type:    "X509",
					Content: blockCert.Bytes,
				},
			}
			str := alias + "-" + strconv.Itoa(i)
			i++
			keyStore[str] = &tse
			// there typically more than one certificate
			if len(rest) > 0 {
				pemCert = rest
			} else {
				break
			}
		}
	}

	// write it as store
	var b bytes.Buffer
	err := keystore.Encode(&b, keyStore, []byte(password))
	if err != nil {
		log.Error("Error encrypting and signing truststore. ", err)
		return nil, err
	}

	return b.Bytes(), nil
}

// GenerateKeystore returns a Java Keystore with a self-signed certificate
func GenerateKeystore(commonName, alias string, password []byte) []byte {
	cert, derPK, err := genCert(commonName)
	if err != nil {
		log.Error("Error generating certificate. ", err)
	}

	var chain []keystore.Certificate
	keyStore := keystore.KeyStore{
		alias: &keystore.PrivateKeyEntry{
			Entry: keystore.Entry{
				CreationDate: time.Now(),
			},
			PrivKey: derPK,
			CertChain: append(chain, keystore.Certificate{
				Type:    "X509",
				Content: cert,
			}),
		},
	}

	var b bytes.Buffer
	err = keystore.Encode(&b, keyStore, password)
	if err != nil {
		log.Error("Error encrypting and signing keystore. ", err)
	}

	return b.Bytes()
}

// ????????????????
// any way to use openshift's CA for signing instead ??
func genCert(commonName string) (cert []byte, derPK []byte, err error) {
	sAndI := pkix.Name{
		CommonName: commonName,
		//OrganizationalUnit: []string{"Engineering"},
		//Organization:       []string{"RedHat"},
		//Locality:           []string{"Raleigh"},
		//Province:           []string{"NC"},
		//Country:            []string{"US"},
	}

	serialNumber, err := crand.Int(crand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		log.Error("Error getting serial number. ", err)
		return nil, nil, err
	}

	ca := &x509.Certificate{
		Subject:            sAndI,
		Issuer:             sAndI,
		SignatureAlgorithm: x509.SHA256WithRSA,
		PublicKeyAlgorithm: x509.ECDSA,
		NotBefore:          time.Now(),
		NotAfter:           time.Now().AddDate(10, 0, 0),
		SerialNumber:       serialNumber,
		SubjectKeyId:       sha256.New().Sum(nil),
		IsCA:               true,
		// BasicConstraintsValid: true,
		// ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		// KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}

	priv, err := rsa.GenerateKey(crand.Reader, 2048)
	if err != nil {
		log.Error("create key failed. ", err)
		return nil, nil, err
	}

	cert, err = x509.CreateCertificate(crand.Reader, ca, ca, &priv.PublicKey, priv)
	if err != nil {
		log.Error("create cert failed. ", err)
		return nil, nil, err
	}

	derPK, err = x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		log.Error("Marshal to PKCS8 key failed. ", err)
		return nil, nil, err
	}

	return cert, derPK, nil
}

// GeneratePassword returns an alphanumeric password of the length provided
func GeneratePassword(length int) []byte {
	rand.Seed(time.Now().UnixNano())
	digits := "0123456789"
	all := "ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		digits
	buf := make([]byte, length)
	buf[0] = digits[rand.Intn(len(digits))]
	for i := 1; i < length; i++ {
		buf[i] = all[rand.Intn(len(all))]
	}

	rand.Shuffle(len(buf), func(i, j int) {
		buf[i], buf[j] = buf[j], buf[i]
	})

	return buf
}
