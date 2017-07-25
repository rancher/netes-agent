package sync

import (
	"fmt"
	"path/filepath"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/pkg/api/v1"

	"github.com/rancher/go-rancher-metadata/metadata"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancherlabs/kattle/labels"
)

func PodFromDeploymentUnit(deploymentUnit metadata.DeploymentUnit) v1.Pod {
	var containers []v1.Container
	for _, container := range deploymentUnit.Containers {
		containers = append(containers, getContainer(container))
	}

	primaryConfig := deploymentUnit.Revision.Config.LaunchConfig
	podSpec := getPodSpec(deploymentUnit, *primaryConfig)
	podSpec.Containers = containers

	return v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: deploymentUnit.Uuid,
			Labels: map[string]string{
				labels.RevisionLabel: deploymentUnit.RevisionId,
			},
		},
		Spec: podSpec,
	}
}

func getContainer(container client.Container) v1.Container {
	return v1.Container{
		Name:            container.Uuid,
		Image:           getImage(container),
		Command:         container.EntryPoint,
		Args:            container.Command,
		TTY:             container.Tty,
		Stdin:           container.StdinOpen,
		WorkingDir:      container.WorkingDir,
		Env:             getEnvironment(container),
		SecurityContext: getSecurityContext(container),
		VolumeMounts:    getVolumeMounts(container),
		LivenessProbe:   getLivenessProbe(container),
	}
}

func getPodSpec(deploymentUnit metadata.DeploymentUnit, config client.LaunchConfig) v1.PodSpec {
	return v1.PodSpec{
		RestartPolicy: v1.RestartPolicyNever,
		HostIPC:       config.IpcMode == "host",
		HostNetwork:   config.NetworkMode == "host",
		HostPID:       config.PidMode == "host",
		DNSPolicy:     v1.DNSDefault,
		NodeName:      deploymentUnit.Host.Name,
		NodeSelector:  getNodeSelector(config),
		Volumes:       getVolumes(deploymentUnit),
	}
}

func getLivenessProbe(container client.Container) *v1.Probe {
	healthcheck := container.HealthCheck
	if healthcheck == nil {
		return nil
	}

	timeoutSeconds := int32(healthcheck.ResponseTimeout / 1000)
	if timeoutSeconds == 0 {
		timeoutSeconds = 1
	}

	periodSeconds := int32(healthcheck.Interval / 1000)
	if periodSeconds == 0 {
		periodSeconds = 1
	}

	probe := v1.Probe{
		TimeoutSeconds:   timeoutSeconds,
		PeriodSeconds:    periodSeconds,
		SuccessThreshold: int32(healthcheck.HealthyThreshold),
		FailureThreshold: int32(healthcheck.UnhealthyThreshold),
	}
	port := intstr.IntOrString{
		IntVal: int32(healthcheck.Port),
	}

	if healthcheck.RequestLine == "" {
		probe.Handler = v1.Handler{
			TCPSocket: &v1.TCPSocketAction{
				Port: port,
			},
		}
	} else {
		parts := strings.Split(healthcheck.RequestLine, " ")
		if len(parts) < 3 {
			return nil
		}
		// TODO: first and third parts completely ignored
		path := parts[1]
		probe.Handler = v1.Handler{
			HTTPGet: &v1.HTTPGetAction{
				Path: path,
				Port: port,
			},
		}
	}

	return &probe
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
	if label, ok := config.Labels[labels.HostAffinityLabel]; ok {
		hostAffinityLabelMap = labels.Parse(label)
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

func getVolumes(deploymentUnit metadata.DeploymentUnit) []v1.Volume {
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
				Name: createBindMountVolumeName(hostPath),
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
			Name:      createBindMountVolumeName(hostPath),
			MountPath: containerPath,
		})
	}

	for _, mount := range container.Mounts {
		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      getVolumeName(mount.VolumeName),
			MountPath: mount.Path,
		})
	}

	return volumeMounts
}

func createBindMountVolumeName(path string) string {
	path = strings.TrimLeft(path, "/")
	path = strings.Replace(path, "/", "-", -1)
	return fmt.Sprintf("%s-%s", path, "volume")
}
