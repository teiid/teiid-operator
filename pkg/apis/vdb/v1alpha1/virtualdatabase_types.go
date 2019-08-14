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
	Replicas        *int32                      `json:"replicas,omitempty"`
	ExposeVia3Scale bool                        `json:"exposeVia3scale,omitempty"`
	Env             []corev1.EnvVar             `json:"env,omitempty"`
	Runtime         RuntimeType                 `json:"runtime,omitempty"`
	Resources       corev1.ResourceRequirements `json:"resources,omitempty"`
	Build           VirtualDatabaseBuildObject  `json:"build"` // S2I Build configuration
}

// VirtualDatabaseStatus defines the observed state of VirtualDatabase
// +k8s:openapi-gen=true
type VirtualDatabaseStatus struct {
	Phase   ReconcilerPhase `json:"phase,omitempty"`
	Digest  string          `json:"digest,omitempty"`
	Failure string          `json:"failure,omitempty"`
	Route   string          `json:"route,omitempty"`
}

// OpenShiftObject ...
type OpenShiftObject interface {
	metav1.Object
	runtime.Object
}

// VirtualDatabaseBuildObject Data to define how to build an application from source
// +k8s:openapi-gen=true
type VirtualDatabaseBuildObject struct {
	Incremental *bool           `json:"incremental,omitempty"`
	Env         []corev1.EnvVar `json:"env,omitempty"`
	Git         Git             `json:"git,omitempty"`
	Source      Source          `json:"source,omitempty"`
	Webhooks    []WebhookSecret `json:"webhooks,omitempty"`
}

// Git coordinates to locate the source code to build
// +k8s:openapi-gen=true
type Git struct {
	URI        string `json:"uri,omitempty"`
	Reference  string `json:"reference,omitempty"`
	ContextDir string `json:"contextDir,omitempty"`
}

// Source Git coordinates to locate the source code to build
// +k8s:openapi-gen=true
type Source struct {
	DDL          string   `json:"ddl,omitempty"`
	OpenAPI      string   `json:"openapi,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
}

// WebhookType literal type to distinguish between different types of Webhooks
type WebhookType string

const (
	// GitHubWebhook GitHub webhook
	GitHubWebhook WebhookType = "GitHub"
	// GenericWebhook Generic webhook
	GenericWebhook WebhookType = "Generic"
)

// WebhookSecret Secret to use for a given webhook
// +k8s:openapi-gen=true
type WebhookSecret struct {
	Type   WebhookType `json:"type,omitempty"`
	Secret string      `json:"secret,omitempty"`
}

// RuntimeType - type of condition
type RuntimeType struct {
	Type    string `json:"type,omitempty"`
	Version string `json:"version,omitempty"`
}

// Image - image details
// +k8s:openapi-gen=true
type Image struct {
	ImageStreamName      string `json:"imageStreamName,omitempty"`
	ImageStreamTag       string `json:"imageStreamTag,omitempty"`
	ImageStreamNamespace string `json:"imageStreamNamespace,omitempty"`
	ImageRegistry        string `json:"imageRegistry,omitempty"`
	ImageRepository      string `json:"imageRepository,omitempty"`
	BuilderImage         bool   `json:"builderImage,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualDatabase is the Schema for the virtualdatabases API
// +k8s:openapi-gen=true
type VirtualDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualDatabaseSpec   `json:"spec,omitempty"`
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

	// ReconcilerPhaseCodeGeneration Code generation
	ReconcilerPhaseCodeGeneration ReconcilerPhase = "Code Generation"
	// ReconcilerPhaseCodeGenerationCompleted Code generation completed
	ReconcilerPhaseCodeGenerationCompleted ReconcilerPhase = "Code Generation Completed"
	// ReconcilerPhaseBuildImageSubmitted --
	ReconcilerPhaseBuildImageSubmitted ReconcilerPhase = "Build Image Submitted"
	// ReconcilerPhaseBuildImageRunning --
	ReconcilerPhaseBuildImageRunning ReconcilerPhase = "Build Image Running"
	// ReconcilerPhaseBuildImageComplete --
	ReconcilerPhaseBuildImageComplete ReconcilerPhase = "Build Image Completed"
	// ReconcilerPhaseDeploying --
	ReconcilerPhaseDeploying ReconcilerPhase = "Deploying"
	// ReconcilerPhaseRunning --
	ReconcilerPhaseRunning ReconcilerPhase = "Running"
	// ReconcilerPhaseError --
	ReconcilerPhaseError ReconcilerPhase = "Error"
	// ReconcilerPhaseBuildFailureRecovery --
	ReconcilerPhaseBuildFailureRecovery ReconcilerPhase = "Building Failure Recovery"
	// ReconcilerPhaseDeleting --
	ReconcilerPhaseDeleting ReconcilerPhase = "Deleting"
)

func init() {
	SchemeBuilder.Register(&VirtualDatabase{}, &VirtualDatabaseList{})
}
