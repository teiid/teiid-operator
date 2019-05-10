package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VirtualDatabaseSpec defines the desired state of VirtualDatabase
// +k8s:openapi-gen=true
type VirtualDatabaseSpec struct {
	ImageSpec     ImageSpec       `json:"image,omitempty"`
	Replicas      *int32          `json:"replicas,omitempty"`
	Content       string          `json:"content,omitempty"`
	Dependencies  []string        `json:"dependencies,omitempty"`
	Configuration []corev1.EnvVar `json:"env,omitempty"`
}

// VirtualDatabaseStatus defines the observed state of VirtualDatabase
// +k8s:openapi-gen=true
type VirtualDatabaseStatus struct {
	Phase          PublishingPhase `json:"phase,omitempty"`
	Digest         string          `json:"digest,omitempty"`
	Failure        string          `json:"failure,omitempty"`
	Image          string          `json:"image,omitempty"`
	RuntimeVersion string          `json:"runtimeVersion,omitempty"`
}

// VirtualDatabaseStatus defines the observed state of VirtualDatabase
// +k8s:openapi-gen=true
type ImageSpec struct {
	BaseImage  string `json:"base,omitempty"`
	DiskSize   string `json:"disk-size,omitempty"`
	MemorySize string `json:"memory-size,omitempty"`
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

// IntegrationPhase --
type PublishingPhase string

const (
	// IntegrationKind --
	VirtualDatabaseKind string = "VirtualDatabase"

	// IntegrationPhaseInitial --
	PublishingPhaseInitial PublishingPhase = ""

	// Code generation
	PublishingPhaseCodeGeneration PublishingPhase = "Code Generation"
	// Code generation completed
	PublishingPhaseCodeGenerationCompleted PublishingPhase = "Code Generation Completed"
	// IntegrationPhaseBuildImageSubmitted --
	PublishingPhaseBuildImageSubmitted PublishingPhase = "Build Image Submitted"
	// IntegrationPhaseBuildImageRunning --
	PublishingPhaseBuildImageRunning PublishingPhase = "Build Image Running"
	// IntegrationPhaseBuildImageRunning --
	PublishingPhaseBuildImageComplete PublishingPhase = "Build Image Completed"
	// IntegrationPhaseDeploying --
	PublishingPhaseDeploying PublishingPhase = "Deploying"
	// IntegrationPhaseRunning --
	PublishingPhaseRunning PublishingPhase = "Running"
	// IntegrationPhaseError --
	PublishingPhaseError PublishingPhase = "Error"
	// IntegrationPhaseBuildFailureRecovery --
	PublishingPhaseBuildFailureRecovery PublishingPhase = "Building Failure Recovery"
	// IntegrationPhaseDeleting --
	PublishingPhaseDeleting PublishingPhase = "Deleting"
)

func init() {
	SchemeBuilder.Register(&VirtualDatabase{}, &VirtualDatabaseList{})
}
