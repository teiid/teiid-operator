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
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/teiid/teiid-operator/pkg/util/maven"
	"github.com/teiid/teiid-operator/pkg/util/proxy"
	"github.com/teiid/teiid-operator/pkg/util/vdbutil"

	obuildv1 "github.com/openshift/api/build/v1"
	scheme "github.com/openshift/client-go/build/clientset/versioned/scheme"
	"github.com/teiid/teiid-operator/pkg/apis/teiid/v1alpha1"
	"github.com/teiid/teiid-operator/pkg/controller/virtualdatabase/constants"
	"github.com/teiid/teiid-operator/pkg/util"
	"github.com/teiid/teiid-operator/pkg/util/envvar"
	"github.com/teiid/teiid-operator/pkg/util/image"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// NewServiceImageAction creates a new initialize action
func NewServiceImageAction() Action {
	return &serviceImageAction{}
}

type serviceImageAction struct {
	baseAction
}

// Name returns a common name of the action
func (action *serviceImageAction) Name() string {
	return "ServiceImageAction"
}

// CanHandle tells whether this action can handle the virtualdatabase
func (action *serviceImageAction) CanHandle(vdb *v1alpha1.VirtualDatabase) bool {
	return vdb.Status.Phase == v1alpha1.ReconcilerPhaseBuilderImageFinished ||
		vdb.Status.Phase == v1alpha1.ReconcilerPhaseServiceImage
}

func (action *serviceImageAction) Handle(ctx context.Context, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) error {
	if vdb.Status.Phase == v1alpha1.ReconcilerPhaseBuilderImageFinished {
		vdb.Status.Phase = v1alpha1.ReconcilerPhaseServiceImage
		err := action.buildServiceImage(ctx, vdb, r)
		if err != nil {
			vdb.Status.Phase = v1alpha1.ReconcilerPhaseError
			vdb.Status.Failure = err.Error()
			return err
		}
	} else if vdb.Status.Phase == v1alpha1.ReconcilerPhaseServiceImage {
		return action.monitorServiceImage(ctx, vdb, r)
	}
	return nil
}

