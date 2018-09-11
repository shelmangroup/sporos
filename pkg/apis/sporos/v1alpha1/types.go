package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterPhase string

const (
	ClusterPhaseInitial ClusterPhase = ""
	ClusterPhaseRunning              = "Running"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type SporosList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Sporos `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Sporos struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              SporosSpec   `json:"spec"`
	Status            SporosStatus `json:"status,omitempty"`
}

type SporosSpec struct {
	ApiServerUrl string `json:"apiServerURL"`
	ApiServerIP  string `json:"apiServerIP"`
	PodCIDR      string `json:"podCIDR"`
	ServiceCIDR  string `json:"serviceCIDR"`
}

type SporosStatus struct {
	Phase ClusterPhase `json:"phase"`
	Nodes []string     `json:"nodes"`
}
