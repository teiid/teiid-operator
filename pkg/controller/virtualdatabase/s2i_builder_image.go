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

package virtualdatabase

import (
	"context"
	"fmt"
	"strings"

	obuildv1 "github.com/openshift/api/build/v1"
	scheme "github.com/openshift/client-go/build/clientset/versioned/scheme"
	"github.com/teiid/teiid-operator/pkg/apis/vdb/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/controller/virtualdatabase/constants"
	"github.com/teiid/teiid-operator/pkg/controller/virtualdatabase/shared"
	"github.com/teiid/teiid-operator/pkg/util/envvar"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// News2IBuilderImageAction creates a new initialize action
func News2IBuilderImageAction() Action {
	return &s2iBuilderImageAction{}
}

type s2iBuilderImageAction struct {
	baseAction
}

// Name returns a common name of the action
func (action *s2iBuilderImageAction) Name() string {
	return "S2IBuilderImageAction"
}

// CanHandle tells whether this action can handle the virtualdatabase
func (action *s2iBuilderImageAction) CanHandle(vdb *v1alpha1.VirtualDatabase) bool {
	return vdb.Status.Phase == v1alpha1.ReconcilerPhaseS2IReady || vdb.Status.Phase == v1alpha1.ReconcilerPhaseBuilderImage
}

// Handle handles the virtualdatabase
func (action *s2iBuilderImageAction) Handle(ctx context.Context, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) error {

	if vdb.Status.Phase == v1alpha1.ReconcilerPhaseS2IReady {
		vdb.Status.Phase = v1alpha1.ReconcilerPhaseBuilderImage

		log.Info("Building Base builder Image")
		// Define new BuildConfig objects
		buildConfig := action.buildBC(vdb)
		// set ownerreference for service BC only
		if _, err := r.ensureImageStream(buildConfig.Name, vdb, false); err != nil {
			return err
		}

		// check to make sure the base s2i image for the build is available
		isName := buildConfig.Spec.Strategy.SourceStrategy.From.Name
		isNameSpace := buildConfig.Spec.Strategy.SourceStrategy.From.Namespace
		_, err := r.imageClient.ImageStreamTags(isNameSpace).Get(isName, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			log.Warn(isNameSpace, "/", isName, " ImageStreamTag does not exist and is required for this build.")
			return err
		} else if err != nil {
			return err
		}

		// Check if this BC already exists
		bc, err := r.buildClient.BuildConfigs(buildConfig.Namespace).Get(buildConfig.Name, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			log.Info("Creating a new BuildConfig ", buildConfig.Name, " in namespace ", buildConfig.Namespace)
			bc, err = r.buildClient.BuildConfigs(buildConfig.Namespace).Create(&buildConfig)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		log.Info("Created BuildConfig")

		// Trigger first build of "builder" and binary BCs
		if bc.Status.LastVersion == 0 {
			log.Info("triggering the base builder image build")
			if err = action.triggerBuild(*bc, vdb, r); err != nil {
				return err
			}
		}
	} else if vdb.Status.Phase == v1alpha1.ReconcilerPhaseBuilderImage {
		builds := &obuildv1.BuildList{}
		options := metav1.ListOptions{
			FieldSelector: "metadata.namespace=" + vdb.ObjectMeta.Namespace,
			LabelSelector: "buildconfig=" + constants.BuilderImageTargetName,
		}

		builds, err := r.buildClient.Builds(vdb.ObjectMeta.Namespace).List(options)
		if err != nil {
			return err
		}

		for _, build := range builds.Items {
			// set status of the build
			if build.Status.Phase == obuildv1.BuildPhaseComplete && vdb.Status.Phase != v1alpha1.ReconcilerPhaseBuilderImageFinished {
				vdb.Status.Phase = v1alpha1.ReconcilerPhaseBuilderImageFinished
			} else if (build.Status.Phase == obuildv1.BuildPhaseError ||
				build.Status.Phase == obuildv1.BuildPhaseFailed ||
				build.Status.Phase == obuildv1.BuildPhaseCancelled) && vdb.Status.Phase != v1alpha1.ReconcilerPhaseBuilderImageFailed {
				vdb.Status.Phase = v1alpha1.ReconcilerPhaseBuilderImageFailed
			} else if build.Status.Phase == obuildv1.BuildPhaseRunning && vdb.Status.Phase != v1alpha1.ReconcilerPhaseBuilderImage {
				vdb.Status.Phase = v1alpha1.ReconcilerPhaseBuilderImage
			}
		}
	}
	return nil
}

// newBCForCR returns a BuildConfig with the same name/namespace as the cr
func (action *s2iBuilderImageAction) buildBC(vdb *v1alpha1.VirtualDatabase) obuildv1.BuildConfig {
	bc := obuildv1.BuildConfig{}
	images := constants.RuntimeImageDefaults[vdb.Spec.Runtime]
	env := []corev1.EnvVar{}
	envvar.SetVal(&env, "DEPLOYMENTS_DIR", "/opt/jboss") // this is avoid copying the jar file
	envvar.SetVal(&env, "MAVEN_ARGS_APPEND", "-Dmaven.compiler.source=1.8 -Dmaven.compiler.target=1.8")
	envvar.SetVal(&env, "ARTIFACT_DIR", "target/")

	incremental := true
	for _, imageDefaults := range images {
		if imageDefaults.BuilderImage {
			builderName := constants.BuilderImageTargetName
			bc = obuildv1.BuildConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      builderName,
					Namespace: vdb.ObjectMeta.Namespace,
				},
			}
			bc.SetGroupVersionKind(obuildv1.SchemeGroupVersion.WithKind("BuildConfig"))
			bc.Spec.Source.Binary = &obuildv1.BinaryBuildSource{}
			bc.Spec.Output.To = &corev1.ObjectReference{Name: strings.Join([]string{builderName, "latest"}, ":"), Kind: "ImageStreamTag"}
			bc.Spec.Strategy.Type = obuildv1.SourceBuildStrategyType
			bc.Spec.Strategy.SourceStrategy = &obuildv1.SourceBuildStrategy{
				Incremental: &incremental,
				Env:         env,
				From: corev1.ObjectReference{
					Name:      fmt.Sprintf("%s:%s", imageDefaults.ImageStreamName, imageDefaults.ImageStreamTag),
					Namespace: imageDefaults.ImageStreamNamespace,
					Kind:      "ImageStreamTag",
				},
			}
		}
	}
	return bc
}

