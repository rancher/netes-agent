package sync

import (
	"fmt"
	"strings"

	"k8s.io/client-go/pkg/api/v1"

	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/kattle/types"
)

func PodFromDeploymentUnit(deploymentUnit types.DeploymentUnit) v1.Pod {
	var containers []v1.Container
	for _, container := range deploymentUnit.Containers {
		containers = append(containers, v1.Container{
			Name:            container.Uuid,
			Image:           getImage(container),
			Command:         container.EntryPoint,
			Args:            container.Command,
			Env:             getEnvironment(container),
			SecurityContext: getSecurityContext(container),
			VolumeMounts:    getVolumeMounts(container),
		})
	}

	primaryConfig := deploymentUnit.RevisionConfig.LaunchConfig
	podSpec := getPodSpec(deploymentUnit, *primaryConfig)
	podSpec.Containers = containers

	return v1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name: deploymentUnit.Uuid,
			Labels: map[string]string{
				revisionLabel: deploymentUnit.RevisionId,
			},
		},
		Spec: podSpec,
	}
}

func getPodSpec(deploymentUnit types.DeploymentUnit, config client.LaunchConfig) v1.PodSpec {
	return v1.PodSpec{
		HostIPC:      config.IpcMode == "host",
		HostNetwork:  config.NetworkMode == "host",
		HostPID:      config.PidMode == "host",
		DNSPolicy:    v1.DNSDefault,
		NodeName:     deploymentUnit.Host.Name,
		NodeSelector: getNodeSelector(config),
		Volumes:      getVolumes(deploymentUnit),
	}
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

func getNodeSelector(config client.LaunchConfig) map[string]string {
	var hostAffinityLabelMap map[string]string
	if label, ok := config.Labels[hostAffinityLabel]; ok {
		hostAffinityLabelMap = ParseLabel(label)
	}
	return hostAffinityLabelMap
}

func getImage(container client.Container) string {
	split := strings.SplitN(container.ImageUuid, ":", 2)
	if len(split) > 1 {
		return split[1]
	}
	return ""
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

func getVolumes(deploymentUnit types.DeploymentUnit) []v1.Volume {
	var volumes []v1.Volume
	for _, container := range deploymentUnit.Containers {
		for _, volume := range container.DataVolumes {
			split := strings.SplitN(volume, ":", -1)
			if len(split) < 2 {
				continue
			}

			hostPath := split[0]

			volumes = append(volumes, v1.Volume{
				Name: createVolumeName(hostPath),
				VolumeSource: v1.VolumeSource{
					HostPath: &v1.HostPathVolumeSource{
						Path: hostPath,
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

		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      createVolumeName(hostPath),
			MountPath: containerPath,
		})
	}
	return volumeMounts
}

func createVolumeName(path string) string {
	path = strings.TrimLeft(path, "/")
	path = strings.Replace(path, "/", "-", -1)
	return fmt.Sprintf("%s-%s", path, "volume")
}
