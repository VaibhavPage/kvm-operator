package deployment

import (
	"fmt"

	"github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	apiv1 "k8s.io/api/core/v1"
	extensionsv1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/giantswarm/kvm-operator/service/controller/v4/key"
)

func newWorkerDeployments(customObject v1alpha1.KVMConfig) ([]*extensionsv1.Deployment, error) {
	var deployments []*extensionsv1.Deployment

	privileged := true
	replicas := int32(1)

	for i, workerNode := range customObject.Spec.Cluster.Workers {
		capabilities := customObject.Spec.KVM.Workers[i]

		cpuQuantity, err := key.CPUQuantity(capabilities)
		if err != nil {
			return nil, microerror.Maskf(err, "creating CPU quantity")
		}

		memoryQuantity, err := key.MemoryQuantity(capabilities)
		if err != nil {
			return nil, microerror.Maskf(err, "creating memory quantity")
		}

		deployment := &extensionsv1.Deployment{
			TypeMeta: apismetav1.TypeMeta{
				Kind:       "deployment",
				APIVersion: "extensions/v1beta",
			},
			ObjectMeta: apismetav1.ObjectMeta{
				Name: key.DeploymentName(key.WorkerID, workerNode.ID),
				Annotations: map[string]string{
					key.VersionBundleVersionAnnotation: key.VersionBundleVersion(customObject),
				},
				Labels: map[string]string{
					"app":      key.WorkerID,
					"cluster":  key.ClusterID(customObject),
					"customer": key.ClusterCustomer(customObject),
					"node":     workerNode.ID,
				},
			},
			Spec: extensionsv1.DeploymentSpec{
				Strategy: extensionsv1.DeploymentStrategy{
					Type: extensionsv1.RecreateDeploymentStrategyType,
				},
				Replicas: &replicas,
				Template: apiv1.PodTemplateSpec{
					ObjectMeta: apismetav1.ObjectMeta{
						Name: key.WorkerID,
						Labels: map[string]string{
							"cluster":  key.ClusterID(customObject),
							"customer": key.ClusterCustomer(customObject),
							"app":      key.WorkerID,
							"node":     workerNode.ID,
						},
						Annotations: map[string]string{
							key.AnnotationIp:      "",
							key.AnnotationService: key.WorkerID,
						},
					},
					Spec: apiv1.PodSpec{
						Affinity:    newWorkerPodAfinity(customObject),
						HostNetwork: true,
						NodeSelector: map[string]string{
							"role": key.WorkerID,
						},
						ServiceAccountName: key.ServiceAccountName(customObject),
						Volumes: []apiv1.Volume{
							{
								Name: "cloud-config",
								VolumeSource: apiv1.VolumeSource{
									ConfigMap: &apiv1.ConfigMapVolumeSource{
										LocalObjectReference: apiv1.LocalObjectReference{
											Name: key.ConfigMapName(customObject, workerNode, key.WorkerID),
										},
									},
								},
							},
							{
								Name: "images",
								VolumeSource: apiv1.VolumeSource{
									HostPath: &apiv1.HostPathVolumeSource{
										Path: key.CoreosImageDir,
									},
								},
							},
							{
								Name: "rootfs",
								VolumeSource: apiv1.VolumeSource{
									EmptyDir: &apiv1.EmptyDirVolumeSource{},
								},
							},
							{
								Name: "flannel",
								VolumeSource: apiv1.VolumeSource{
									HostPath: &apiv1.HostPathVolumeSource{
										Path: key.FlannelEnvPathPrefix,
									},
								},
							},
						},
						Containers: []apiv1.Container{
							{
								Name:            "k8s-endpoint-updater",
								Image:           key.K8SEndpointUpdaterDocker,
								ImagePullPolicy: apiv1.PullIfNotPresent,
								Command: []string{
									"/bin/sh",
									"-c",
									"/opt/k8s-endpoint-updater update --provider.bridge.name=" + key.NetworkBridgeName(customObject) +
										" --service.kubernetes.cluster.namespace=" + key.ClusterNamespace(customObject) +
										" --service.kubernetes.cluster.service=" + key.WorkerID +
										" --service.kubernetes.inCluster=true" +
										" --service.kubernetes.pod.name=${POD_NAME}",
								},
								SecurityContext: &apiv1.SecurityContext{
									Privileged: &privileged,
								},
								Env: []apiv1.EnvVar{
									{
										Name: "POD_NAME",
										ValueFrom: &apiv1.EnvVarSource{
											FieldRef: &apiv1.ObjectFieldSelector{
												APIVersion: "v1",
												FieldPath:  "metadata.name",
											},
										},
									},
								},
							},
							{
								Name:            "k8s-kvm",
								Image:           key.K8SKVMDockerImage,
								ImagePullPolicy: apiv1.PullIfNotPresent,
								SecurityContext: &apiv1.SecurityContext{
									Privileged: &privileged,
								},
								Args: []string{
									key.WorkerID,
								},
								Env: []apiv1.EnvVar{
									{
										Name:  "CORES",
										Value: fmt.Sprintf("%d", capabilities.CPUs),
									},
									{
										Name:  "COREOS_VERSION",
										Value: key.CoreosVersion,
									},
									{
										Name:  "DISK",
										Value: fmt.Sprintf("%.0fG", capabilities.Disk),
									},
									{
										Name: "HOSTNAME",
										ValueFrom: &apiv1.EnvVarSource{
											FieldRef: &apiv1.ObjectFieldSelector{
												APIVersion: "v1",
												FieldPath:  "metadata.name",
											},
										},
									},
									{
										Name:  "NETWORK_BRIDGE_NAME",
										Value: key.NetworkBridgeName(customObject),
									},
									{
										Name:  "NETWORK_TAP_NAME",
										Value: key.NetworkTapName(customObject),
									},
									{
										Name: "MEMORY",
										// TODO provide memory like disk as float64 and format here.
										Value: capabilities.Memory,
									},
									{
										Name:  "ROLE",
										Value: key.WorkerID,
									},
									{
										Name:  "CLOUD_CONFIG_PATH",
										Value: "/cloudconfig/user_data",
									},
								},
								Lifecycle: &apiv1.Lifecycle{
									PreStop: &apiv1.Handler{
										Exec: &apiv1.ExecAction{
											Command: []string{"/qemu-shutdown"},
										},
									},
								},
								LivenessProbe: &apiv1.Probe{
									InitialDelaySeconds: key.InitialDelaySeconds,
									TimeoutSeconds:      key.TimeoutSeconds,
									PeriodSeconds:       key.PeriodSeconds,
									FailureThreshold:    key.FailureThreshold,
									SuccessThreshold:    key.SuccessThreshold,
									Handler: apiv1.Handler{
										HTTPGet: &apiv1.HTTPGetAction{
											Path: key.HealthEndpoint,
											Port: intstr.IntOrString{IntVal: key.LivenessPort(customObject)},
											Host: key.ProbeHost,
										},
									},
								},
								Resources: apiv1.ResourceRequirements{
									Requests: map[apiv1.ResourceName]resource.Quantity{
										apiv1.ResourceCPU:    cpuQuantity,
										apiv1.ResourceMemory: memoryQuantity,
									},
									Limits: map[apiv1.ResourceName]resource.Quantity{
										apiv1.ResourceCPU:    cpuQuantity,
										apiv1.ResourceMemory: memoryQuantity,
									},
								},
								VolumeMounts: []apiv1.VolumeMount{
									{
										Name:      "cloud-config",
										MountPath: "/cloudconfig/",
									},
									{
										Name:      "images",
										MountPath: "/usr/code/images/",
									},
									{
										Name:      "rootfs",
										MountPath: "/usr/code/rootfs/",
									},
								},
							},
							{
								Name:            "k8s-kvm-health",
								Image:           key.K8SKVMHealthDocker,
								ImagePullPolicy: apiv1.PullAlways,
								Env: []apiv1.EnvVar{
									{
										Name:  "LISTEN_ADDRESS",
										Value: key.HealthListenAddress(customObject),
									},
									{
										Name:  "NETWORK_ENV_FILE_PATH",
										Value: key.NetworkEnvFilePath(customObject),
									},
								},
								SecurityContext: &apiv1.SecurityContext{
									Privileged: &privileged,
								},
								VolumeMounts: []apiv1.VolumeMount{
									{
										Name:      "flannel",
										MountPath: key.FlannelEnvPathPrefix,
									},
								},
							},
						},
					},
				},
			},
		}

		deployments = append(deployments, deployment)
	}

	return deployments, nil
}
