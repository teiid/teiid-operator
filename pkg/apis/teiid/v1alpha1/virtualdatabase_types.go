package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VirtualDatabaseSpec defines the desired state of VirtualDatabase
// +k8s:openapi-gen=true
type VirtualDatabaseSpec struct {
	// Number Of deployment units required
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Replicas"
	Replicas *int32 `json:"replicas,omitempty"`
	// Environment properties required for deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Properties"
	Env []corev1.EnvVar `json:"env,omitempty"`
	// memory, disk cpu requirements
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Resources"
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// S2I Build configuration
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="VDB Build"
	Build VirtualDatabaseBuildObject `json:"build"`
	// Jaeger instance to use to push the tracing information
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Jaeger Name"
	Jaeger string `json:"jaeger,omitempty"`
	// DataSources configuration for this Virtual Database
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Datasources Configuration"
	DataSources []DataSourceObject `json:"datasources,omitempty"`
	// Defines the services (LoadBalancer, NodePort, 3scale) to expose
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Services Created"
	Expose ExposeObject `json:"expose,omitempty"`
}

// VirtualDatabaseStatus defines the observed state of VirtualDatabase
// +k8s:openapi-gen=true
type VirtualDatabaseStatus struct {

	// The current phase of the build the operator deployment is running
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.displayName="Phase"
	Phase ReconcilerPhase `json:"phase,omitempty"`

	// Digest value of the vdb
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.displayName="SHA Of the VDB"
	Digest string `json:"digest,omitempty"`

	// ConfigDigest value of the vdb
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.displayName="SHA Of the Configuration"
	ConfigDigest string `json:"configdigest,omitempty"`

	// Failure message if deployment ended in failure
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.displayName="Failure Message"
	Failure string `json:"failure,omitempty"`

	// Route information that is exposed for clients
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.displayName="Route Exposed for OData"
	Route string `json:"route,omitempty"`

	// Deployed vdb version.
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.displayName="Version Of the VDB deployed"
	Version string `json:"version,omitempty"`

	// Deployed vdb version.
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.displayName="CacheStore In use"
	CacheStore string `json:"cachestore,omitempty"`
}

// OpenShiftObject ...
type OpenShiftObject interface {
	metav1.Object
	runtime.Object
}

// VirtualDatabaseBuildObject Data to define how to build an application from source
// +k8s:openapi-gen=true
type VirtualDatabaseBuildObject struct {
	// Environment properties set build purpose
	Env []corev1.EnvVar `json:"env,omitempty"`
	// VDB Source details
	Source Source `json:"source,omitempty"`
}

// Source VDB coordinates to locate the source code to build
// +k8s:openapi-gen=true
type Source struct {
	// Deployed vdb version. For embedded DDL version this will be implicitly provided when ignored, for maven based vdb the maven version is always
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.displayName="Version Of the VDB"
	Version string `json:"version,omitempty"`

	// DDL based VDB
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.displayName="DDL Of the VDB"
	DDL string `json:"ddl,omitempty"`

	// A VDB defined in GAV format
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.displayName="Maven Coordinates for VDB"
	Maven string `json:"maven,omitempty"`

	// Open API contract that is exposed by the VDB
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.displayName="OpenAPI of exposed"
	OpenAPI string `json:"openapi,omitempty"`

	// List of maven dependencies for the build in GAV format
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.displayName="Maven Dependencies for VDB"
	Dependencies []string `json:"dependencies,omitempty"`

	// Custom maven repositories that need to be used for the S2I build
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.displayName="Custom Maven Repositories"
	MavenRepositories map[string]string `json:"mavenRepositories,omitempty"`
}

// ValueSource --
// +k8s:openapi-gen=true
type ValueSource struct {
	// Selects a key of a ConfigMap.
	ConfigMapKeyRef *corev1.ConfigMapKeySelector `json:"configMapKeyRef,omitempty"`
	// Selects a key of a secret.
	SecretKeyRef *corev1.SecretKeySelector `json:"secretKeyRef,omitempty"`
}

// ExposeType - type of service to be exposed
type ExposeType string

