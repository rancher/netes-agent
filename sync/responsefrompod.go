package sync

import (
	"github.com/rancher/go-rancher/v3"
	"k8s.io/client-go/pkg/api/v1"
	"strings"
)

const (
	dockerContainerIdPrefix = "docker://"
)

func responseFromPod(pod v1.Pod) client.DeploymentSyncResponse {
	var instanceStatuses []client.InstanceStatus
	for _, containerStatus := range pod.Status.ContainerStatuses {
		// Don't report back on Rancher pause container
		if containerStatus.Name == rancherPauseContainerName {
			continue
		}

		var containerUuid string
		if len(containerStatus.Name) > 36 {
			containerUuid = containerStatus.Name[len(containerStatus.Name)-36:]
		}

		instanceStatus := client.InstanceStatus{
			ExternalId:   strings.Replace(containerStatus.ContainerID, dockerContainerIdPrefix, "", -1),
			InstanceUuid: containerUuid,
		}
		if !pod.Spec.HostNetwork {
			instanceStatus.PrimaryIpAddress = pod.Status.PodIP
		}
		instanceStatuses = append(instanceStatuses, instanceStatus)
	}

	return client.DeploymentSyncResponse{
		ExternalId:     pod.Name,
		NodeName:       pod.Spec.NodeName,
		InstanceStatus: instanceStatuses,
	}
}
