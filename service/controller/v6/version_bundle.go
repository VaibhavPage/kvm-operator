package v6

import (
	"github.com/giantswarm/versionbundle"
)

func VersionBundle() versionbundle.Bundle {
	return versionbundle.Bundle{
		Changelogs: []versionbundle.Changelog{
			{
				Component:   "kvm-node-controller",
				Description: "Updated KVM node controller with pod status bugfix.",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "Calico",
				Description: "Updated to 3.0.2.",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "kubelet",
				Description: "Tune kubelet flags for protecting key units (kubelet and container runtime) from workload overloads.",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "etcd",
				Description: "Updated to 3.3.1.",
				Kind:        versionbundle.KindChanged,
			},
			{
				Component:   "qemu",
				Description: "Fixed formula for calculating qemu memory overhead.",
				Kind:        versionbundle.KindFixed,
			},
			{
				Component:   "monitoring",
				Description: "Added configuration for monitoring endpoint IP addresses.",
				Kind:        versionbundle.KindAdded,
			},
		},
		Components: []versionbundle.Component{
			{
				Name:    "calico",
				Version: "3.0.2",
			},
			{
				Name:    "containerlinux",
				Version: "1576.5.0",
			},
			{
				Name:    "docker",
				Version: "17.09.0",
			},
			{
				Name:    "etcd",
				Version: "3.3.1",
			},
			{
				Name:    "coredns",
				Version: "1.0.5",
			},
			{
				Name:    "kubernetes",
				Version: "1.9.2",
			},
			{
				Name:    "nginx-ingress-controller",
				Version: "0.10.2",
			},
		},
		Name:    "kvm-operator",
		Version: "2.0.1",
	}
}
