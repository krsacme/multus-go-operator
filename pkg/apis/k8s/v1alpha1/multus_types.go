package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	NetAttachDefResourceKind   = "NetworkAttachDefinition"
	NetAttachDefResourcePlural = "NetworkAttachDefinitions"
	GroupName                  = "k8s.cni.cncf.io"
)

// NetworkType defines the supported networks for multus delegates
type NetworkType string
type ExtensionType string

const (
	// NetworkTypeFlannel defines the flannel network delegate
	NetworkTypeFlannel NetworkType = "flannel"

	ExtensionTypeSriov ExtensionType = "sriov"
)

// SR-IOV Resource definition
type SriovResource struct {
	Name        string   `json:"name"`
	RootDevices []string `json:"rootDevices"`
	Type        string   `json:"type"`
}

// TODO: How to generalize for multiple type of extension?
// SR-IOV extension
type Extension struct {
	Resources []SriovResource `json:"resources"`
}

// NetworkDelegates defines the network delegates to be used with multus
type NetworkDelegates struct {
	// Type of the delegate
	Type NetworkType `json:"type"`
	// Name of the network delegate
	Name string `json:"name"`
}

// MultusSpec defines the desired state of Multus
type MultusSpec struct {
	// Multus CNI Image to run as DaemonSet on all the nodes
	Image string `json:"image"`
	// Multus CNI Release version as image tag
	Release string `json:"release"`
	// List of delegates to be used bu Multus
	// TODO: Its better to use 'clusterNetwork' and 'defaultNetworks'
	Delegates []NetworkDelegates `json:"delegates"`
	// Map of Extensions
	Extensions map[ExtensionType]Extension `json:"extensions"`
}

// MultusStatus defines the observed state of Multus
type MultusStatus struct {
	// TODO:
	Message string `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Multus is the Schema for the multus API
// +k8s:openapi-gen=true
type Multus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MultusSpec   `json:"spec,omitempty"`
	Status MultusStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MultusList contains a list of Multus
type MultusList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Multus `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Multus{}, &MultusList{})
}