// Handle handles the virtualdatabase
func (action *serviceImageAction) buildServiceImage(ctx context.Context, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) error {
	// check for the VDB source type
	if vdb.Spec.Build.Source.DDL == "" && vdb.Spec.Build.Source.Maven == "" {
		return errors.New("Only Git and DDL Content based, Maven based VDBs are allowed, none of these types are defined")
	}

	// Define new BuildConfig objects
	if _, err := image.EnsureImageStream(vdb.ObjectMeta.Name, vdb.ObjectMeta.Namespace, true, vdb, r.imageClient, r.client.GetScheme()); err != nil {
		return err
	}

	// Check if this BC already exists
	bc, err := r.buildClient.BuildConfigs(vdb.ObjectMeta.Namespace).Get(vdb.ObjectMeta.Name, metav1.GetOptions{})
	if err != nil && apierr.IsNotFound(err) {
		log.Info("Creating a new BuildConfig ", vdb.ObjectMeta.Name, " in namespace ", vdb.ObjectMeta.Namespace)
		// set ownerreference for service BC only
		buildConfig, err := action.newServiceBC(vdb)
		if err != nil {
			return err
		}
		err = controllerutil.SetControllerReference(vdb, &buildConfig, r.client.GetScheme())
		if err != nil {
			log.Error(err)
		}
		bc, err = r.buildClient.BuildConfigs(buildConfig.Namespace).Create(&buildConfig)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// check the digest of the previous build, if does not match rebuild
	digest := envvar.Get(bc.Spec.Strategy.SourceStrategy.Env, "DIGEST")

	// Trigger first build of "builder" and binary BCs
	if bc.Status.LastVersion == 0 || digest.Value != vdb.Status.Digest {
		envvar.SetVal(&bc.Spec.Strategy.SourceStrategy.Env, "DIGEST", vdb.Status.Digest)

		if err := r.client.Update(ctx, bc); err != nil {
			return err
		}

		var payload map[string]string
		if isFatJarBuild(vdb) {
			files, err := buildJarBasedPayload(vdb, r)
			if err != nil {
				return err
			}
			payload = files
		} else {
			files, err := buildVdbBasedPayload(ctx, vdb, r)
			if err != nil {
				return err
			}
			payload = files
		}

		if err = action.triggerBuild(*bc, payload, r); err != nil {
			return err
		}
	}
	return nil
}

func isFatJarBuild(vdb *v1alpha1.VirtualDatabase) bool {
	if vdb.Spec.Build.Source.Maven != "" {
		if !strings.Contains(vdb.Spec.Build.Source.Maven, ":vdb:") {
			return true
		}
	}
	return false
}

func (action *serviceImageAction) monitorServiceImage(ctx context.Context, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) error {
	builds := &obuildv1.BuildList{}
	options := metav1.ListOptions{
		FieldSelector: "metadata.namespace=" + vdb.ObjectMeta.Namespace,
		LabelSelector: "buildconfig=" + vdb.ObjectMeta.Name,
	}

	builds, err := r.buildClient.Builds(vdb.ObjectMeta.Namespace).List(options)
	if err != nil {
		return err
	}

	// there could be multiple builds, find the latest one as that is one
	// we are currently running
	build := obuildv1.Build{}
	maxBuildNumber := 0
	if len(builds.Items) >= 1 {
		for _, b := range builds.Items {
			i, _ := strconv.Atoi(b.ObjectMeta.Annotations["openshift.io/build.number"])
			if i > maxBuildNumber {
				maxBuildNumber = i
				build = b
			}
		}
	}

	// set status of the build
	if build.Status.Phase == obuildv1.BuildPhaseComplete {
		vdb.Status.Phase = v1alpha1.ReconcilerPhaseServiceImageFinished
	} else if build.Status.Phase == obuildv1.BuildPhaseError ||
		build.Status.Phase == obuildv1.BuildPhaseFailed ||
		build.Status.Phase == obuildv1.BuildPhaseCancelled {
		vdb.Status.Phase = v1alpha1.ReconcilerPhaseServiceImageFailed
	} else if build.Status.Phase == obuildv1.BuildPhaseRunning {
		vdb.Status.Phase = v1alpha1.ReconcilerPhaseServiceImage
	}
	return nil
}

func defaultBuildOptions() string {
	str := strings.Join([]string{
		" ",
		"-Djava.net.useSystemProxies=true",
		"-Dmaven.compiler.source=1.8",
		"-Dmaven.compiler.target=1.8",
		"-DskipTests",
		"-Dmaven.javadoc.skip=true",
		"-Dmaven.site.skip=true",
		"-Dmaven.source.skip=true",
		"-Djacoco.skip=true",
		"-Dcheckstyle.skip=true",
		"-Dfindbugs.skip=true",
		"-Dpmd.skip=true",
		"-Dfabric8.skip=true",
		"-Dbasepom.check.skip-dependency=true",
		"-Dbasepom.check.skip-duplicate-finder=true",
		"-Dbasepom.check.skip-spotbugs=true",
		"-Dbasepom.check.skip-dependency-versions-check=true",
		"-Dbasepom.check.skip-dependency-management=true",
		"-Dbasepom.check.skip-dependency-scope=true",
		"-Dbasepom.check.skip-license=true",
		"-Dbasepom.check.skip-pmd=true",
		"-Dbasepom.check.skip-checkstyle=true",
		"-Dbasepom.check.skip-javadoc=true",
		"-e",
		"-B",
	}, " ")
	return str
}

func (action *serviceImageAction) newServiceBC(vdb *v1alpha1.VirtualDatabase) (obuildv1.BuildConfig, error) {
	baseImage := strings.Join([]string{constants.BuilderImageTargetName, "latest"}, ":")

	envs := envvar.Clone(vdb.Spec.Build.Env)

	// handle proxy settings
	envs, jp := proxy.HTTPSettings(envs)
	var javaProperties string
	for k, v := range jp {
		javaProperties = javaProperties + "-D" + k + "=" + v + " "
	}

	str := defaultBuildOptions()

	// set it back original default
	envvar.SetVal(&envs, "DEPLOYMENTS_DIR", "/deployments")
	// this below is add clean, to remove the previous jar file in target from builder image
	envvar.SetVal(&envs, "MAVEN_ARGS", "clean package "+javaProperties+str)
	envvar.SetVal(&envs, "DIGEST", vdb.Status.Digest)

	// build config
	bc := obuildv1.BuildConfig{}
	bc = obuildv1.BuildConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vdb.ObjectMeta.Name,
			Namespace: vdb.ObjectMeta.Namespace,
			Labels: map[string]string{
				"app": vdb.ObjectMeta.Name,
			},
		},
	}
	bc.SetGroupVersionKind(obuildv1.SchemeGroupVersion.WithKind("BuildConfig"))
	bc.Spec.Output.To = &corev1.ObjectReference{Name: strings.Join([]string{vdb.ObjectMeta.Name, "latest"}, ":"), Kind: "ImageStreamTag"}

	// for some reason "vdb.Spec.Build.Source" comes in as empty object rather than nil
	// create the source build object
	inc := false
	bc.Spec.Source.Type = obuildv1.BuildSourceBinary
	bc.Spec.Source.Binary = &obuildv1.BinaryBuildSource{}
	bc.Spec.Strategy.Type = obuildv1.SourceBuildStrategyType
	bc.Spec.Strategy.SourceStrategy = &obuildv1.SourceBuildStrategy{
		From:        corev1.ObjectReference{Name: baseImage, Kind: "ImageStreamTag"},
		ForcePull:   false,
		Incremental: &inc,
		Env:         envs,
	}

	if vdb.Spec.Build.Source.DDL != "" {
		log.Info("DDL based build is chosen..")
	} else if vdb.Spec.Build.Source.Maven != "" {
		if strings.Contains(vdb.Spec.Build.Source.Maven, ":vdb:") {
			log.Info("Maven based VDB build is chosen..")
		} else {
			log.Info("Maven Repo Fat Jar based Docker build is chosen..")
		}
	}

	// when trigger is defined the build starts immediately without the
	// binary, using previous base build's source directory which is not
	// intended result, so do not add triggers
	return bc, nil
}

