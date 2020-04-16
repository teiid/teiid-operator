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
package vdbutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenizer(t *testing.T) {
	var ddl = "foo;bar;"

	lines := Tokenize(ddl)
	assert.Equal(t, "foo;", lines[0])
	assert.Equal(t, "bar;", lines[1])
}

func TestDS(t *testing.T) {
	var ddl = `CREATE DATABASE customer OPTIONS (ANNOTATION 'Customer VDB');
	USE DATABASE customer;

	CREATE SERVER sampledb TYPE "NONE" FOREIGN DATA WRAPPER postgresql;

	CREATE SERVER mongo TYPE 'NONE' FOREIGN DATA WRAPPER mongodb;

	CREATE SCHEMA accounts SERVER sampledb;
	CREATE VIRTUAL SCHEMA portfolio;`

	sources := ParseDataSourcesInfoFromDdl(ddl)

	assert.Equal(t, "sampledb", sources[0].Name)
	assert.Equal(t, "postgresql", sources[0].Type)

	assert.Equal(t, "mongo", sources[1].Name)
	assert.Equal(t, "mongodb", sources[1].Type)

}

func TestDS2(t *testing.T) {

	ddl := `CREATE DATABASE customer OPTIONS (ANNOTATION 'Customer VDB');
	USE DATABASE customer;

	CREATE SERVER sampledb 
		FOREIGN DATA WRAPPER postgresql 
		OPTIONS( foo 'bar');

	CREATE SERVER "mongo" FOREIGN DATA WRAPPER mongodb;

	CREATE SCHEMA accounts SERVER sampledb;
	CREATE VIRTUAL SCHEMA portfolio;`

	sources := ParseDataSourcesInfoFromDdl(ddl)

	assert.Equal(t, "sampledb", sources[0].Name)
	assert.Equal(t, "postgresql", sources[0].Type)

	assert.Equal(t, "mongo", sources[1].Name)
	assert.Equal(t, "mongodb", sources[1].Type)

}
