//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Code generated by defaulter-gen. DO NOT EDIT.

package management

import (
	v1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// RegisterDefaults adds defaulters functions to the given scheme.
// Public to allow building arbitrary schemes.
// All generated defaulters are covering - they call all nested defaulters.
func RegisterDefaults(scheme *runtime.Scheme) error {
	scheme.AddTypeDefaultingFunc(&ProjectClusters{}, func(obj interface{}) { SetObjectDefaults_ProjectClusters(obj.(*ProjectClusters)) })
	scheme.AddTypeDefaultingFunc(&ProjectClustersList{}, func(obj interface{}) { SetObjectDefaults_ProjectClustersList(obj.(*ProjectClustersList)) })
	scheme.AddTypeDefaultingFunc(&ProjectRunners{}, func(obj interface{}) { SetObjectDefaults_ProjectRunners(obj.(*ProjectRunners)) })
	scheme.AddTypeDefaultingFunc(&ProjectRunnersList{}, func(obj interface{}) { SetObjectDefaults_ProjectRunnersList(obj.(*ProjectRunnersList)) })
	scheme.AddTypeDefaultingFunc(&Runner{}, func(obj interface{}) { SetObjectDefaults_Runner(obj.(*Runner)) })
	scheme.AddTypeDefaultingFunc(&RunnerList{}, func(obj interface{}) { SetObjectDefaults_RunnerList(obj.(*RunnerList)) })
	return nil
}

func SetObjectDefaults_ProjectClusters(in *ProjectClusters) {
	for i := range in.Runners {
		a := &in.Runners[i]
		SetObjectDefaults_Runner(a)
	}
}

func SetObjectDefaults_ProjectClustersList(in *ProjectClustersList) {
	for i := range in.Items {
		a := &in.Items[i]
		SetObjectDefaults_ProjectClusters(a)
	}
}

func SetObjectDefaults_ProjectRunners(in *ProjectRunners) {
	for i := range in.Runners {
		a := &in.Runners[i]
		SetObjectDefaults_Runner(a)
	}
}

func SetObjectDefaults_ProjectRunnersList(in *ProjectRunnersList) {
	for i := range in.Items {
		a := &in.Items[i]
		SetObjectDefaults_ProjectRunners(a)
	}
}

func SetObjectDefaults_Runner(in *Runner) {
	if in.Spec.RunnerSpec.ClusterRef != nil {
		if in.Spec.RunnerSpec.ClusterRef.PodTemplate != nil {
			for i := range in.Spec.RunnerSpec.ClusterRef.PodTemplate.Spec.Volumes {
				a := &in.Spec.RunnerSpec.ClusterRef.PodTemplate.Spec.Volumes[i]
				if a.VolumeSource.ISCSI != nil {
					if a.VolumeSource.ISCSI.ISCSIInterface == "" {
						a.VolumeSource.ISCSI.ISCSIInterface = "default"
					}
				}
				if a.VolumeSource.RBD != nil {
					if a.VolumeSource.RBD.RBDPool == "" {
						a.VolumeSource.RBD.RBDPool = "rbd"
					}
					if a.VolumeSource.RBD.RadosUser == "" {
						a.VolumeSource.RBD.RadosUser = "admin"
					}
					if a.VolumeSource.RBD.Keyring == "" {
						a.VolumeSource.RBD.Keyring = "/etc/ceph/keyring"
					}
				}
				if a.VolumeSource.AzureDisk != nil {
					if a.VolumeSource.AzureDisk.CachingMode == nil {
						ptrVar1 := v1.AzureDataDiskCachingMode(v1.AzureDataDiskCachingReadWrite)
						a.VolumeSource.AzureDisk.CachingMode = &ptrVar1
					}
					if a.VolumeSource.AzureDisk.FSType == nil {
						var ptrVar1 string = "ext4"
						a.VolumeSource.AzureDisk.FSType = &ptrVar1
					}
					if a.VolumeSource.AzureDisk.ReadOnly == nil {
						var ptrVar1 bool = false
						a.VolumeSource.AzureDisk.ReadOnly = &ptrVar1
					}
					if a.VolumeSource.AzureDisk.Kind == nil {
						ptrVar1 := v1.AzureDataDiskKind(v1.AzureSharedBlobDisk)
						a.VolumeSource.AzureDisk.Kind = &ptrVar1
					}
				}
				if a.VolumeSource.ScaleIO != nil {
					if a.VolumeSource.ScaleIO.StorageMode == "" {
						a.VolumeSource.ScaleIO.StorageMode = "ThinProvisioned"
					}
					if a.VolumeSource.ScaleIO.FSType == "" {
						a.VolumeSource.ScaleIO.FSType = "xfs"
					}
				}
			}
			for i := range in.Spec.RunnerSpec.ClusterRef.PodTemplate.Spec.InitContainers {
				a := &in.Spec.RunnerSpec.ClusterRef.PodTemplate.Spec.InitContainers[i]
				for j := range a.Ports {
					b := &a.Ports[j]
					if b.Protocol == "" {
						b.Protocol = "TCP"
					}
				}
				if a.LivenessProbe != nil {
					if a.LivenessProbe.ProbeHandler.GRPC != nil {
						if a.LivenessProbe.ProbeHandler.GRPC.Service == nil {
							var ptrVar1 string = ""
							a.LivenessProbe.ProbeHandler.GRPC.Service = &ptrVar1
						}
					}
				}
				if a.ReadinessProbe != nil {
					if a.ReadinessProbe.ProbeHandler.GRPC != nil {
						if a.ReadinessProbe.ProbeHandler.GRPC.Service == nil {
							var ptrVar1 string = ""
							a.ReadinessProbe.ProbeHandler.GRPC.Service = &ptrVar1
						}
					}
				}
				if a.StartupProbe != nil {
					if a.StartupProbe.ProbeHandler.GRPC != nil {
						if a.StartupProbe.ProbeHandler.GRPC.Service == nil {
							var ptrVar1 string = ""
							a.StartupProbe.ProbeHandler.GRPC.Service = &ptrVar1
						}
					}
				}
			}
		}
	}
}

func SetObjectDefaults_RunnerList(in *RunnerList) {
	for i := range in.Items {
		a := &in.Items[i]
		SetObjectDefaults_Runner(a)
	}
}
