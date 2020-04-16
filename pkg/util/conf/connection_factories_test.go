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

package conf

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestConnectionFactories --
func TestConnectionFactories(t *testing.T) {

	contents, _ := ioutil.ReadFile("../../../build/conf/connection_factories.json")
	factories := loadConnectionFactories(contents)

	sample := ConnectionFactory{
		Name:                     "db2",
		DriverNames:              []string{"com.ibm.db2.jcc.DB2Driver"},
		TranslatorName:           "db2",
		Dialect:                  "org.hibernate.dialect.DB2Dialect",
		Gav:                      []string{"com.ibm.db2.jcc:db2jcc4"},
		SpringBootPropertyPrefix: "spring.datasource",
	}
	assert.NotNil(t, factories)
	assert.Equal(t, sample, factories["db2"])
}
