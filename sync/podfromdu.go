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
	hostNetworkingKind        = "host"
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
			Name:        getPodName(deploymentUnit),
			Namespace:   deploymentUnit.Namespace,
			Labels:      getLabels(deploymentUnit),
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
		labels.RevisionLabel:       deploymentUnit.Revision,
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
		Affinity:      getAffinity(Primary(deploymentUnit)),
		HostAliases:   getHostAliases(Primary(deploymentUnit)),
		Volumes:       getVolumes(deploymentUnit),
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

func getHostAliases(container client.Container) []v1.HostAlias {
	var hostAliases []v1.HostAlias
	for _, extraHost := range container.ExtraHosts {
		parts := strings.SplitN(extraHost, ":", 2)
		if len(parts) < 2 {
			continue
		}
		hostAliases = append(hostAliases, v1.HostAlias{
			IP: parts[1],
			Hostnames: []string{
				parts[0],
			},
		})
	}
	return hostAliases
}

func getAffinity(container client.Container) *v1.Affinity {
	// No affinity for global services
	if val, ok := container.Labels[labels.GlobalLabel]; ok && val == "true" {
		return nil
	}

	var matchExpressions []v1.NodeSelectorRequirement
	hostAffinity, ok := container.Labels[labels.HostAffinityLabel]
	if ok {
		affinitySelectors := getMatchExpressions(labels.Parse(hostAffinity), v1.NodeSelectorOpIn)
		matchExpressions = append(matchExpressions, affinitySelectors...)
	}
	hostAntiAffinity, ok := container.Labels[labels.HostAntiAffinityLabel]
	if ok {
		antiAffinitySelectors := getMatchExpressions(labels.Parse(hostAntiAffinity), v1.NodeSelectorOpNotIn)
		matchExpressions = append(matchExpressions, antiAffinitySelectors...)
	}

	var softMatchExpressions []v1.NodeSelectorRequirement
	softHostAffinity, ok := container.Labels[labels.HostSoftAffinityLabel]
	if ok {
		softAffinitySelectors := getMatchExpressions(labels.Parse(softHostAffinity), v1.NodeSelectorOpIn)
		softMatchExpressions = append(softMatchExpressions, softAffinitySelectors...)
	}
	softHostAntiAffinity, ok := container.Labels[labels.HostSoftAntiAffinityLabel]
	if ok {
		softAntiAffinitySelectors := getMatchExpressions(labels.Parse(softHostAntiAffinity), v1.NodeSelectorOpNotIn)
		softMatchExpressions = append(softMatchExpressions, softAntiAffinitySelectors...)
	}

	if len(matchExpressions) == 0 && len(softMatchExpressions) == 0 {
		return nil
	}

	affinity := v1.Affinity{
		NodeAffinity: &v1.NodeAffinity{},
	}
	if len(matchExpressions) > 0 {
		affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &v1.NodeSelector{
			NodeSelectorTerms: []v1.NodeSelectorTerm{
				{
					MatchExpressions: matchExpressions,
				},
			},
		}
	}
	if len(softMatchExpressions) > 0 {
		affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = []v1.PreferredSchedulingTerm{
			{
				Preference: v1.NodeSelectorTerm{
					MatchExpressions: softMatchExpressions,
				},
			},
		}
	}
	return &affinity
}

func getMatchExpressions(labelMap map[string]string, operator v1.NodeSelectorOperator) []v1.NodeSelectorRequirement {
	var matchExpressions []v1.NodeSelectorRequirement
	for k, v := range labelMap {
		matchExpressions = append(matchExpressions, v1.NodeSelectorRequirement{
			Key:      k,
			Operator: operator,
			Values: []string{
				v,
			},
		})
	}
	return matchExpressions
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
				Name: utils.Hash(hostPath),
				VolumeSource: v1.VolumeSource{
					HostPath: &v1.HostPathVolumeSource{
						Path: hostPath,
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
			Name:      utils.Hash(hostPath),
			MountPath: containerPath,
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
