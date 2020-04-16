package conf

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
	"encoding/json"
	"io/ioutil"
	"runtime"
	"strings"

	"github.com/teiid/teiid-operator/pkg/util/logs"
)

// ConnectionFactory --
type ConnectionFactory struct {
	Name                     string   `json:"name,omitempty"`
	DriverNames              []string `json:"driverNames,omitempty"`
	TranslatorName           string   `json:"translatorName,omitempty"`
	Dialect                  string   `json:"dialect,omitempty"`
	Gav                      []string `json:"gav,omitempty"`
	SpringBootPropertyPrefix string   `json:"springBootPropertyPrefix,omitempty"`
}

// ConnectionFactoryList --
type ConnectionFactoryList struct {
	Items map[string]ConnectionFactory `json:"items"`
}

// GetConnectionFactories --
func GetConnectionFactories() map[string]ConnectionFactory {
	log := logs.GetLogger("configuration")

	rootDirectory := ""

	_, filename, _, _ := runtime.Caller(0)
	if idx := strings.Index(filename, "/pkg/"); idx != -1 {
		rootDirectory = filename[:idx]
	}

	jsonFile, err := ioutil.ReadFile(rootDirectory + "/conf/connection_factories.json")
	if err != nil {
		jsonFile, err = ioutil.ReadFile(rootDirectory + "/build/conf/connection_factories.json")
		if err != nil {
			log.Error("Failed to read Connection Factories Configuration file at /conf/connection-factories.json", err)
			return map[string]ConnectionFactory{}
		}
	}
	return loadConnectionFactories(jsonFile)
}

// LoadConnectionFactories --
func loadConnectionFactories(contents []byte) map[string]ConnectionFactory {
	log := logs.GetLogger("configuration")
	var c ConnectionFactoryList
	err := json.Unmarshal(contents, &c)
	if err != nil {
		log.Error("Unmarshal: %v", err)
	}
	log.Debug("Connection Factories:", c)
	return c.Items
}
