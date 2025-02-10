package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NetworkPeer hols the information of network peers
// +k8s:openapi-gen=true
type NetworkPeer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,inline"`

	Spec   NetworkPeerSpec   `json:"spec,omitempty"`
	Status NetworkPeerStatus `json:"status,omitempty"`
}

type NetworkPeerSpec struct {
	// DiscoKey is a key used for DERP discovery
	DiscoKey string `json:"discoKey,omitempty"`
	// MachineKey is used to identify a network peer
	MachineKey string `json:"machineKey,omitempty"`
	// NodeKey is used to identify a session
	NodeKey string `json:"nodeKey,omitempty"`
	// Addresses is a list of IP addresses of this Node directly.
	Addresses []string `json:"addresses,omitempty"`
	// AllowedIPs is a range of IP addresses to route to this node.
	AllowedIPs []string `json:"allowedIPs,omitempty"`
	// Endpoints is a list of IP+port (public via STUN, and local LANs) where
	// this node can be reached.
	Endpoints []string `json:"endpoints,omitempty"`
}

type NetworkPeerStatus struct {
	// LastSeen is when the network peer was last online. It is not updated when
	// Online is true.
	LastSeen string `json:"lastSeen,omitempty"`
	// HomeDerpRegion is the currently preferred DERP region by the network peer
	HomeDerpRegion int `json:"homeDerpRegion,omitempty"`
	// Online is whether the node is currently connected to the coordination
	// server.
	Online bool `json:"online,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NetworkPeerList contains a list of NetworkPeers
type NetworkPeerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkPeer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NetworkPeer{}, &NetworkPeerList{})
}
