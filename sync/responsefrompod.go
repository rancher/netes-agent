package sync

import (
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/netes-agent/labels"
	"k8s.io/client-go/pkg/api/v1"
)

func responseFromPod(pod v1.Pod) client.DeploymentSyncResponse {
	var instanceStatuses []client.InstanceStatus
	for _, containerStatus := range pod.Status.ContainerStatuses {
		// Don't report back on Rancher pause container
		if containerStatus.Name == rancherPauseContainerName {
			continue
		}

		annotationName := getAnnotationName(containerStatus.Name, labels.ContainerUuidLabel, false)
		containerUuid, ok := pod.Annotations[annotationName]
		if !ok {
			continue
		}

		instanceStatuses = append(instanceStatuses, client.InstanceStatus{
			ExternalId:       containerStatus.ContainerID,
			InstanceUuid:     containerUuid,
			PrimaryIpAddress: pod.Status.PodIP,
		})
	}

	return client.DeploymentSyncResponse{
		ExternalId:     pod.Name,
		NodeName:       pod.Spec.NodeName,
		InstanceStatus: instanceStatuses,
	}
}