const (
	// LoadBalancer type service to expose
	LoadBalancer ExposeType = "LoadBalancer"
	// NodePort type service to expose
	NodePort ExposeType = "NodePort"
	// Route Openshift Route to expose
	Route ExposeType = "Route"
	// ExposeVia3scale just service, not route
	ExposeVia3scale ExposeType = "ExposeVia3scale"
)

// ExposeObject - defines the services that need to be exposed
// +k8s:openapi-gen=true
type ExposeObject struct {
	// Types of services to expose
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="types"
	Types []ExposeType `json:"types,omitempty"`
}

// DataSourceObject - define the datasources that this Virtual Database integrates
// +k8s:openapi-gen=true
type DataSourceObject struct {
	// Name of the Data Source
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Properties"
	Name string `json:"name,omitempty"`
	// Type of Data Source. ex: Oracle, PostgreSQL, MySQL, Salesforce etc.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Properties"
	Type string `json:"type,omitempty"`
	// Properties required for Data Source connection
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Properties"
	Properties []corev1.EnvVar `json:"properties,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualDatabase is the Schema for the virtualdatabases API
// +k8s:openapi-gen=true
// +operator-sdk:gen-csv:customresourcedefinitions.displayName="Virtual Database Application"
// +kubebuilder:resource:path=virtualdatabases,shortName=vdb;vdbs
// +kubebuilder:singular=virtualdatabase
type VirtualDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Virtual Database specification
	Spec VirtualDatabaseSpec `json:"spec,omitempty"`
	// Virtual Database Status
	Status VirtualDatabaseStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualDatabaseList contains a list of VirtualDatabase
type VirtualDatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualDatabase `json:"items"`
}

// ReconcilerPhase --
type ReconcilerPhase string

const (
	// VirtualDatabaseKind --
	VirtualDatabaseKind string = "VirtualDatabase"

	// ReconcilerPhaseInitial --
	ReconcilerPhaseInitial ReconcilerPhase = ""

	// ReconcilerPhaseCreateCacheStore --
	ReconcilerPhaseCreateCacheStore ReconcilerPhase = "Creating Cache Store"

	// ReconcilerPhaseS2IReady --
	ReconcilerPhaseS2IReady ReconcilerPhase = "Ready For S2I"

	// ReconcilerPhaseBuilderImage --
	ReconcilerPhaseBuilderImage ReconcilerPhase = "Building Base Builder Image"
	// ReconcilerPhaseBuilderImageFinished --
	ReconcilerPhaseBuilderImageFinished ReconcilerPhase = "Builder Image Finished"
	// ReconcilerPhaseBuilderImageFailed --
	ReconcilerPhaseBuilderImageFailed ReconcilerPhase = "Builder Image Failed"

	// ReconcilerPhaseServiceImage --
	ReconcilerPhaseServiceImage ReconcilerPhase = "Building Service Image"
	// ReconcilerPhaseServiceImageFinished --
	ReconcilerPhaseServiceImageFinished ReconcilerPhase = "Service Image Finished"
	// ReconcilerPhaseServiceImageFailed --
	ReconcilerPhaseServiceImageFailed ReconcilerPhase = "Service Image Failed"

	//ReconcilerPhaseServiceCreated --
	ReconcilerPhaseServiceCreated ReconcilerPhase = "Service Created"

	//ReconcilerPhaseKeystoreCreated --
	ReconcilerPhaseKeystoreCreated ReconcilerPhase = "Keystore Created"

	// ReconcilerPhaseDeploying --
	ReconcilerPhaseDeploying ReconcilerPhase = "Deploying"
	// ReconcilerPhaseRunning --
	ReconcilerPhaseRunning ReconcilerPhase = "Running"
	// ReconcilerPhaseError --
	ReconcilerPhaseError ReconcilerPhase = "Error"
	// ReconcilerPhaseDeleting --
	ReconcilerPhaseDeleting ReconcilerPhase = "Deleting"
)

func init() {
	SchemeBuilder.Register(&VirtualDatabase{}, &VirtualDatabaseList{})
}