func buildJarBasedPayload(vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) (map[string]string, error) {
	files := map[string]string{}

	//Binary build, generate the pom file
	pom, err := GenerateJarPom(vdb)
	if err != nil {
		return files, err
	}

	jarDependency, err := maven.ParseGAV(vdb.Spec.Build.Source.Maven)
	if err != nil {
		log.Error("The Maven based JAR is provided in bad format", err)
		return files, err
	}
	// configure pom.xml to copy the teiid.vdb into classpath
	addCopyPlugIn(jarDependency, "jar", "app.jar", "${project.build.directory}", &pom)

	pomContent, err := maven.EncodeXML(pom)
	if err != nil {
		return files, err
	}

	log.Info("Pom file generated %s", pomContent)

	files["/pom.xml"] = pomContent
	files["/src/main/resources/prometheus-config.yml"] = PrometheusConfig(r.client, vdb.ObjectMeta.Namespace)

	return files, nil
}

func buildVdbBasedPayload(ctx context.Context, vdb *v1alpha1.VirtualDatabase, r *ReconcileVirtualDatabase) (map[string]string, error) {
	files := map[string]string{}

	// add OpenAPI document
	addOpenAPI := false
	if len(vdb.Spec.Build.Source.OpenAPI) > 0 {
		files["/src/main/resources/openapi.json"] = vdb.Spec.Build.Source.OpenAPI
		addOpenAPI = true
	}

	var ddlStr string
	ddlStr, err := vdbutil.FetchDdl(vdb, "/tmp/teiid.vdb")
	if err != nil {
		log.Error("failed to read VDB from maven ", err)
		return files, err
	}

	// check to see if Infinispan is available
	ignoreIspn := IgnoreCacheStore(r, vdb)

	// Check to see if the VDB has materialization annoations, if yes then generate
	// materialization schema
	vdbNeedsCacheStore := vdbutil.ShouldMaterialize(ddlStr) && !ignoreIspn

	// read data source names from the DDL and make sure they follow the naming restrictions
	dataSourceInfos := vdbutil.ParseDataSourcesInfoFromDdl(ddlStr)
	err = vdbutil.ValidateDataSourceNames(dataSourceInfos)
	if err != nil {
		log.Error("Data Source names are not in valid format ", err)
		return files, err
	}

	//Binary build, generate the pom file
	pom, err := GenerateVdbPom(vdb, dataSourceInfos, false, addOpenAPI, vdbNeedsCacheStore)
	if err != nil {
		return files, err
	}
	vdbFile := "teiid.vdb-file=teiid.ddl"

	addVdbCodeGenPlugIn(&pom, "/tmp/src/src/main/resources/teiid.ddl", vdbNeedsCacheStore, vdb.Status.Version)
	files["/src/main/resources/teiid.ddl"] = ddlStr

	// if the materialization is in play then we have a new vdb file
	if vdbNeedsCacheStore {
		vdbFile = "teiid.vdb-file=materialized.ddl"
	}

	pomContent, err := maven.EncodeXML(pom)
	if err != nil {
		return files, err
	}

	log.Debugf("Pom file generated %s", pomContent)

	// build default maven repository
	repositories := []maven.Repository{}
	mavenRepos := constants.GetMavenRepositories(vdb)
	for k, v := range mavenRepos {
		repositories = append(repositories, maven.NewRepository(v+"@id="+k))
	}
	// read the settings file
	settingsContent, err := readMavenSettingsFile(ctx, vdb, r, repositories)
	if err != nil {
		log.Debugf("Failed reading the settings.xml file for vdb %s", vdb.ObjectMeta.Name)
		return files, err
	}

	log.Debugf("settings.xml file generated %s", settingsContent)

	files["/configuration/settings.xml"] = settingsContent
	files["/pom.xml"] = pomContent
	files["/src/main/resources/prometheus-config.yml"] = PrometheusConfig(r.client, vdb.ObjectMeta.Namespace)
	files["/src/main/resources/application.properties"] = applicationProperties(vdbFile, vdb.ObjectMeta.Name)

	return files, nil
}

