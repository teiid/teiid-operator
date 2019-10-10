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

	"github.com/teiid/teiid-operator/pkg/util/logs"
	"gopkg.in/yaml.v2"
)

// Configuration --
type Configuration struct {
	TeiidSpringBootVersion string            `yaml:"teiidSpringBootVersion,omitempty"`
	SpringBootVersion      string            `yaml:"springBootVersion,omitempty"`
	MavenRepositories      map[string]string `yaml:"mavenRepositories,omitempty"`
	Productized            bool              `yaml:"productized,omitempty"`
	EarlyAccess            bool              `yaml:"earlyAccess,omitempty"`
	BuildImage             BuildImage        `yaml:"buildImage,omitempty"`
	Drivers                map[string]string `yaml:"drivers,omitempty"`
}

// BuildImage --
type BuildImage struct {
	Registry    string `yaml:"registry,omitempty"`
	ImagePrefix string `yaml:"prefix,omitempty"`
	ImageName   string `yaml:"name,omitempty"`
	Tag         string `yaml:"tag,omitempty"`
}

// GetConfiguration --
func GetConfiguration() Configuration {
	log := logs.GetLogger("configuration")

	var c Configuration
	yamlFile, err := ioutil.ReadFile("/conf/config.yaml")
	if err != nil {
		log.Error("Failed to read configuration file ", err)
	}
	err = yaml.Unmarshal(yamlFile, &c)
	if err != nil {
		log.Error("Unmarshal: %v", err)
	}
	log.Info("Configuration:", c)
	return c
}
