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
	// Expose route via 3scale
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Expose Via 3scale"
	ExposeVia3Scale bool `json:"exposeVia3scale,omitempty"`
	// Environment properties required for deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Properties"
	Env []corev1.EnvVar `json:"env,omitempty"`
	// Runtime engine type (ex: spring boot)
	Runtime RuntimeType `json:"runtime,omitempty"`
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
}

// VirtualDatabaseStatus defines the observed state of VirtualDatabase
// +k8s:openapi-gen=true
type VirtualDatabaseStatus struct {
	Phase ReconcilerPhase `json:"phase,omitempty"`
	// Digest value of the vdb
	Digest string `json:"digest,omitempty"`
	// Failure message if deployment ended in failure
	Failure string `json:"failure,omitempty"`
	// Route information that is exposed for clients
	Route string `json:"route,omitempty"`
	// Deployed vdb version.
	Version string `json:"version,omitempty"`
}

// OpenShiftObject ...
type OpenShiftObject interface {
	metav1.Object
	runtime.Object
}

// VirtualDatabaseBuildObject Data to define how to build an application from source
// +k8s:openapi-gen=true
type VirtualDatabaseBuildObject struct {
	// Should incremental build is being used
	Incremental *bool `json:"incremental,omitempty"`
	// Environment properties set build purpose
	Env []corev1.EnvVar `json:"env,omitempty"`
	// VDB Source details
	Source Source `json:"source,omitempty"`
	// source to image details
	S2i S2i `json:"s2i,omitempty"`
}

// Source VDB coordinates to locate the source code to build
// +k8s:openapi-gen=true
type Source struct {
	// Deployed vdb version. For embedded DDL version this will be implicitly provided when ignored, for maven based vdb the maven version is always
	Version string `json:"version,omitempty"`
	// DDL based VDB
	DDL string `json:"ddl,omitempty"`
	// A VDB defined in GAV format
	Maven string `json:"maven,omitempty"`
	// Open API contract that is exposed by the VDB
	OpenAPI string `json:"openapi,omitempty"`
	// List of maven dependencies for the build in GAV format
	Dependencies []string `json:"dependencies,omitempty"`
	// Custom maven repositories that need to be used for the S2I build
	MavenRepositories map[string]string `json:"mavenRepositories,omitempty"`
	// Custom Maven settings.xml file to go with build in a configmap or secret
	MavenSettings ValueSource `json:"mavenSettings,omitempty"`
}

// ValueSource --
type ValueSource struct {
	// Selects a key of a ConfigMap.
	ConfigMapKeyRef *corev1.ConfigMapKeySelector `json:"configMapKeyRef,omitempty"`
	// Selects a key of a secret.
	SecretKeyRef *corev1.SecretKeySelector `json:"secretKeyRef,omitempty"`
}

// S2i Git coordinates to locate the s2i image
// +k8s:openapi-gen=true
type S2i struct {
	// S2I image registry
	Registry string `json:"registry,omitempty"`
	// S2I image prefix
	ImagePrefix string `json:"imagePrefix,omitempty"`
	// S2I image name
	ImageName string `json:"imageName,omitempty"`
	// S2I image tag
	Tag string `json:"tag,omitempty"`
}

// RuntimeType - type of condition
type RuntimeType struct {
	Type    string `json:"type,omitempty"`
	Version string `json:"version,omitempty"`
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
