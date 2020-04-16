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
	"io/ioutil"
	"os"
	"runtime"
	"strings"

	"github.com/teiid/teiid-operator/pkg/util/logs"
	"gopkg.in/yaml.v2"
)

// Configuration --
type Configuration struct {
	TeiidSpringBootVersion string            `yaml:"teiidSpringBootVersion,omitempty"`
	TeiidVersion           string            `yaml:"teiidVersion,omitempty"`
	SpringBootVersion      string            `yaml:"springBootVersion,omitempty"`
	MavenRepositories      map[string]string `yaml:"mavenRepositories,omitempty"`
	Productized            bool              `yaml:"productized,omitempty"`
	EarlyAccess            bool              `yaml:"earlyAccess,omitempty"`
	BuildImage             BuildImage        `yaml:"buildImage,omitempty"`
	Drivers                map[string]string `yaml:"drivers,omitempty"`
	Prometheus             PrometheusConfig  `yaml:"prometheus,omitempty"`
	Labels                 map[string]string `yaml:"labels,omitempty"`
}

// BuildImage --
type BuildImage struct {
	Registry    string `yaml:"registry,omitempty"`
	ImagePrefix string `yaml:"prefix,omitempty"`
	ImageName   string `yaml:"name,omitempty"`
	Tag         string `yaml:"tag,omitempty"`
}

// PrometheusConfig --
type PrometheusConfig struct {
	MatchLabels map[string]string `yaml:"matchLabels,omitempty"`
}

// GetConfiguration --
func GetConfiguration() Configuration {

	log := logs.GetLogger("configuration")

	rootDirectory := ""
	_, filename, _, _ := runtime.Caller(0)
	if idx := strings.Index(filename, "/pkg/"); idx != -1 && !strings.HasPrefix(filename, "teiid-operator") {
		rootDirectory = filename[:idx]
	}

	var c Configuration
	yamlFile, err := ioutil.ReadFile(rootDirectory + "/conf/config.yaml")
	if err != nil {
		yamlFile, err = ioutil.ReadFile(rootDirectory + "/build/conf/config.yaml")
		if err != nil {
			log.Error("Failed to read configuration file at /conf/config.yaml", err)
			return c
		}
	}
	err = yaml.Unmarshal(yamlFile, &c)
	if err != nil {
		log.Error("Unmarshal: %v", err)
	}
	log.Info("Configuration:", c)

	// add or overide any values about prometheus scan values
	if os.Getenv("PROMETHEUS_MONITOR_LABEL_KEY") != "" && os.Getenv("PROMETHEUS_MONITOR_LABEL_VALUE") != "" {
		c.Prometheus.MatchLabels[os.Getenv("PROMETHEUS_MONITOR_LABEL_KEY")] = os.Getenv("PROMETHEUS_MONITOR_LABEL_VALUE")
	}

	return c
}
