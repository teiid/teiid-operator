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
	"regexp"
	"strings"
)

// DatasourceInfo --
type DatasourceInfo struct {
	Name string `yaml:"name,omitempty"`
	Type string `yaml:"type,omitempty"`
}

// Tokenize --
func Tokenize(ddl string) []string {
	regEx := "(?is)\\w('[^']*'|\"[^\"]*\"|[^'\";])*;"
	var compRegEx *regexp.Regexp
	compRegEx = regexp.MustCompile(regEx)
	matches := compRegEx.FindAllString(ddl, -1)
	return matches
}

// ParseDataSourcesInfoFromDdl --
func ParseDataSourcesInfoFromDdl(ddl string) []DatasourceInfo {

	var sources []DatasourceInfo
	id := "(\\w+|(?:\"[^\"]*\")|()'[^']*'+)"
	regEx := "CREATE\\s+SERVER\\s+" + id + "\\s+(TYPE\\s+" + id + "\\s+)??FOREIGN\\s+DATA\\s+WRAPPER\\s+" + id

	var compRegEx *regexp.Regexp
	compRegEx = regexp.MustCompile(regEx)

	lines := Tokenize(ddl)
	for _, line := range lines {
		line = strings.ToUpper(line)

		if ok, _ := regexp.Match(regEx, []byte(line)); ok {
			match := compRegEx.FindStringSubmatch(line)
			sources = append(sources, DatasourceInfo{
				Name: stripQuotes(match[1]),
				Type: stripQuotes(match[6]),
			})
		}
	}
	return sources
}

// MaterializiedViewsInDdl --
func MaterializiedViewsInDdl(ddl string) []string {
	materialization := "MATERIALIZED\\s+'?TRUE'?"
	materializationTable := "MATERIALIZED_TABLE\\s+"
	id := "(\\w+|(?:\"[^\"]*\")|()'[^']*'+)"

	var views []string = make([]string, 0)
	viewEx := "CREATE\\s+VIEW\\s+" + id + "\\s+.*"
	viewRegEx := regexp.MustCompile(viewEx)

	databaseEx := "CREATE\\s+DATABASE\\s+" + id + ";?\\s+.*"
	databaseRegEx := regexp.MustCompile(databaseEx)

	createSchemaEx := "CREATE\\s+VIRTUAL\\s+SCHEMA\\s+" + id + "[;?$|\\s+.*]"
	createSchemaRegEx := regexp.MustCompile(createSchemaEx)

	setSchemaEx := "SET\\s+SCHEMA\\s+" + id + "[;?$|\\s+.*]"
	setSchemaRegEx := regexp.MustCompile(setSchemaEx)

	var database string
	var schema string
	lines := Tokenize(ddl)
	for _, line := range lines {
		line = strings.ToUpper(line)

		if ok, _ := regexp.Match(databaseEx, []byte(line)); ok {
			database = databaseRegEx.FindStringSubmatch(line)[1]
		}

		if ok, _ := regexp.Match(createSchemaEx, []byte(line)); ok {
			schema = createSchemaRegEx.FindStringSubmatch(line)[1]
		}

		if ok, _ := regexp.Match(setSchemaEx, []byte(line)); ok {
			schema = setSchemaRegEx.FindStringSubmatch(line)[1]
		}

		// parse view
		if ok, _ := regexp.Match(viewEx, []byte(line)); ok {
			match := viewRegEx.FindStringSubmatch(line)
			if ok, _ := regexp.Match(materialization, []byte(line)); ok {
				if ok, _ := regexp.Match(materializationTable, []byte(line)); !ok {
					views = append(views, stripQuotes(database+"."+schema+"."+match[1]))
				}
			}
		}
	}
	return views
}

func stripQuotes(s string) string {
	if strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") {
		return strings.ToLower(s[1 : len(s)-1])
	}
	return strings.ToLower(s)
}
