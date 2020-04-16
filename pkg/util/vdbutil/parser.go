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

func stripQuotes(s string) string {
	if strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") {
		return strings.ToLower(s[1 : len(s)-1])
	}
	return strings.ToLower(s)
}
