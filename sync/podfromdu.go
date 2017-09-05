package sync

import (
	"fmt"
	"path/filepath"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"

	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/netes-agent/labels"
	"github.com/rancher/netes-agent/utils"
)

const (
	rancherPauseContainerName = "rancher-pause"
	hostNetworkingKind        = "dockerHost"
)

var (
	rancherPauseContainer = v1.Container{
		Name: rancherPauseContainerName,
		// TODO: figure out where to read this so it's not hard-coded
		Image: "gcr.io/google_containers/pause-amd64:3.0",
	}
)

// TODO: move this
func Primary(d client.DeploymentSyncRequest) client.Container {
	if len(d.Containers) == 1 {
		return d.Containers[0]
	}
	for _, container := range d.Containers {
		value, ok := container.Labels[labels.ServiceLaunchConfig]
		if ok && value == labels.ServicePrimaryLaunchConfig {
			return container
		}
	}
	return client.Container{}
}

func podFromDeploymentUnit(deploymentUnit client.DeploymentSyncRequest) v1.Pod {
	containers := []v1.Container{rancherPauseContainer}
	for _, container := range deploymentUnit.Containers {
		containers = append(containers, getContainer(container))
	}

	podSpec := getPodSpec(deploymentUnit)
	podSpec.Containers = containers

	return v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: getPodName(deploymentUnit),
			Namespace: deploymentUnit.Namespace,
			Labels: getLabels(deploymentUnit),
			Annotations: getAnnotations(deploymentUnit),
		},
		Spec: podSpec,
	}
}

func getContainer(container client.Container) v1.Container {
	return v1.Container{
		Name:            transformContainerName(container.Name),
		Image:           container.Image,
		Command:         container.EntryPoint,
		Args:            container.Command,
		TTY:             container.Tty,
		Stdin:           container.StdinOpen,
		WorkingDir:      container.WorkingDir,
		Env:             getEnvironment(container),
		SecurityContext: getSecurityContext(container),
		VolumeMounts:    getVolumeMounts(container),
	}
}

func getPodName(deploymentUnit client.DeploymentSyncRequest) string {
	return fmt.Sprintf("%s-%s", transformContainerName(Primary(deploymentUnit).Name), deploymentUnit.DeploymentUnitUuid[:8])
}

func transformContainerName(name string) string {
	return strings.ToLower(name)
}

func getLabels(deploymentUnit client.DeploymentSyncRequest) map[string]string {
	return map[string]string{
		labels.RevisionLabel: deploymentUnit.Revision,
		labels.DeploymentUuidLabel: deploymentUnit.DeploymentUnitUuid,
	}
}

func getAnnotations(deploymentUnit client.DeploymentSyncRequest) map[string]string {
	primary := Primary(deploymentUnit)
	annotations := map[string]string{}

	for k, v := range primary.Labels {
		if k != labels.ServiceLaunchConfig {
			annotations[k] = fmt.Sprint(v)
		}
	}
	annotations[getContainerUuidAnnotationName(primary.Name)] = primary.Uuid

	for _, container := range deploymentUnit.Containers {
		if container.Name == primary.Name {
			continue
		}
		for k, v := range container.Labels {
			annotations[fmt.Sprintf("%s/%s", transformContainerName(container.Name), k)] = fmt.Sprint(v)
		}
		annotations[getContainerUuidAnnotationName(container.Name)] = container.Uuid
	}

	return annotations
}

func getContainerUuidAnnotationName(containerName string) string {
	return fmt.Sprintf("%s/%s", transformContainerName(containerName), labels.ContainerUuidLabel)
}

func getPodSpec(deploymentUnit client.DeploymentSyncRequest) v1.PodSpec {
	return v1.PodSpec{
		RestartPolicy: v1.RestartPolicyNever,
		HostNetwork:   getHostNetwork(deploymentUnit),
		HostIPC:       Primary(deploymentUnit).IpcMode == "host",
		HostPID:       Primary(deploymentUnit).PidMode == "host",
		DNSPolicy:     v1.DNSDefault,
		NodeName:      deploymentUnit.NodeName,
		NodeSelector: getNodeSelector(Primary(deploymentUnit)),
		Volumes:      getVolumes(deploymentUnit),
	}
}

