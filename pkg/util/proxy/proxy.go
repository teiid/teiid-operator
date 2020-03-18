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
	"os"
	"regexp"
	"strings"

	"github.com/teiid/teiid-operator/pkg/util/envvar"
	corev1 "k8s.io/api/core/v1"
)

// HTTPSettings --
func HTTPSettings(envs []corev1.EnvVar) ([]corev1.EnvVar, map[string]string) {

	userDefinedProxy := envvar.Get(envs, "HTTPS_PROXY") != nil || envvar.Get(envs, "HTTP_PROXY") != nil || envvar.Get(envs, "NO_PROXY") != nil

	if !userDefinedProxy {
		clusterDefinedProxy := os.Getenv("HTTPS_PROXY") != "" || os.Getenv("HTTP_PROXY") != "" || os.Getenv("NO_PROXY") != ""
		// if cluster defined properties available
		if clusterDefinedProxy {

			if _, ok := os.LookupEnv("HTTPS_PROXY"); ok {
				envvar.SetVal(&envs, "HTTPS_PROXY", os.Getenv("HTTPS_PROXY"))
			}

			if _, ok := os.LookupEnv("HTTP_PROXY"); ok {
				envvar.SetVal(&envs, "HTTP_PROXY", os.Getenv("HTTP_PROXY"))
			}

			if _, ok := os.LookupEnv("NO_PROXY"); ok {
				envvar.SetVal(&envs, "NO_PROXY", os.Getenv("NO_PROXY"))
			}
		}
	}

	var javaProps = make(map[string]string, 0)
	if envvar.Get(envs, "HTTPS_PROXY") != nil {
		javaProps = parseHTTPProxy(envvar.Get(envs, "HTTPS_PROXY").Value)
	}

	if envvar.Get(envs, "HTTP_PROXY") != nil {
		javaProps = parseHTTPProxy(envvar.Get(envs, "HTTP_PROXY").Value)
	}

	if envvar.Get(envs, "NO_PROXY") != nil {
		nonProxyHosts := parseNoProxy(envvar.Get(envs, "NO_PROXY").Value)
		if nonProxyHosts != "" {
			javaProps["http.nonProxyHosts"] = nonProxyHosts
		}
	}
	return envs, javaProps
}

func parseNoProxy(value string) string {
	var noproxyStr string
	strs := strings.Split(value, ",")
	for _, s := range strs {
		s = strings.Trim(s, " ")
		if noproxyStr == "" {
			noproxyStr = s
		} else {
			noproxyStr = noproxyStr + "|" + s
		}
	}
	return noproxyStr
}

func parseHTTPProxy(url string) (paramsMap map[string]string) {
	// it can be in format
	//http://USERNAME:PASSWORD@PROXY_ADDRESS:PROXY_PORT
	//http://PROXY_ADDRESS:PROXY_PORT
	paramsMap = make(map[string]string)
	regEx := "^https?://(?P<proxyHost>[^@].*):(?P<proxyPort>\\d+)"
	regExWithUser := "^https?://(?P<proxyUser>[^@].*):(?P<proxyPassword>.*)@(?P<proxyHost>.*):(?P<proxyPort>\\d+)"

	var match []string
	var compRegEx *regexp.Regexp

	isHTTPS := strings.HasPrefix(url, "https")

	if m, _ := regexp.Match(regExWithUser, []byte(url)); m {
		compRegEx = regexp.MustCompile(regExWithUser)
		match = compRegEx.FindStringSubmatch(url)
	} else if m, _ := regexp.Match(regEx, []byte(url)); m {
		compRegEx = regexp.MustCompile(regEx)
		match = compRegEx.FindStringSubmatch(url)
	} else {
		return paramsMap
	}

	for i, name := range compRegEx.SubexpNames() {
		if i > 0 && i <= len(match) {
			if isHTTPS {
				paramsMap["https."+name] = match[i]
			} else {
				paramsMap["http."+name] = match[i]
			}
		}
	}
	return paramsMap
}
