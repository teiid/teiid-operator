package vdbutil

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
	"errors"
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
	commentOrSpace := "(/\\*([^*]|\\*[^/])*\\*/|--[^\r\n]*[\r\n]|\\s)+"
	serverRegEx := "CREATE" + commentOrSpace + "SERVER" + commentOrSpace + id + commentOrSpace + "(TYPE" + commentOrSpace + id + commentOrSpace + ")??FOREIGN" + commentOrSpace + "DATA" + commentOrSpace + "WRAPPER" + commentOrSpace + id
	wrapperRegEx := "CREATE" + commentOrSpace + "FOREIGN" + commentOrSpace + "DATA" + commentOrSpace + "WRAPPER" + commentOrSpace + id + commentOrSpace + "TYPE" + commentOrSpace + id

	compiledServerRegEx := regexp.MustCompile(serverRegEx)
	compiledWrapperRegEx := regexp.MustCompile(wrapperRegEx)

	lines := Tokenize(ddl)
	for _, line := range lines {
		line = strings.ToUpper(line)

		if ok, _ := regexp.Match(serverRegEx, []byte(line)); ok {
			match := compiledServerRegEx.FindStringSubmatch(line)
			sources = append(sources, DatasourceInfo{
				Name: stripQuotes(match[5]),
				Type: stripQuotes(match[22]),
			})
		}

		if ok, _ := regexp.Match(wrapperRegEx, []byte(line)); ok {
			match := compiledWrapperRegEx.FindStringSubmatch(line)
			sources = append(sources, DatasourceInfo{
				Name: stripQuotes(match[9]),
				Type: stripQuotes(match[15]),
			})
		}
	}
	return sources
}

// ShouldMaterialize --
func ShouldMaterialize(ddl string) bool {
	commentOrSpace := "(/\\*([^*]|\\*[^/])*\\*/|--[^\r\n]*[\r\n]|\\s)+"
	materialization := "(?i)MATERIALIZED" + commentOrSpace + "'?TRUE'?"
	materializationTable := "(?i)MATERIALIZED_TABLE" + commentOrSpace
	viewEx := "(?i)CREATE" + commentOrSpace + "(VIRTUAL" + commentOrSpace + ")?" + "VIEW" + commentOrSpace

	lines := Tokenize(ddl)
	for _, line := range lines {
		line = strings.ToUpper(line)
		// parse view
		if ok, _ := regexp.Match(viewEx, []byte(line)); ok {
			if ok, _ := regexp.Match(materialization, []byte(line)); ok {
				if ok, _ := regexp.Match(materializationTable, []byte(line)); !ok {
					return true
				}
			}
		}
	}
	return false
}

func stripQuotes(s string) string {
	if strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") {
		return strings.ToLower(s[1 : len(s)-1])
	}
	return strings.ToLower(s)
}

// ValidateDataSourceNames --
func ValidateDataSourceNames(ds []DatasourceInfo) error {
	re := regexp.MustCompile("^[a-zA-Z]{1}[a-zA-Z0-9_]*$")
	for _, ds := range ds {
		if !re.MatchString(ds.Name) {
			return errors.New("The datasource with name " + ds.Name + " does not confirm to naming rules. Can not contain any special characters/hyphens/periods")
		}
	}
	return nil
}
