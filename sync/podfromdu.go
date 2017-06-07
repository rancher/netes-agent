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
		image := strings.SplitN(container.ImageUuid, ":", 2)[1]
		var environment []v1.EnvVar
		for k, v := range container.Environment {
			environment = append(environment, v1.EnvVar{
				Name:  k,
				Value: fmt.Sprint(v),
			})
		}
		containers = append(containers, v1.Container{
			Name:  container.Uuid,
			Image: image,
			Env:   environment,
			SecurityContext: &v1.SecurityContext{
				Privileged: &container.Privileged,
			},
			VolumeMounts: getVolumeMounts(container),
			// TODO: entrypoint, commands
		})
	}

	primaryConfig := deploymentUnit.RevisionConfig.LaunchConfig

	var hostAffinityLabelMap map[string]string
	if label, ok := primaryConfig.Labels[hostAffinityLabel]; ok {
		hostAffinityLabelMap = ParseLabel(label)
	}

	return v1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name: deploymentUnit.Uuid,
			Labels: map[string]string{
				"io.rancher.kattle":   "true",
				"io.rancher.revision": deploymentUnit.RevisionId,
			},
		},
		Spec: v1.PodSpec{
			Containers:  containers,
			HostIPC:     primaryConfig.IpcMode == "host",
			HostNetwork: primaryConfig.NetworkMode == "host",
			HostPID:     primaryConfig.PidMode == "host",
			// Rancher DNS
			DNSPolicy: v1.DNSDefault,
			// Handle global service case
			NodeName: deploymentUnit.Host.Name,
			// TODO: all types of affinity
			NodeSelector: hostAffinityLabelMap,
			Volumes:      getVolumes(deploymentUnit),
		},
	}
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
