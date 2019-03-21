package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VirtualDatabaseSpec defines the desired state of VirtualDatabase
// +k8s:openapi-gen=true
type VirtualDatabaseSpec struct {
	Replicas *int32 `json:"replicas,omitempty"`
}

// VirtualDatabaseStatus defines the observed state of VirtualDatabase
// +k8s:openapi-gen=true
type VirtualDatabaseStatus struct {
	Phase          PublishingPhase `json:"phase,omitempty"`
	Failure        string          `json:"failure,omitempty"`
	Image          string          `json:"image,omitempty"`
	RuntimeVersion string          `json:"runtimeVersion,omitempty"`
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
	IntegrationKind string = "Integration"

	// IntegrationPhaseInitial --
	PublishingPhaseInitial PublishingPhase = ""
	// IntegrationPhaseWaitingForPlatform --
	PublishingPhaseWaitingForPlatform PublishingPhase = "Waiting For Platform"
	// IntegrationPhaseBuildingContext --
	PublishingPhaseBuildingContext PublishingPhase = "Building Context"
	// IntegrationPhaseResolvingContext --
	PublishingPhaseResolvingContext PublishingPhase = "Resolving Context"
	// IntegrationPhaseBuildImageSubmitted --
	PublishingPhaseBuildImageSubmitted PublishingPhase = "Build Image Submitted"
	// IntegrationPhaseBuildImageRunning --
	PublishingPhaseBuildImageRunning PublishingPhase = "Build Image Running"
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
