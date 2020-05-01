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
	"crypto/x509"
	"encoding/base64"
	"io/ioutil"
	"testing"

	gopkcs12 "github.com/hetesiistvan/go-pkcs12"
	"github.com/stretchr/testify/assert"
)

func TestCreateStorePkcs12(t *testing.T) {
	password := "changeit"

	cert := []byte("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURtVENDQW9HZ0F3SUJBZ0lJSGtWN2VQUU9OZnd3RFFZSktvWklodmNOQVFFTEJRQXdOakUwTURJR0ExVUUKQXd3cmIzQmxibk5vYVdaMExYTmxjblpwWTJVdGMyVnlkbWx1WnkxemFXZHVaWEpBTVRVNE1ETTFOekk1TlRBZQpGdzB5TURBek1USXdNVEl5TWpCYUZ3MHlNakF6TVRJd01USXlNakZhTUNReElqQWdCZ05WQkFNVEdXUjJMV04xCmMzUnZiV1Z5TG0xNWNISnZhbVZqZEM1emRtTXdnZ0VpTUEwR0NTcUdTSWIzRFFFQkFRVUFBNElCRHdBd2dnRUsKQW9JQkFRQ2oxRm03K3RDSm1Pa3NKMFhWVFNhZ3pBYVRXK3M0N2xHWUtRZXpiNUM2R0I5ZVdmaHY2VGxaV29NawpMQWJLSUR3SCsxa2pmUHlqSjR5YXZtRTU0MjhlTmtUdldSUXZVcDVmMDhNSE9tbEtVb3dVelhmbnNjTVNyTWJwCjJ2RTZlQ2tqd1dKcWlsYUtMMExmdGhaS2JvRTRWbVZ5VlFpQWtydW5odjhYNkcxbjBxOGdPSDM5VFpRckpIZXkKOWhuTTJMb3BFSUpGaGdOUVVzQ1FuaUtzRXdMWmxVWFRkWS9LZm42eWpmUnRHWloyN0ZnN0gzMXNCNEFTajNkTgowemJlZU1XNXcwNU5LOFB4ZU9JeDEvVkQ4dGVzWFphQlhJREkrRWNMTzBwOTF4WkFjMVBEbUpxRHpFcXN4N2VKCldCSDVqVnFYVENXRC9yVmVId1JFWUkzQkYvR2JBZ01CQUFHamdid3dnYmt3RGdZRFZSMFBBUUgvQkFRREFnV2cKTUJNR0ExVWRKUVFNTUFvR0NDc0dBUVVGQndNQk1Bd0dBMVVkRXdFQi93UUNNQUF3VFFZRFZSMFJCRVl3UklJWgpaSFl0WTNWemRHOXRaWEl1Ylhsd2NtOXFaV04wTG5OMlk0SW5aSFl0WTNWemRHOXRaWEl1Ylhsd2NtOXFaV04wCkxuTjJZeTVqYkhWemRHVnlMbXh2WTJGc01EVUdDeXNHQVFRQmtnZ1JaQUlCQkNZVEpHVmlPV1JtTVdNMUxUWXoKWm1ZdE1URmxZUzFpWlRBNUxUVXlabVJtWXpBM01qRTRNakFOQmdrcWhraUc5dzBCQVFzRkFBT0NBUUVBb0dCcQp5S09vNzZoYWQ2RWZJa0FsalpNN2JZdGtDOFBzM0pYR0RXMnJxSmN5S3RnUkR6Wnp0RndOTXlodytqZnBZU014CmVmYmtvK25DVzQxRkhpaTlTMGVXc3publdHZi9WbkgxWTRXVTgrRVV5NnhZQkFpVUoyZ3UzV1M4ZWZuUkJsb3kKWXgwSUJNNkZPMW5WbWJya2FkZHR0K096WWdWSmhqYnpzMWlaRzIvNWZIbEJVa01WdFNRVE4rd3dWWWN6RHpnMwpmTVI1RkJDOS9Ed0x1eVp6aStlWlJJck9URWtUdVpqaFNQZEFyaWNpeldLeUxaS3E1cXBmRWNWRjk0WjBxOUpECnZxWTNqSzExcVd6VmdIY3RROGpVVFhxbk1hMlA0TS8zM1FBSERDekVESUpKR1MwNzdDQVpicHhGd2V5WEl4VUoKM0lleGFMM0lFTXhoV0tRTlVnPT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQotLS0tLUJFR0lOIENFUlRJRklDQVRFLS0tLS0KTUlJRENqQ0NBZktnQXdJQkFnSUJBVEFOQmdrcWhraUc5dzBCQVFzRkFEQTJNVFF3TWdZRFZRUUREQ3R2Y0dWdQpjMmhwWm5RdGMyVnlkbWxqWlMxelpYSjJhVzVuTFhOcFoyNWxja0F4TlRnd016VTNNamsxTUI0WERUSXdNREV6Ck1EQTBNRGd4TkZvWERUSXhNREV5T1RBME1EZ3hOVm93TmpFME1ESUdBMVVFQXd3cmIzQmxibk5vYVdaMExYTmwKY25acFkyVXRjMlZ5ZG1sdVp5MXphV2R1WlhKQU1UVTRNRE0xTnpJNU5UQ0NBU0l3RFFZSktvWklodmNOQVFFQgpCUUFEZ2dFUEFEQ0NBUW9DZ2dFQkFMWW9GdnJPdWxmTENHUU1OZ0VRSEg5OE5vajM2K3pQcm5sM2ZpbkRCVU03CnpZTm1oT2xCSW1lQW5EUkdpaEJOajRJa29qMmFpaDZ1cGQwVFp4SUVBaGMwSVN3dXU0d1A5NEVxNXNwczFkemIKbVBFT3FRd2xzMURjdFI0QUNqdSt0ZXZGSDFhQlk1VU5WUjNkWWRxeFNaVDdFYjk3Zkk1S05JbzJCbDlkQ2hXdgp5ckErUkozRWFnNUs2ejFWUWIvUVdmWURlNGtBd2hsenZyZGxYcTBxQlZhRFNSQTZKYitRNEkrc2c0S3ZqUm5nCk9CSG1pN1NzdWNacVFBbDY1M21SWWVmYjhyRTFtMG4rMUhKZUxscU1ROVI3SXRtMXFvRFRod1VZWThrVkJ6dlkKM2VRNW42WGk1SnRjdGJpcU5vSE0zaFBUcGJ6blQ4czk4V0JaMG9Id2JxRUNBd0VBQWFNak1DRXdEZ1lEVlIwUApBUUgvQkFRREFnS2tNQThHQTFVZEV3RUIvd1FGTUFNQkFmOHdEUVlKS29aSWh2Y05BUUVMQlFBRGdnRUJBSU1OCmF0amJqdDNMMVk3cU0rZkYySDlNMEdtYmhBcFVqY3psYTV0Yys2WXNMeGdLS1d5eEJCdzIxcGJGdld6NDRHWFYKWTdrMHRKTWV6MUZUbi9RZ1Arcm9lM3hWd2dJdmh4dEpaalhxZS9xdzErTlM4RE9WQlVodWJvYjJGMHFTQnhpUgpPek9kMFljZjY5YkdxMzVvSTJWWkR3RzYvSWEwYW9RamRqSEpsVUF4R2JOeXk3c3htenBGM3pCU3NOZW1ySkhTCnRBVlZhMG1wRUhwWjg0UFJFWWROZ1BPdlJSR25lck05WldWZWIvWUFSaUk0K3pubE5uRjlkdytaSXNoeGs0V1YKS0l6R1Vxd3BDQm1yN2tJTVIxTWh4cmlIekdpZVJqWDZydTNiNEZ3YnZWQnJZOEtwbGJjZi9aVEYrSGlkbTY5Lwo1cnVJN2k4bXlNdnJoRWFxd0lZPQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==")
	key := []byte("LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFcEFJQkFBS0NBUUVBbzlSWnUvclFpWmpwTENkRjFVMG1vTXdHazF2ck9PNVJtQ2tIczIrUXVoZ2ZYbG40CmIrazVXVnFESkN3R3lpQThCL3RaSTN6OG95ZU1tcjVoT2VOdkhqWkU3MWtVTDFLZVg5UERCenBwU2xLTUZNMTMKNTdIREVxekc2ZHJ4T25ncEk4Rmlhb3BXaWk5QzM3WVdTbTZCT0ZabGNsVUlnSks3cDRiL0YraHRaOUt2SURoOQovVTJVS3lSM3N2WVp6Tmk2S1JDQ1JZWURVRkxBa0o0aXJCTUMyWlZGMDNXUHluNStzbzMwYlJtV2R1eFlPeDk5CmJBZUFFbzkzVGRNMjNuakZ1Y05PVFN2RDhYamlNZGYxUS9MWHJGMldnVnlBeVBoSEN6dEtmZGNXUUhOVHc1aWEKZzh4S3JNZTNpVmdSK1kxYWwwd2xnLzYxWGg4RVJHQ053UmZ4bXdJREFRQUJBb0lCQVFDRXhQdFVGSmdjYXdmTQorS2Juam5iWHFZRkt1eHVPTDlXQWN3QUNzMCtmQVIycTRVOHRvdDBQUlFNeXRWdHJRMlJqTTVleDR3RDdXSG5pCmpwZE15cnlxeDJCbWVOS2E1MkhpVjBPZS8vK0VkQkdDYW1IYUszM2tESkhIdzkvcmVxWWNqQVN1UXg2UExtNEwKencyUmxLeTBjNUFUY0VaTHJKN1h6ZGUrRUdkWjAyQjN0VnpVL0pGN0N4WlU2L1N2a2N0T29icTJEYUtTdXlCaApHbm10Z0ZjUVJESmpmMVVwdkV6YlE2NzBvME9DaUIyVzRQRm54SUdZZ0orUU5vSDM3TkxJTi9WNERpdW1oOVMwCkRESTViOFYzTU5RR2ZtYkZVS2UvRFNyU3orYWRuUTNSUm1hQTRIT3BRVWRiNEVzaXBaRVhXNEZTWXZnalBMdFkKakU0blpDK0JBb0dCQU1jZHYzOU1FaU41OEptTkl3N3ZMV1dYNjd2TTJrTDdLZ2hNTEw4d0hYWk9aSmZ2Sm05awpScFBrWnNBNEJmdERVSjZGY25URC9ldDJJRG9xMmcyM0hDdithYzdrTktnZlUySC94NHlzL2RWaDJSSTB6WHhBCnl5cThDMy9DVWJ5bmt0c1BaLzVSbWptVXcrWEQ0ajBTeHFuVEZMSFRHbnM0dmlxdUh6eCttK1FEQW9HQkFOS2gKN2NWMU9kM2JoUDNPTkhuVEhIT2JFVitFWkRnczZkb2FTSGFJY1dHOU9iQ0Q4SFZoWkREZTluUVBxYVNCVSswVAprNThUWEN0bEt0VkFleUxXWkRqci84ZVlKU3VIM1BRcVdRYUJIMXBuM3J0VDFiWWZHbGlVZm5JQmlHT3NYdHJQCkFlQmZxYkpYOWxJSklvSVpVeTlwbEJIZjFrR1l3OWNoMkJmbk02U0pBb0dBZXExdllNVERvQ3Z1K3d3ai8zMkoKSU1EYk1wZmlHY2FaZlFkQndvR29oVTJEV01DMWs0ZmFuQi9xMXA4dHdFTVhGclB0Y3RlV1NFNDlTTmxDQTVVLwp2RE5CaVlDOG1LREVST3JNVFhYLzVrb2s3Ynl1cGRGZDIzU0VPVERHSDArM2dWUWFwR3d1Y3krZkNwOEhjczF3CnJRMHFBTzJwc1NXaXRMVVc5YlNqNDNVQ2dZRUFuZlpjZC9JakZKUDFsOVlXR3FyTk1wRy9wSytIN1cwWmI3eTQKVFZTa0cxV3F0d3Nyd1F2cDlKQ3hxWGE1bGFwN3cxY2tKVytDZHZUbSs0amhEODVTMlRGNzREYms0VkdCemdjWQpQcjJGUXVxVTZrM0QvMUl5RXU1Q0tjT21nb0dabldVVGxpNkgrRHpwZUxwckM4QnNWeWxKcDJJRHI2d2VhdTl4CnZQTmlFbWtDZ1lCYnVzSTZLYVdGWGRUaVdwZ29HUExrZFlMMjEySktTdUxRdFNNQXZsUS9ZY1oyaDhTSE1xT1kKZ21yN3YwNUc5Y3VWNGNEMlpJT3I2cmFsZUl4VmF6d1p0a2QwZjEvb0xia0gvRm9ocnJJQ2t0bUthbll2UVlJLwpCK2dLVHhQVndiNG9QT3dCVVVMbE50WGdpL04zREZvTi9IMEI1U3Y2QTkwUGNMKzMzR2ZoVlE9PQotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQo=")

	// decode from base 64 first
	pemCert, err := base64.StdEncoding.DecodeString(string(cert))
	assert.Nil(t, err)

	pemKey, err := base64.StdEncoding.DecodeString(string(key))
	assert.Nil(t, err)

	pfxdata, err := CreatePkcs12Keystore(pemCert, pemKey, password)
	assert.Nil(t, err)

	//err = ioutil.WriteFile("keystore.pkcs12", keyBytes, 0644)
	//assert.Nil(t, err)

	c := &x509.Certificate{}
	k, c, err := gopkcs12.Decode(pfxdata, password)
	assert.Nil(t, err)
	assert.NotNil(t, k)
	assert.NotNil(t, c)

	assert.Nil(t, err)
	assert.NotNil(t, k)

	assert.Nil(t, err)
	assert.Equal(t, "dv-customer.myproject.svc", c.Subject.CommonName)
}

func TestCreatePkcs12Truststore(t *testing.T) {
	password := "changeit"
	defaultTrustCert, err := ioutil.ReadFile("service-ca.crt")
	assert.Nil(t, err)

	pfxData, err := CreatePkcs12Truststore(password, defaultTrustCert)
	assert.Nil(t, err)

	//ioutil.WriteFile("truststore.jks", keyBytes, 0644)

	keyStore, err := gopkcs12.DecodeTrustStore(pfxData, password)
	assert.Nil(t, err)

	for _, v := range keyStore {
		assert.NotNil(t, v)
		assert.NotNil(t, v.Subject.CommonName)
	}
}

func TestParse(t *testing.T) {
	defaultCert, err := ioutil.ReadFile("tls.cert")
	assert.Nil(t, err)

	pfxData, err := parseCrt(defaultCert)
	assert.Nil(t, err)

	assert.NotNil(t, pfxData)

	defaultCert, err = ioutil.ReadFile("tls.key")
	assert.Nil(t, err)

	pfxKey, err := parseKey(defaultCert)
	assert.Nil(t, err)

	assert.NotNil(t, pfxKey)
}
