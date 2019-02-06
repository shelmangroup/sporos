package v1alpha1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ControlplanePhase string

const (
	ControlplanePhaseInitial ControlplanePhase = ""
	ControlplanePhaseRunning                   = "Running"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type SporosList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Sporos `json:"items"`
}

type PodPolicy struct {
	// Resources is the resource requirements for the containers.
	Resources v1.ResourceRequirements `json:"resources,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Sporos struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              SporosSpec   `json:"spec"`
	Status            SporosStatus `json:"status,omitempty"`
}

type SporosSpec struct {
	PodCIDR     string `json:"podCIDR"`
	ServiceCIDR string `json:"serviceCIDR"`
	// Base image to use for a k8s deployment.
	BaseImage string `json:"baseImage"`

	// Version of k8s to be deployed.
	Version string `json:"version"`

	// Pod defines the policy for pods owned by sporos operator.
	// This field cannot be updated once the CR is created.
	Pod *PodPolicy `json:"pod,omitempty"`
}

type SporosStatus struct {
	Phase       ControlplanePhase `json:"phase"`
	Nodes       []string          `json:"nodes"`
	ApiServerIP string            `json:"apiserverip"`
}

func init() {
	SchemeBuilder.Register(&Sporos{}, &SporosList{})
}
