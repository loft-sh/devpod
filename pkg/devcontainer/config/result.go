package config

type Result struct {
	ContainerDetails    *ContainerDetails
	MergedConfig        *MergedDevContainerConfig
	SubstitutionContext *SubstitutionContext
}
