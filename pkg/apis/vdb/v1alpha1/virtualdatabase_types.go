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
	Name         string                      `json:"name,omitempty"`
	Image        ImageSpec                   `json:"image,omitempty"`
	Replicas     *int32                      `json:"replicas,omitempty"`
	Content      string                      `json:"content,omitempty"`
	Dependencies []string                    `json:"dependencies,omitempty"`
	Env          []corev1.EnvVar             `json:"env,omitempty"`
	Runtime      RuntimeType                 `json:"runtime,omitempty"`
	Resources    corev1.ResourceRequirements `json:"resources,omitempty"`
	Build        VirtualDatabaseBuildObject  `json:"build"` // S2I Build configuration
}

// VirtualDatabaseStatus defines the observed state of VirtualDatabase
// +k8s:openapi-gen=true
type VirtualDatabaseStatus struct {
	Phase          PublishingPhase `json:"phase,omitempty"`
	Digest         string          `json:"digest,omitempty"`
	Failure        string          `json:"failure,omitempty"`
	Image          string          `json:"image,omitempty"`
	RuntimeVersion string          `json:"runtimeVersion,omitempty"`
	Conditions     []Condition     `json:"conditions"`
	Route          string          `json:"route,omitempty"`
	Deployments    Deployments     `json:"deployments"`
}

// OpenShiftObject ...
type OpenShiftObject interface {
	metav1.Object
	runtime.Object
}

// VirtualDatabaseBuildObject Data to define how to build an application from source
// +k8s:openapi-gen=true
type VirtualDatabaseBuildObject struct {
	Incremental       *bool               `json:"incremental,omitempty"`
	Env               []corev1.EnvVar     `json:"env,omitempty"`
	GitSource         GitSource           `json:"gitSource,omitempty"`
	Webhooks          []WebhookSecret     `json:"webhooks,omitempty"`
	SourceFileChanges []SourceFileChanges `json:"sourceFileChanges,omitempty"`
}

// SourceFileChanges ...
// +k8s:openapi-gen=true
type SourceFileChanges struct {
	RelativePath string `json:"relativePath,omitempty"`
	Contents     string `json:"contents,omitempty"`
}

// GitSource Git coordinates to locate the source code to build
// +k8s:openapi-gen=true
type GitSource struct {
	URI        string `json:"uri,omitempty"`
	Reference  string `json:"reference,omitempty"`
	ContextDir string `json:"contextDir,omitempty"`
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
type RuntimeType string

const (
	// SpringbootRuntimeType - the virtualdatabase is being provisioned
	SpringbootRuntimeType RuntimeType = "springboot"
)

// Image - image details
// +k8s:openapi-gen=true
type Image struct {
	ImageStreamName      string `json:"imageStreamName,omitempty"`
	ImageStreamTag       string `json:"imageStreamTag,omitempty"`
	ImageStreamNamespace string `json:"imageStreamNamespace,omitempty"`
	ImageRegistry        string `json:"imageRegistry,omitempty"`
	ImageRepo            string `json:"imageRepo,omitempty"`
	BuilderImage         bool   `json:"builderImage,omitempty"`
}

// ConditionType - type of condition
type ConditionType string

const (
	// DeployedConditionType - the virtualdatabase is deployed
	DeployedConditionType ConditionType = "Deployed"
	// ProvisioningConditionType - the virtualdatabase is being provisioned
	ProvisioningConditionType ConditionType = "Provisioning"
	// FailedConditionType - the virtualdatabase is in a failed state
	FailedConditionType ConditionType = "Failed"
)

// ReasonType - type of reason
type ReasonType string

// Condition - The condition for the teiid-operator
// +k8s:openapi-gen=true
type Condition struct {
	Type               ConditionType          `json:"type"`
	Status             corev1.ConditionStatus `json:"status"`
	LastTransitionTime metav1.Time            `json:"lastTransitionTime,omitempty"`
	Reason             ReasonType             `json:"reason,omitempty"`
	Message            string                 `json:"message,omitempty"`
}

// Deployments ...
// +k8s:openapi-gen=true
type Deployments struct {
	// Deployments are ready to serve requests
	Ready []string `json:"ready,omitempty"`
	// Deployments are starting, may or may not succeed
	Starting []string `json:"starting,omitempty"`
	// Deployments are not starting, unclear what next step will be
	Stopped []string `json:"stopped,omitempty"`
	// Deployments failed
	Failed []string `json:"failed,omitempty"`
}

// ImageSpec ...
// +k8s:openapi-gen=true
type ImageSpec struct {
	BaseImage  string `json:"baseImage,omitempty"`
	DiskSize   string `json:"diskSize,omitempty"`
	MemorySize string `json:"memorySize,omitempty"`
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

	// DeploymentFailedReason - Unable to deploy the application
	DeploymentFailedReason ReasonType = "DeploymentFailed"
	// ConfigurationErrorReason - An invalid configuration caused an error
	ConfigurationErrorReason ReasonType = "ConfigurationError"
	// UnknownReason - Unable to determine the error
	UnknownReason ReasonType = "Unknown"
)

func init() {
	SchemeBuilder.Register(&VirtualDatabase{}, &VirtualDatabaseList{})
}
