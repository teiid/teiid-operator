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

func TestDSWithComments(t *testing.T) {

	ddl := `CREATE DATABASE customer OPTIONS (ANNOTATION 'Customer VDB');
	USE DATABASE customer;

	CREATE /*comment -- */ SERVER sampledb 
		FOREIGN DATA WRAPPER postgresql 
		OPTIONS( foo 'bar');

	CREATE /*comm
		ent*/SERVER /*comment */ "mongo" FOREIGN /*sfsf*/DATA WRAPPER mongodb;

	CREATE SCHEMA accounts SERVER sampledb;
	CREATE VIRTUAL SCHEMA portfolio;`

	sources := ParseDataSourcesInfoFromDdl(ddl)

	assert.Equal(t, "sampledb", sources[0].Name)
	assert.Equal(t, "postgresql", sources[0].Type)

	assert.Equal(t, "mongo", sources[1].Name)
	assert.Equal(t, "mongodb", sources[1].Type)

}

func TestMaterialized(t *testing.T) {
	ddl := `CREATE DATABASE customer OPTIONS (ANNOTATION 'Customer VDB');
	USE DATABASE customer;

	CREATE VIRTUAL SCHEMA portfolio;
	CREATE VIEW vix (
		"date" date primary key,
		"close" double,
		MA10 double
	) OPTIONS (
		MaterIALIZED 'TRUE',
		"teiid_rel:ALLOW_MATVIEW_MANAGEMENT" 'true',
		"teiid_rel:MATVIEW_LOADNUMBER_COLUMN" 'LoadNumber',
		"teiid_rel:MATVIEW_STATUS_TABLE" 'vix_mat.status'
	) AS 
		select t.*, AVG("close") OVER (ORDER BY "date" ASC ROWS 9 PRECEDING) AS MA10 from 
		  (call vix_source.invokeHttp(action=>'GET', endpoint=>'https://datahub.io/core/finance-vix/r/vix-daily.csv')) w, 
		texttable(to_chars(w.result, 'ascii') COLUMNS "date" date, "open" HEADER 'Vix Open' double, "high" HEADER 'Vix High' double, "low" HEADER 'Vix Low' double, "close" HEADER 'Vix Close' double HEADER) t;	  
	`
	assert.Equal(t, true, ShouldMaterialize(ddl))
}

func TestMaterializedWithVirtual(t *testing.T) {
	ddl := `CREATE DATABASE customer OPTIONS (ANNOTATION 'Customer VDB');
	USE DATABASE customer;

	CREATE VIRTUAL SCHEMA portfolio;
	create virtual view vix (
		"date" date primary key,
		"close" double,
		MA10 double
	) OPTIONS (
		MaterIALIZED 'TRUE',
		"teiid_rel:ALLOW_MATVIEW_MANAGEMENT" 'true',
		"teiid_rel:MATVIEW_LOADNUMBER_COLUMN" 'LoadNumber',
		"teiid_rel:MATVIEW_STATUS_TABLE" 'vix_mat.status'
	) AS 
		select t.*, AVG("close") OVER (ORDER BY "date" ASC ROWS 9 PRECEDING) AS MA10 from 
		  (call vix_source.invokeHttp(action=>'GET', endpoint=>'https://datahub.io/core/finance-vix/r/vix-daily.csv')) w, 
		texttable(to_chars(w.result, 'ascii') COLUMNS "date" date, "open" HEADER 'Vix Open' double, "high" HEADER 'Vix High' double, "low" HEADER 'Vix Low' double, "close" HEADER 'Vix Close' double HEADER) t;	  
	`
	assert.Equal(t, true, ShouldMaterialize(ddl))
}

func TestMaterializedwithComments(t *testing.T) {

	ddl := `CREATE DATABASE customer OPTIONS (ANNOTATION 'Customer VDB');
	USE DATABASE customer;

	CREATE VIRTUAL SCHEMA portfolio;

	CREATE VIEW /* this is 
		a comment*/ vix2 (
		"date" date primary key,
	) OPTIONS (
		MATERIALIZED TRUE,
		"teiid_rel:ALLOW_MATVIEW_MANAGEMENT" 'true'
	) AS 
		select t.*, AVG("close") OVER (ORDER BY "date" ASC ROWS 9 PRECEDING) AS MA10 from foo;		  
	`
	assert.Equal(t, true, ShouldMaterialize(ddl))
}

func TestMaterializedwithNestedSchema(t *testing.T) {
	ddl := `CREATE DATABASE customer OPTIONS (ANNOTATION 'Customer VDB');
	USE DATABASE customer;

	CREATE VIRTUAL SCHEMA portfolio
	CREATE VIEW /* this is 
		a comment*/ vix2 (
		"date" date primary key,
	) OPTIONS (
		"teiid_rel:ALLOW_MATVIEW_MANAGEMENT" 'true'
	) AS 
		select t.*, AVG("close") OVER (ORDER BY "date" ASC ROWS 9 PRECEDING) AS MA10 from foo
	CREATE VIEW vix3 (
		"date" date primary key,
	) OPTIONS (
		MATERIALIZED TRUE,
		"teiid_rel:ALLOW_MATVIEW_MANAGEMENT" 'true'
	) AS 
		select t.*, AVG("close") OVER (ORDER BY "date" ASC ROWS 9 PRECEDING) AS MA10 from foo;		  

	`
	assert.Equal(t, true, ShouldMaterialize(ddl))
}

func TestNotMaterialized(t *testing.T) {

	ddl := `CREATE DATABASE customer OPTIONS (ANNOTATION 'Customer VDB');
	USE DATABASE customer;

	CREATE VIRTUAL SCHEMA portfolio;
	CREATE VIEW vix (
		"date" date primary key,
		MA10 double
	) OPTIONS (
		MATERIALIZED 'TRUE',
		materialized_table 'vix_mat.vixcache',
		"teiid_rel:ALLOW_MATVIEW_MANAGEMENT" 'true',
		"teiid_rel:MATVIEW_LOADNUMBER_COLUMN" 'LoadNumber',
		"teiid_rel:MATVIEW_STATUS_TABLE" 'vix_mat.status'
	) AS 
		select t.*, AVG("close") OVER (ORDER BY "date" ASC ROWS 9 PRECEDING) AS MA10 from 
		  (call vix_source.invokeHttp(action=>'GET', endpoint=>'https://datahub.io/core/finance-vix/r/vix-daily.csv')) w, 
		  texttable(to_chars(w.result, 'ascii') COLUMNS "date" date, "open" HEADER 'Vix Open' double, "high" HEADER 'Vix High' double, "low" HEADER 'Vix Low' double, "close" HEADER 'Vix Close' double HEADER) t;	
	`
	assert.Equal(t, false, ShouldMaterialize(ddl))
}

func TestValidateDataSourceNames(t *testing.T) {
	ds := []DatasourceInfo{
		{
			Name: "foo",
			Type: "postgresql",
		},
	}
	assert.Nil(t, ValidateDataSourceNames(ds))

	ds = []DatasourceInfo{
		{
			Name: "foo-bar",
			Type: "postgresql",
		},
	}
	assert.NotNil(t, ValidateDataSourceNames(ds))

	ds = []DatasourceInfo{
		{
			Name: "foo.bar",
			Type: "postgresql",
		},
	}
	assert.NotNil(t, ValidateDataSourceNames(ds))

	ds = []DatasourceInfo{
		{
			Name: "fooêbar",
			Type: "postgresql",
		},
	}
	assert.NotNil(t, ValidateDataSourceNames(ds))
}