// triggerBuild triggers a BuildConfig to start a new build
func (action *serviceImageAction) triggerBuild(bc obuildv1.BuildConfig, files map[string]string, r *ReconcileVirtualDatabase) error {
	log := log.With("kind", "BuildConfig", "name", bc.GetName(), "namespace", bc.GetNamespace())
	buildConfig, err := r.buildClient.BuildConfigs(bc.Namespace).Get(bc.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if buildConfig.Spec.Source.Type == obuildv1.BuildSourceBinary {
		log.Info("starting the binary build for service image ")
		tarReader, err := util.Tar(files)
		if err != nil {
			return err
		}
		isName := buildConfig.Spec.Strategy.SourceStrategy.From.Name
		_, err = r.imageClient.ImageStreamTags(buildConfig.Namespace).Get(isName, metav1.GetOptions{})
		if err != nil && apierr.IsNotFound(err) {
			log.Warn(isName, " ImageStreamTag does not exist yet and is required for this build.")
		} else if err != nil {
			return err
		} else {
			binaryBuildRequest := obuildv1.BinaryBuildRequestOptions{ObjectMeta: metav1.ObjectMeta{Name: buildConfig.Name}}
			binaryBuildRequest.SetGroupVersionKind(obuildv1.SchemeGroupVersion.WithKind("BinaryBuildRequestOptions"))
			log.Info("Triggering binary build ", buildConfig.Name)
			err = r.buildClient.RESTClient().Post().
				Namespace(bc.ObjectMeta.Namespace).
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
		}
	} else {
		buildRequest := obuildv1.BuildRequest{ObjectMeta: metav1.ObjectMeta{Name: buildConfig.Name}}
		buildRequest.SetGroupVersionKind(obuildv1.SchemeGroupVersion.WithKind("BuildRequest"))
		buildRequest.TriggeredBy = []obuildv1.BuildTriggerCause{{Message: fmt.Sprintf("Triggered by %s operator", "VirtualDatabase")}}
		log.Info("Triggering build ", buildConfig.Name)
		_, err := r.buildClient.BuildConfigs(buildConfig.Namespace).Instantiate(buildConfig.Name, &buildRequest)
		if err != nil {
			return err
		}
	}
	return nil
}

func applicationProperties(vdbProperty string, vdbName string) string {
	str := strings.Join([]string{
		"logging.level.io.jaegertracing.internal.reporters=WARN",
		"logging.level.i.j.internal.reporters.LoggingReporter=WARN",
		"logging.level.org.teiid.SECURITY=WARN",
		"spring.main.allow-bean-definition-overriding=true",
		"teiid.jdbc-secure-enable=true",
		"teiid.pg-secure-enable=true",
		"teiid.jdbc-enable=true",
		"teiid.pg-enable=true",
		"teiid.ssl.keyStoreType=pkcs12",
		"teiid.ssl.keyStoreFileName=" + constants.KeystoreLocation + "/" + constants.KeystoreName,
		"teiid.ssl.keyStorePassword=" + constants.KeystorePassword,
		"teiid.ssl.trustStoreFileName=" + constants.KeystoreLocation + "/" + constants.TruststoreName,
		"teiid.ssl.trustStorePassword=" + constants.KeystorePassword,
		"keycloak.truststore=" + constants.KeystoreLocation + "/" + constants.TruststoreName,
		"keycloak.truststore-password=" + constants.KeystorePassword,
		"springfox.documentation.swagger.v2.path=/openapi.json",
		"spring.teiid.model.package=io.integration",
		"spring.application.name=" + vdbName,
		"management.health.mongo.enabled=false",
		"management.health.db.enabled=false",
		vdbProperty,
	}, "\n")
	return str
}