// triggerBuild triggers a BuildConfig to start a new build
func (action *s2iBuilderImageAction) triggerBuild(bc obuildv1.BuildConfig, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) error {
	log := log.With("kind", "BuildConfig", "name", bc.GetName(), "namespace", bc.GetNamespace())
	buildConfig, err := r.buildClient.BuildConfigs(bc.Namespace).Get(bc.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	files := map[string]string{}
	files["/pom.xml"] = action.pomFile()
	files["/src/main/resources/teiid.ddl"] = action.ddlFile()

	tarReader, err := shared.Tar(files)
	if err != nil {
		return err
	}

	// do the binary build
	binaryBuildRequest := obuildv1.BinaryBuildRequestOptions{ObjectMeta: metav1.ObjectMeta{Name: buildConfig.Name}}
	binaryBuildRequest.SetGroupVersionKind(obuildv1.SchemeGroupVersion.WithKind("BinaryBuildRequestOptions"))
	log.Info("Triggering binary build ", buildConfig.Name)
	err = r.buildClient.RESTClient().Post().
		Namespace(vdb.ObjectMeta.Namespace).
		Resource("buildconfigs").
		Name(buildConfig.Name).
		SubResource("instantiatebinary").
		Body(tarReader).
		VersionedParams(&binaryBuildRequest, scheme.ParameterCodec).
		Do().
		Into(&obuildv1.Build{})
	if err != nil {
		return err
	}
	return nil
}

func (action *s2iBuilderImageAction) ddlFile() string {
	return `CREATE DATABASE customer OPTIONS (ANNOTATION 'Customer VDB');	USE DATABASE customer;
	SET NAMESPACE 'http://teiid.org/rest' AS REST;
	CREATE FOREIGN DATA WRAPPER h2;
	CREATE SERVER mydb TYPE 'NONE' FOREIGN DATA WRAPPER h2 OPTIONS ("resource-name" 'mydb');`
}

func (action *s2iBuilderImageAction) pomFile() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
	<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
	  xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">
	  <modelVersion>4.0.0</modelVersion>
	  <groupId>org.teiid</groupId>
	  <artifactId>base-image</artifactId>
	  <name>base-image</name>
	  <description>Base Image</description>
	  <packaging>jar</packaging>
	  <version>1.0.0</version>
	
	  <properties>
		   <version.teiid.springboot>1.2.0-SNAPSHOT</version.teiid.springboot>
	  </properties>
	
	  <repositories>
		 <repository>
		   <id>snapshots-repo</id>
		   <name>snapshots-repo</name>
		   <url>https://oss.sonatype.org/content/repositories/snapshots</url>
		   <releases><enabled>false</enabled></releases>
		   <snapshots><enabled>true</enabled></snapshots>
		 </repository>
	  </repositories>
	  <pluginRepositories>
		<pluginRepository>
			<id>snapshots-repo</id>
			<name>snapshots-repo</name>
			<url>https://oss.sonatype.org/content/repositories/snapshots</url>
			<releases><enabled>false</enabled></releases>
			<snapshots><enabled>true</enabled></snapshots>
		</pluginRepository>
      </pluginRepositories>	  
	
	  <dependencies>
		<dependency>
		  <groupId>org.teiid</groupId>
		  <artifactId>teiid-spring-boot-starter</artifactId>
		  <version>${version.teiid.springboot}</version>
		</dependency>
		<dependency>
		  <groupId>org.teiid</groupId>
		  <artifactId>spring-odata</artifactId>
		  <version>${version.teiid.springboot}</version>
		</dependency>
		<dependency>
		  <groupId>org.teiid</groupId>
		  <artifactId>spring-keycloak</artifactId>
		  <version>${version.teiid.springboot}</version>
		</dependency>
		<dependency>
		  <groupId>org.teiid</groupId>
		  <artifactId>spring-data-excel</artifactId>
		  <version>${version.teiid.springboot}</version>
		</dependency>
		<dependency>
		  <groupId>org.teiid</groupId>
		  <artifactId>spring-data-google</artifactId>
		  <version>${version.teiid.springboot}</version>
		</dependency>
		<dependency>
		  <groupId>org.teiid</groupId>
		  <artifactId>spring-data-mongodb</artifactId>
		  <version>${version.teiid.springboot}</version>
		</dependency>
		<dependency>
		  <groupId>org.teiid</groupId>
		  <artifactId>spring-data-rest</artifactId>
		  <version>${version.teiid.springboot}</version>
		</dependency>
		<dependency>
		  <groupId>org.teiid</groupId>
		  <artifactId>spring-data-salesforce</artifactId>
		  <version>${version.teiid.springboot}</version>
		</dependency>
		<dependency>
			<groupId>com.h2database</groupId>
			<artifactId>h2</artifactId>
			<version>1.4.199</version>
		</dependency>
		<dependency>
		  <groupId>io.opentracing.contrib</groupId>
		  <artifactId>opentracing-spring-jaeger-web-starter</artifactId>
		  <version>1.0.1</version>
		</dependency>
		<dependency>    
		  <groupId>org.springframework.boot</groupId>   
		  <artifactId>spring-boot-starter-actuator</artifactId>
		  <version>2.1.3.RELEASE</version> 
		</dependency>
	  </dependencies>
	
	  <build>
		<plugins>
		  <plugin>
			<groupId>org.teiid</groupId>
			<artifactId>vdb-codegen-plugin</artifactId>
			<version>${version.teiid.springboot}</version>
		   <configuration>
			  <packageName>com.example</packageName>
			  <generateApplicationClass>true</generateApplicationClass>
			</configuration>
			<executions>
			  <execution>
				<goals>
				  <goal>vdb-codegen</goal>
				</goals>
			  </execution>
			</executions>
		  </plugin>
		  <plugin>
			<groupId>org.springframework.boot</groupId>
			<artifactId>spring-boot-maven-plugin</artifactId>
			<executions>
			  <execution>
				<goals>
				  <goal>repackage</goal>
				</goals>
			  </execution>
			</executions>
		  </plugin>
		</plugins>
	  </build>
	</project>`
}
