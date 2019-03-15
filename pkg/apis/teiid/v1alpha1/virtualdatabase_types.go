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
	Phase          IntegrationPhase `json:"phase,omitempty"`
	Failure        string           `json:"failure,omitempty"`
	Image          string           `json:"image,omitempty"`
	RuntimeVersion string           `json:"runtimeVersion,omitempty"`
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
type IntegrationPhase string

const (
	// IntegrationKind --
	IntegrationKind string = "Integration"

	// IntegrationPhaseInitial --
	IntegrationPhaseInitial IntegrationPhase = ""
	// IntegrationPhaseWaitingForPlatform --
	IntegrationPhaseWaitingForPlatform IntegrationPhase = "Waiting For Platform"
	// IntegrationPhaseBuildingContext --
	IntegrationPhaseBuildingContext IntegrationPhase = "Building Context"
	// IntegrationPhaseResolvingContext --
	IntegrationPhaseResolvingContext IntegrationPhase = "Resolving Context"
	// IntegrationPhaseBuildImageSubmitted --
	IntegrationPhaseBuildImageSubmitted IntegrationPhase = "Build Image Submitted"
	// IntegrationPhaseBuildImageRunning --
	IntegrationPhaseBuildImageRunning IntegrationPhase = "Build Image Running"
	// IntegrationPhaseDeploying --
	IntegrationPhaseDeploying IntegrationPhase = "Deploying"
	// IntegrationPhaseRunning --
	IntegrationPhaseRunning IntegrationPhase = "Running"
	// IntegrationPhaseError --
	IntegrationPhaseError IntegrationPhase = "Error"
	// IntegrationPhaseBuildFailureRecovery --
	IntegrationPhaseBuildFailureRecovery IntegrationPhase = "Building Failure Recovery"
	// IntegrationPhaseDeleting --
	IntegrationPhaseDeleting IntegrationPhase = "Deleting"
)

func init() {
	SchemeBuilder.Register(&VirtualDatabase{}, &VirtualDatabaseList{})
}
