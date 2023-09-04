package framework

// PodList is a list of Pods.
type PodList struct {
	Items []Pod `json:"items"`
}

type Pod struct {
	Spec PodSpec `json:"spec,omitempty"`
}

type PodSpec struct {
	Containers []PodContainer `json:"containers,omitempty"`
}

type PodContainer struct {
	Image string `json:"image,omitempty"`
}
