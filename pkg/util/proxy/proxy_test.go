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

package proxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseProxyHost(t *testing.T) {
	actual := parseHTTPProxy("http://myhost:8080")

	assert.Equal(t, "myhost", actual["http.proxyHost"])
	assert.Equal(t, "8080", actual["http.proxyPort"])

	actual = parseHTTPProxy("https://myhost:8080")

	assert.Equal(t, "myhost", actual["https.proxyHost"])
	assert.Equal(t, "8080", actual["https.proxyPort"])

	actual = parseHTTPProxy("https://johndoe:ficklebird@myhost:8080")

	assert.Equal(t, "myhost", actual["https.proxyHost"])
	assert.Equal(t, "8080", actual["https.proxyPort"])
	assert.Equal(t, "johndoe", actual["https.proxyUser"])
	assert.Equal(t, "ficklebird", actual["https.proxyPassword"])

	actual = parseHTTPProxy("https://johndoe:fickle@bird@myhost:8080")

	assert.Equal(t, "myhost", actual["https.proxyHost"])
	assert.Equal(t, "8080", actual["https.proxyPort"])
	assert.Equal(t, "johndoe", actual["https.proxyUser"])
	assert.Equal(t, "fickle@bird", actual["https.proxyPassword"])
}

func TestParseNoProxy(t *testing.T) {
	actual := parseNoProxy("localhost,   foo.bat.com, linux.net")

	assert.Equal(t, "localhost|foo.bat.com|linux.net", actual)
}