func getHostNetwork(deploymentUnit client.DeploymentSyncRequest) bool {
	networkId := Primary(deploymentUnit).PrimaryNetworkId
	for _, network := range deploymentUnit.Networks {
		if network.Id == networkId && network.Kind == hostNetworkingKind {
			return true
		}
	}
	return false
}

func getSecurityContext(container client.Container) *v1.SecurityContext {
	var capAdd []v1.Capability
	for _, cap := range container.CapAdd {
		capAdd = append(capAdd, v1.Capability(cap))
	}
	var capDrop []v1.Capability
	for _, cap := range container.CapDrop {
		capDrop = append(capDrop, v1.Capability(cap))
	}
	return &v1.SecurityContext{
		Privileged:             &container.Privileged,
		ReadOnlyRootFilesystem: &container.ReadOnly,
		Capabilities: &v1.Capabilities{
			Add:  capAdd,
			Drop: capDrop,
		},
	}
}

func getNodeSelector(container client.Container) map[string]string {
	var hostAffinityLabelMap map[string]string
	if label, ok := container.Labels[labels.HostAffinityLabel]; ok {
		hostAffinityLabelMap = labels.Parse(label)
	}
	return hostAffinityLabelMap
}

func getEnvironment(container client.Container) []v1.EnvVar {
	var environment []v1.EnvVar
	for k, v := range container.Environment {
		environment = append(environment, v1.EnvVar{
			Name:  k,
			Value: fmt.Sprint(v),
		})
	}
	return environment
}

func getVolumes(deploymentUnit client.DeploymentSyncRequest) []v1.Volume {
	var volumes []v1.Volume

	for _, container := range deploymentUnit.Containers {
		for _, volume := range container.DataVolumes {
			split := strings.SplitN(volume, ":", -1)
			if len(split) < 2 {
				continue
			}

			hostPath := split[0]

			if !filepath.IsAbs(hostPath) {
				continue
			}

			volumes = append(volumes, v1.Volume{
				Name: getBindMountVolumeName(hostPath),
				VolumeSource: v1.VolumeSource{
					HostPath: &v1.HostPathVolumeSource{
						Path: hostPath,
					},
				},
			})
		}
	}

	for _, container := range deploymentUnit.Containers {
		for _, mount := range container.Mounts {
			volumeName := getVolumeName(mount.VolumeName)
			volumes = append(volumes, v1.Volume{
				Name: volumeName,
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						ClaimName: volumeName,
					},
				},
			})
		}
	}

	for _, container := range deploymentUnit.Containers {
		for tmpfs := range container.Tmpfs {
			volumes = append(volumes, v1.Volume{
				Name: utils.Hash(tmpfs),
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{
						Medium: v1.StorageMediumMemory,
					},
				},
			})
		}
	}

	return volumes
}

func getVolumeMounts(container client.Container) []v1.VolumeMount {
	var volumeMounts []v1.VolumeMount

	for _, volume := range container.DataVolumes {
		split := strings.SplitN(volume, ":", -1)
		if len(split) < 2 {
			continue
		}

		hostPath := split[0]
		containerPath := split[1]

		if !filepath.IsAbs(hostPath) {
			continue
		}

		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      getBindMountVolumeName(hostPath),
			MountPath: containerPath,
		})
	}

	for _, mount := range container.Mounts {
		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      getVolumeName(mount.VolumeName),
			MountPath: mount.Path,
		})
	}

	for tmpfs := range container.Tmpfs {
		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      utils.Hash(tmpfs),
			MountPath: tmpfs,
		})
	}

	return volumeMounts
}

func getBindMountVolumeName(path string) string {
	return utils.Hash(path)
}
